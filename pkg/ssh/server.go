// Copyright 2023 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"

	"github.com/purpose168/frp/client/proxy"
	"github.com/purpose168/frp/pkg/config"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/util/log"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/util"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/pkg/virtual"
)

const (
	// https://datatracker.ietf.org/doc/html/rfc4254#page-16
	ChannelTypeServerOpenChannel = "forwarded-tcpip"
	RequestTypeForward           = "tcpip-forward"
)

// tcpipForward 表示 TCP/IP 转发请求
type tcpipForward struct {
	Host string
	Port uint32
}

// https://datatracker.ietf.org/doc/html/rfc4254#page-16
// forwardedTCPPayload 表示转发 TCP 负载
type forwardedTCPPayload struct {
	Addr string
	Port uint32

	OriginAddr string
	OriginPort uint32
}

// TunnelServer 是 SSH 隧道服务器
type TunnelServer struct {
	// underlyingConn 是底层连接
	underlyingConn net.Conn
	// sshConn 是 SSH 服务器连接
	sshConn *ssh.ServerConn
	// sc 是 SSH 服务器配置
	sc *ssh.ServerConfig
	// firstChannel 是第一个 SSH 通道
	firstChannel ssh.Channel

	// vc 是虚拟客户端
	vc *virtual.Client
	// peerServerListener 是对端服务器监听器
	peerServerListener *netpkg.InternalListener
	// doneCh 是完成通道
	doneCh chan struct{}
	// closeDoneChOnce 是关闭完成通道的同步对象
	closeDoneChOnce sync.Once
}

// NewTunnelServer 创建一个新的 SSH 隧道服务器
// 参数 conn 是网络连接
// 参数 sc 是 SSH 服务器配置
// 参数 peerServerListener 是对端服务器监听器
// 返回隧道服务器实例和可能的错误
func NewTunnelServer(conn net.Conn, sc *ssh.ServerConfig, peerServerListener *netpkg.InternalListener) (*TunnelServer, error) {
	s := &TunnelServer{
		underlyingConn:     conn,
		sc:                 sc,
		peerServerListener: peerServerListener,
		doneCh:             make(chan struct{}),
	}
	return s, nil
}

// Run 运行隧道服务器
func (s *TunnelServer) Run() error {
	sshConn, channels, requests, err := ssh.NewServerConn(s.underlyingConn, s.sc)
	if err != nil {
		return err
	}

	s.sshConn = sshConn

	addr, extraPayload, err := s.waitForwardAddrAndExtraPayload(channels, requests, 3*time.Second)
	if err != nil {
		return err
	}

	clientCfg, pc, helpMessage, err := s.parseClientAndProxyConfigurer(addr, extraPayload)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			s.writeToClient(helpMessage)
			return nil
		}
		s.writeToClient(err.Error())
		return fmt.Errorf("从 SSH 客户端解析标志错误：%v", err)
	}
	if err := clientCfg.Complete(); err != nil {
		s.writeToClient(fmt.Sprintf("完成客户端配置失败：%v", err))
		return fmt.Errorf("完成客户端配置错误：%v", err)
	}
	if sshConn.Permissions != nil {
		clientCfg.User = util.EmptyOr(sshConn.Permissions.Extensions["user"], clientCfg.User)
	}
	pc.Complete(clientCfg.User)

	vc, err := virtual.NewClient(virtual.ClientOptions{
		Common: clientCfg,
		Spec: &msg.ClientSpec{
			Type: "ssh-tunnel",
			// 如果 SSH 不需要认证，则虚拟客户端需要通过令牌进行认证。
			// 否则，一旦 SSH 认证通过，虚拟客户端就不需要再次认证。
			AlwaysAuthPass: !s.sc.NoClientAuth,
		},
		HandleWorkConnCb: func(base *v1.ProxyBaseConfig, workConn net.Conn, m *msg.StartWorkConn) bool {
			// 连接工作连接和 SSH 通道
			c, err := s.openConn(addr)
			if err != nil {
				log.Tracef("打开连接错误：%v", err)
				workConn.Close()
				return false
			}
			libio.Join(c, workConn)
			return false
		},
	})
	if err != nil {
		return err
	}
	s.vc = vc

	// 将连接从虚拟客户端传输到服务器对端监听器
	go func() {
		l := s.vc.PeerListener()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_ = s.peerServerListener.PutConn(conn)
		}
	}()
	xl := xlog.New().AddPrefix(xlog.LogPrefix{Name: "sshVirtualClient", Value: "sshVirtualClient", Priority: 100})
	ctx := xlog.NewContext(context.Background(), xl)
	go func() {
		vcErr := s.vc.Run(ctx)
		if vcErr != nil {
			s.writeToClient(vcErr.Error())
		}

		// 如果 vc.Run 返回，意味着虚拟客户端已关闭，SSH 隧道连接也应该关闭。
		// 一种情况是虚拟客户端因登录失败而退出。
		s.closeDoneChOnce.Do(func() {
			_ = sshConn.Close()
			close(s.doneCh)
		})
	}()

	s.vc.UpdateProxyConfigurer([]v1.ProxyConfigurer{pc})

	if ps, err := s.waitProxyStatusReady(pc.GetBaseConfig().Name, time.Second); err != nil {
		s.writeToClient(err.Error())
		log.Warnf("等待代理状态就绪错误：%v", err)
	} else {
		// 成功
		s.writeToClient(createSuccessInfo(clientCfg.User, pc, ps))
		_ = sshConn.Wait()
	}

	s.vc.Close()
	log.Tracef("来自 %v 的 SSH 隧道连接已关闭", sshConn.RemoteAddr())
	s.closeDoneChOnce.Do(func() {
		_ = sshConn.Close()
		close(s.doneCh)
	})
	return nil
}

