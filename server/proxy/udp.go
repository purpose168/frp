// Copyright 2019 fatedier, fatedier@gmail.com
//
// 依据 Apache License, Version 2.0 许可证授权；
// 除非符合许可证的要求，否则您不能使用此文件。
// 您可以在以下网址获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则软件
// 根据许可证分发是基于“按原样”基础，
// 不附带任何明示或暗示的担保或条件。
// 请参阅许可证中有关管理权限和
// 限制的特定语言。

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/fatedier/golib/errors"
	libio "github.com/fatedier/golib/io"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/proto/udp"
	"github.com/purpose168/frp/pkg/util/limit"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/server/metrics"
)

// init 注册UDP代理工厂
func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.UDPProxyConfig{}), NewUDPProxy)
}

// UDPProxy UDP代理
// 实现了Proxy接口，用于处理UDP代理请求
type UDPProxy struct {
	*BaseProxy
	// cfg UDP代理配置
	cfg *v1.UDPProxyConfig

	// realBindPort 实际绑定的端口号
	realBindPort int

	// udpConn 是UDP包的监听器
	udpConn *net.UDPConn

	// 同一时间始终只有一个workConn
	// 如果它关闭了，获取另一个
	workConn net.Conn

	// sendCh 用于向workConn发送包
	sendCh chan *msg.UDPPacket

	// readCh 用于从workConn读取包
	readCh chan *msg.UDPPacket

	// checkCloseCh 用于监视workConn是否关闭
	checkCloseCh chan int

	isClosed bool
}

// NewUDPProxy 创建一个新的UDP代理
// 参数baseProxy是基础代理
func NewUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.UDPProxyConfig)
	if !ok {
		return nil
	}
	baseProxy.usedPortsNum = 1
	return &UDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// Run 启动UDP代理
// 返回值是远程地址和可能的错误
func (pxy *UDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	pxy.realBindPort, err = pxy.rc.UDPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
	if err != nil {
		return "", fmt.Errorf("获取端口 %d 错误: %v", pxy.cfg.RemotePort, err)
	}
	defer func() {
		if err != nil {
			pxy.rc.UDPPortManager.Release(pxy.realBindPort)
		}
	}()

	remoteAddr = fmt.Sprintf(":%d", pxy.realBindPort)
	pxy.cfg.RemotePort = pxy.realBindPort
	addr, errRet := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realBindPort)))
	if errRet != nil {
		err = errRet
		return
	}
	udpConn, errRet := net.ListenUDP("udp", addr)
	if errRet != nil {
		err = errRet
		xl.Warnf("监听udp端口错误: %v", err)
		return
	}
	xl.Infof("udp代理监听端口 [%d]", pxy.cfg.RemotePort)

	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UDPPacket, 1024)
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.checkCloseCh = make(chan int)

	// workConnReaderFn 从workConn读取消息，如果返回任何错误，通知代理启动一个新的workConn
	workConnReaderFn := func(conn net.Conn) {
		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			xl.Tracef("循环等待来自udp workConn的消息")
			// 客户端将在workConn中发送心跳以保持连接
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(60) * time.Second))
			if rawMsg, errRet = msg.ReadMsg(conn); errRet != nil {
				xl.Warnf("从udp workConn读取错误: %v", errRet)
				_ = conn.Close()
				// 通知代理启动一个新的工作连接
				// 忽略这里的错误，这意味着代理已关闭
				_ = errors.PanicToError(func() {
					pxy.checkCloseCh <- 1
				})
				return
			}
			if err := conn.SetReadDeadline(time.Time{}); err != nil {
				xl.Warnf("设置读取超时错误: %v", err)
			}
			switch m := rawMsg.(type) {
			case *msg.Ping:
				xl.Tracef("udp工作连接获取ping消息")
				continue
			case *msg.UDPPacket:
				if errRet := errors.PanicToError(func() {
					xl.Tracef("从workConn获取udp消息: %s", m.Content)
					pxy.readCh <- m
					metrics.Server.AddTrafficOut(
						pxy.GetName(),
						pxy.GetConfigurer().GetBaseConfig().Type,
						int64(len(m.Content)),
					)
				}); errRet != nil {
					conn.Close()
					xl.Infof("udp工作连接的读取goroutine已关闭")
					return
				}
			}
		}
	}

	// workConnSenderFn 向workConn发送消息
	workConnSenderFn := func(conn net.Conn, ctx context.Context) {
		var errRet error
		for {
			select {
			case udpMsg, ok := <-pxy.sendCh:
				if !ok {
					xl.Infof("udp工作连接的发送goroutine已关闭")
					return
				}
				if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
					xl.Infof("udp工作连接的发送goroutine已关闭: %v", errRet)
					conn.Close()
					return
				}
				xl.Tracef("向udp workConn发送消息: %s", udpMsg.Content)
				metrics.Server.AddTrafficIn(
					pxy.GetName(),
					pxy.GetConfigurer().GetBaseConfig().Type,
					int64(len(udpMsg.Content)),
				)
				continue
			case <-ctx.Done():
				xl.Infof("udp工作连接的发送goroutine已关闭")
				return
			}
		}
	}

	go func() {
		// 等待一段时间，让控制层将NewProxyResp发送给客户端
		time.Sleep(500 * time.Millisecond)
		for {
			workConn, err := pxy.GetWorkConnFromPool(nil, nil)
			if err != nil {
				time.Sleep(1 * time.Second)
				// 检查代理是否关闭
				select {
				case _, ok := <-pxy.checkCloseCh:
					if !ok {
						return
					}
				default:
				}
				continue
			}
			// 关闭旧的workConn，并用新的替换它
			if pxy.workConn != nil {
				pxy.workConn.Close()
			}

			var rwc io.ReadWriteCloser = workConn
			if pxy.cfg.Transport.UseEncryption {
				rwc, err = libio.WithEncryption(rwc, pxy.encryptionKey)
				if err != nil {
					xl.Errorf("创建加密流错误: %v", err)
					workConn.Close()
					continue
				}
			}
			if pxy.cfg.Transport.UseCompression {
				rwc = libio.WithCompression(rwc)
			}

			if pxy.GetLimiter() != nil {
				rwc = libio.WrapReadWriteCloser(limit.NewReader(rwc, pxy.GetLimiter()), limit.NewWriter(rwc, pxy.GetLimiter()), func() error {
					return rwc.Close()
				})
			}

			pxy.workConn = netpkg.WrapReadWriteCloserToConn(rwc, workConn)
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(pxy.workConn)
			go workConnSenderFn(pxy.workConn, ctx)
			_, ok := <-pxy.checkCloseCh
			cancel()
			if !ok {
				return
			}
		}
	}()

	// 从用户连接读取数据，并将包装后的udp消息发送到sendCh（由workConn转发）。
	// 客户端将udp消息转换为本地udp服务，并等待一段时间的响应。
	// 响应将被包装，通过工作连接转发到服务器。
	// 最后关闭readCh和sendCh。
	go func() {
		udp.ForwardUserConn(udpConn, pxy.readCh, pxy.sendCh, int(pxy.serverCfg.UDPPacketSize))
		pxy.Close()
	}()
	return remoteAddr, nil
}

// Close 关闭UDP代理
func (pxy *UDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	if !pxy.isClosed {
		pxy.isClosed = true

		pxy.BaseProxy.Close()
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		pxy.udpConn.Close()

		// 所有通道只在这里关闭
		close(pxy.checkCloseCh)
		close(pxy.readCh)
		close(pxy.sendCh)
	}
	pxy.rc.UDPPortManager.Release(pxy.realBindPort)
}
