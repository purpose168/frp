// 版权所有 2023 frp 作者
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的管理权限和
// 限制，请参阅许可证。

//go:build !frps

package proxy

import (
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
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.UDPProxyConfig{}), NewUDPProxy)
}

// UDPProxy UDP 代理结构
type UDPProxy struct {
	*BaseProxy

	cfg *v1.UDPProxyConfig

	localAddr *net.UDPAddr
	readCh    chan *msg.UDPPacket

	// 包含 msg.UDPPacket 和 msg.Ping
	sendCh   chan msg.Message
	workConn net.Conn
	closed   bool
}

// NewUDPProxy 创建新的 UDP 代理实例
func NewUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.UDPProxyConfig)
	if !ok {
		return nil
	}
	return &UDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// Run 运行 UDP 代理，解析本地 UDP 地址
func (pxy *UDPProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.cfg.LocalIP, strconv.Itoa(pxy.cfg.LocalPort)))
	if err != nil {
		return
	}
	return
}

// Close 关闭 UDP 代理并释放资源
func (pxy *UDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()

	if !pxy.closed {
		pxy.closed = true
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		if pxy.readCh != nil {
			close(pxy.readCh)
		}
		if pxy.sendCh != nil {
			close(pxy.sendCh)
		}
	}
}

// InWorkConn 处理工作连接
func (pxy *UDPProxy) InWorkConn(conn net.Conn, _ *msg.StartWorkConn) {
	xl := pxy.xl
	xl.Infof("接收到 udp 代理的新工作连接，%s", conn.RemoteAddr().String())
	// 关闭与旧工作连接相关的资源
	pxy.Close()

	var rwc io.ReadWriteCloser = conn
	var err error
	if pxy.limiter != nil {
		rwc = libio.WrapReadWriteCloser(limit.NewReader(conn, pxy.limiter), limit.NewWriter(conn, pxy.limiter), func() error {
			return conn.Close()
		})
	}
	if pxy.cfg.Transport.UseEncryption {
		rwc, err = libio.WithEncryption(rwc, pxy.encryptionKey)
		if err != nil {
			conn.Close()
			xl.Errorf("创建加密流错误: %v", err)
			return
		}
	}
	if pxy.cfg.Transport.UseCompression {
		rwc = libio.WithCompression(rwc)
	}
	conn = netpkg.WrapReadWriteCloserToConn(rwc, conn)

	pxy.mu.Lock()
	pxy.workConn = conn
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.sendCh = make(chan msg.Message, 1024)
	pxy.closed = false
	pxy.mu.Unlock()

	workConnReaderFn := func(conn net.Conn, readCh chan *msg.UDPPacket) {
		for {
			var udpMsg msg.UDPPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				xl.Warnf("从 udp 的工作连接读取错误: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				xl.Tracef("从工作连接获取 UDP 数据包: %s", udpMsg.Content)
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Infof("udp 工作连接的读取器协程已关闭: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(conn net.Conn, sendCh chan msg.Message) {
		defer func() {
			xl.Infof("udp 工作连接的写入器协程已关闭")
		}()
		var errRet error
		for rawMsg := range sendCh {
			switch m := rawMsg.(type) {
			case *msg.UDPPacket:
				xl.Tracef("发送 UDP 数据包到工作连接: %s", m.Content)
			case *msg.Ping:
				xl.Tracef("发送 ping 消息到 udp 工作连接")
			}
			if errRet = msg.WriteMsg(conn, rawMsg); errRet != nil {
				xl.Errorf("udp 工作连接写入错误: %v", errRet)
				return
			}
		}
	}
	heartbeatFn := func(sendCh chan msg.Message) {
		var errRet error
		for {
			time.Sleep(time.Duration(30) * time.Second)
			if errRet = errors.PanicToError(func() {
				sendCh <- &msg.Ping{}
			}); errRet != nil {
				xl.Tracef("udp 工作连接的心跳协程已关闭")
				break
			}
		}
	}

	go workConnSenderFn(pxy.workConn, pxy.sendCh)
	go workConnReaderFn(pxy.workConn, pxy.readCh)
	go heartbeatFn(pxy.sendCh)

	// 使用代理协议版本调用 Forwarder（空字符串表示不使用代理协议）
	udp.Forwarder(pxy.localAddr, pxy.readCh, pxy.sendCh, int(pxy.clientCfg.UDPPacketSize), pxy.cfg.Transport.ProxyProtocolVersion)
}