// writeToClient 向客户端写入数据
func (s *TunnelServer) writeToClient(data string) {
	if s.firstChannel == nil {
		return
	}
	_, _ = s.firstChannel.Write([]byte(data + "\n"))
}

// waitForwardAddrAndExtraPayload 等待转发地址和额外负载
func (s *TunnelServer) waitForwardAddrAndExtraPayload(
	channels <-chan ssh.NewChannel,
	requests <-chan *ssh.Request,
	timeout time.Duration,
) (*tcpipForward, string, error) {
	addrCh := make(chan *tcpipForward, 1)
	extraPayloadCh := make(chan string, 1)

	// 获取转发地址
	go func() {
		addrGot := false
		for req := range requests {
			if req.Type == RequestTypeForward && !addrGot {
				payload := tcpipForward{}
				if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
					return
				}
				addrGot = true
				addrCh <- &payload
			}
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		}
	}()

	// 获取额外负载
	go func() {
		for newChannel := range channels {
			// extraPayload 将发送到 extraPayloadCh
			go s.handleNewChannel(newChannel, extraPayloadCh)
		}
	}()

	var (
		addr         *tcpipForward
		extraPayload string
	)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case v := <-addrCh:
			addr = v
		case extra := <-extraPayloadCh:
			extraPayload = extra
		case <-timer.C:
			return nil, "", fmt.Errorf("获取地址和额外负载超时")
		}
		if addr != nil && extraPayload != "" {
			break
		}
	}
	return addr, extraPayload, nil
}

// parseClientAndProxyConfigurer 解析客户端和代理配置器
func (s *TunnelServer) parseClientAndProxyConfigurer(_ *tcpipForward, extraPayload string) (*v1.ClientCommonConfig, v1.ProxyConfigurer, string, error) {
	helpMessage := ""
	cmd := &cobra.Command{
		Use:   "ssh v0@{address} [command]",
		Short: "ssh v0@{address} [command]",
		Run:   func(*cobra.Command, []string) {},
	}
	cmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)

	args := strings.Split(extraPayload, " ")
	if len(args) < 1 {
		return nil, nil, helpMessage, fmt.Errorf("无效的额外负载")
	}
	proxyType := strings.TrimSpace(args[0])
	supportTypes := []string{"tcp", "http", "https", "tcpmux", "stcp"}
	if !slices.Contains(supportTypes, proxyType) {
		return nil, nil, helpMessage, fmt.Errorf("无效的代理类型：%s，支持的类型：%v", proxyType, supportTypes)
	}
	pc := v1.NewProxyConfigurerByType(v1.ProxyType(proxyType))
	if pc == nil {
		return nil, nil, helpMessage, fmt.Errorf("新建代理配置器错误")
	}
	config.RegisterProxyFlags(cmd, pc, config.WithSSHMode())

	clientCfg := v1.ClientCommonConfig{}
	config.RegisterClientCommonConfigFlags(cmd, &clientCfg, config.WithSSHMode())

	cmd.InitDefaultHelpCmd()
	if err := cmd.ParseFlags(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			helpMessage = cmd.UsageString()
		}
		return nil, nil, helpMessage, err
	}
	// 如果名称未设置，则生成一个随机名称
	if pc.GetBaseConfig().Name == "" {
		id, err := util.RandIDWithLen(8)
		if err != nil {
			return nil, nil, helpMessage, fmt.Errorf("生成随机 ID 错误：%v", err)
		}
		pc.GetBaseConfig().Name = fmt.Sprintf("sshtunnel-%s-%s", proxyType, id)
	}
	return &clientCfg, pc, helpMessage, nil
}

// handleNewChannel 处理新通道
func (s *TunnelServer) handleNewChannel(channel ssh.NewChannel, extraPayloadCh chan string) {
	ch, reqs, err := channel.Accept()
	if err != nil {
		return
	}
	if s.firstChannel == nil {
		s.firstChannel = ch
	}
	go s.keepAlive(ch)

	for req := range reqs {
		if req.WantReply {
			_ = req.Reply(true, nil)
		}
		if req.Type != "exec" || len(req.Payload) <= 4 {
			continue
		}
		end := 4 + binary.BigEndian.Uint32(req.Payload[:4])
		if len(req.Payload) < int(end) {
			continue
		}
		extraPayload := string(req.Payload[4:end])
		select {
		case extraPayloadCh <- extraPayload:
		default:
		}
	}
}

// keepAlive 保持连接活跃
func (s *TunnelServer) keepAlive(ch ssh.Channel) {
	tk := time.NewTicker(time.Second * 30)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			_, err := ch.SendRequest("heartbeat", false, nil)
			if err != nil {
				return
			}
		case <-s.doneCh:
			return
		}
	}
}

// openConn 打开连接
func (s *TunnelServer) openConn(addr *tcpipForward) (net.Conn, error) {
	payload := forwardedTCPPayload{
		Addr: addr.Host,
		Port: addr.Port,
		// 注意：这里只是为了兼容性，不是真实的源地址。
		OriginAddr: addr.Host,
		OriginPort: addr.Port,
	}
	channel, reqs, err := s.sshConn.OpenChannel(ChannelTypeServerOpenChannel, ssh.Marshal(&payload))
	if err != nil {
		return nil, fmt.Errorf("打开 SSH 通道错误：%v", err)
	}
	go ssh.DiscardRequests(reqs)

	conn := netpkg.WrapReadWriteCloserToConn(channel, s.underlyingConn)
	return conn, nil
}

// waitProxyStatusReady 等待代理状态就绪
func (s *TunnelServer) waitProxyStatusReady(name string, timeout time.Duration) (*proxy.WorkingStatus, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	statusExporter := s.vc.Service().StatusExporter()

	for {
		select {
		case <-ticker.C:
			ps, ok := statusExporter.GetProxyStatus(name)
			if !ok {
				continue
			}
			switch ps.Phase {
			case proxy.ProxyPhaseRunning:
				return ps, nil
			case proxy.ProxyPhaseStartErr, proxy.ProxyPhaseClosed:
				return ps, errors.New(ps.Err)
			}
		case <-timer.C:
			return nil, fmt.Errorf("等待代理状态就绪超时")
		case <-s.doneCh:
			return nil, fmt.Errorf("SSH 隧道服务器已关闭")
		}
	}
}
