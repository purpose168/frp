// 版权所有 2017 fatedier, fatedier@gmail.com
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"基础分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的管理权限和
// 限制，请参阅许可证。

package client

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/purpose168/frp/client/proxy"
	"github.com/purpose168/frp/client/visitor"
	"github.com/purpose168/frp/pkg/auth"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/transport"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/wait"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/pkg/vnet"
)

// SessionContext 会话上下文，包含客户端与服务端连接的相关信息
type SessionContext struct {
	// 客户端通用配置
	Common *v1.ClientCommonConfig

	// 从 frps 获取的唯一标识符
	// 重连时应将其附加到登录消息中
	RunID string
	// 底层控制连接，一旦 conn 关闭，msgDispatcher 和整个 Control 将退出
	Conn net.Conn
	// 指示连接是否已加密
	ConnEncrypted bool
	// 用于登录、心跳和加密的身份验证运行时
	Auth *auth.ClientAuth
	// 连接器用于创建新连接，可以是真实的 TCP 连接或虚拟流
	Connector Connector
	// 虚拟网络控制器
	VnetController *vnet.Controller
}

// Control 控制器，负责管理客户端与服务端的连接、代理和访问者
type Control struct {
	// 服务上下文
	ctx context.Context
	xl  *xlog.Logger

	// 会话上下文
	sessionCtx *SessionContext

	// 管理所有代理
	pm *proxy.Manager

	// 管理所有访问者
	vm *visitor.Manager

	doneCh chan struct{}

	// 最后一次收到 Pong 消息的时间
	lastPong atomic.Value

	// msgTransporter 的作用类似于 HTTP2
	// 它允许在同一个控制连接上同时发送多条消息
	// 服务端的响应消息将根据 laneKey 和消息类型分发到相应的等待协程
	msgTransporter transport.MessageTransporter

	// msgDispatcher 是控制连接的包装器
	// 它提供发送消息的通道，您可以注册处理器根据各自的消息类型处理消息
	msgDispatcher *msg.Dispatcher
}

// NewControl 创建新的控制器实例
func NewControl(ctx context.Context, sessionCtx *SessionContext) (*Control, error) {
	// 创建新的 xlog 实例
	ctl := &Control{
		ctx:        ctx,
		xl:         xlog.FromContextSafe(ctx),
		sessionCtx: sessionCtx,
		doneCh:     make(chan struct{}),
	}
	ctl.lastPong.Store(time.Now())

	if sessionCtx.ConnEncrypted {
		cryptoRW, err := netpkg.NewCryptoReadWriter(sessionCtx.Conn, sessionCtx.Auth.EncryptionKey())
		if err != nil {
			return nil, err
		}
		ctl.msgDispatcher = msg.NewDispatcher(cryptoRW)
	} else {
		ctl.msgDispatcher = msg.NewDispatcher(sessionCtx.Conn)
	}
	ctl.registerMsgHandlers()
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher)

	ctl.pm = proxy.NewManager(ctl.ctx, sessionCtx.Common, sessionCtx.Auth.EncryptionKey(), ctl.msgTransporter, sessionCtx.VnetController)
	ctl.vm = visitor.NewManager(ctl.ctx, sessionCtx.RunID, sessionCtx.Common,
		ctl.connectServer, ctl.msgTransporter, sessionCtx.VnetController)
	return ctl, nil
}

// Run 启动控制器，开始处理代理和访问者
func (ctl *Control) Run(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) {
	go ctl.worker()

	// 启动所有代理
	ctl.pm.UpdateAll(proxyCfgs)

	// 启动所有访问者
	ctl.vm.UpdateAll(visitorCfgs)
}

// SetInWorkConnCallback 设置工作连接的回调函数
func (ctl *Control) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	ctl.pm.SetInWorkConnCallback(cb)
}

// handleReqWorkConn 处理工作连接请求
func (ctl *Control) handleReqWorkConn(_ msg.Message) {
	xl := ctl.xl
	workConn, err := ctl.connectServer()
	if err != nil {
		xl.Warnf("建立与服务端的新连接失败: %v", err)
		return
	}

	m := &msg.NewWorkConn{
		RunID: ctl.sessionCtx.RunID,
	}
	if err = ctl.sessionCtx.Auth.Setter.SetNewWorkConn(m); err != nil {
		xl.Warnf("NewWorkConn 身份验证期间出错: %v", err)
		workConn.Close()
		return
	}
	if err = msg.WriteMsg(workConn, m); err != nil {
		xl.Warnf("工作连接写入服务端失败: %v", err)
		workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = msg.ReadMsgInto(workConn, &startMsg); err != nil {
		xl.Tracef("工作连接在响应 StartWorkConn 消息之前关闭: %v", err)
		workConn.Close()
		return
	}
	if startMsg.Error != "" {
		xl.Errorf("StartWorkConn 包含错误: %s", startMsg.Error)
		workConn.Close()
		return
	}

	// 将此工作连接分发到相关代理
	ctl.pm.HandleWorkConn(startMsg.ProxyName, workConn, &startMsg)
}

// handleNewProxyResp 处理新代理响应消息
func (ctl *Control) handleNewProxyResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NewProxyResp)
	// 服务端会对每个 NewProxy 消息返回 NewProxyResp 消息
	// 如果没有错误，则启动新的代理处理器
	err := ctl.pm.StartProxy(inMsg.ProxyName, inMsg.RemoteAddr, inMsg.Error)
	if err != nil {
		xl.Warnf("[%s] 启动错误: %v", inMsg.ProxyName, err)
	} else {
		xl.Infof("[%s] 启动代理成功", inMsg.ProxyName)
	}
}

// handleNatHoleResp 处理 NAT 穿透响应消息
func (ctl *Control) handleNatHoleResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NatHoleResp)

	// 将 NatHoleResp 消息分发到相关代理
	ok := ctl.msgTransporter.DispatchWithType(inMsg, msg.TypeNameNatHoleResp, inMsg.TransactionID)
	if !ok {
		xl.Tracef("将 NatHoleResp 消息分发到相关代理失败")
	}
}

// handlePong 处理 Pong 响应消息
func (ctl *Control) handlePong(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.Pong)

	if inMsg.Error != "" {
		xl.Errorf("pong 消息包含错误: %s", inMsg.Error)
		ctl.closeSession()
		return
	}
	ctl.lastPong.Store(time.Now())
	xl.Debugf("收到来自服务端的心跳")
}

// closeSession 关闭控制连接
func (ctl *Control) closeSession() {
	ctl.sessionCtx.Conn.Close()
	ctl.sessionCtx.Connector.Close()
}

// Close 关闭控制器
func (ctl *Control) Close() error {
	return ctl.GracefulClose(0)
}

// GracefulClose 优雅关闭控制器，等待指定时间后关闭
func (ctl *Control) GracefulClose(d time.Duration) error {
	ctl.pm.Close()
	ctl.vm.Close()

	time.Sleep(d)

	ctl.closeSession()
	return nil
}

// Done 返回一个通道，该通道将在所有资源释放后关闭
func (ctl *Control) Done() <-chan struct{} {
	return ctl.doneCh
}

// connectServer 返回与 frps 的新连接
func (ctl *Control) connectServer() (net.Conn, error) {
	return ctl.sessionCtx.Connector.Connect()
}

// registerMsgHandlers 注册消息处理器
func (ctl *Control) registerMsgHandlers() {
	ctl.msgDispatcher.RegisterHandler(&msg.ReqWorkConn{}, msg.AsyncHandler(ctl.handleReqWorkConn))
	ctl.msgDispatcher.RegisterHandler(&msg.NewProxyResp{}, ctl.handleNewProxyResp)
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleResp{}, ctl.handleNatHoleResp)
	ctl.msgDispatcher.RegisterHandler(&msg.Pong{}, ctl.handlePong)
}

// heartbeatWorker 向服务端发送心跳并检查心跳超时
func (ctl *Control) heartbeatWorker() {
	xl := ctl.xl

	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 {
		// 向服务端发送心跳
		sendHeartBeat := func() (bool, error) {
			xl.Debugf("向服务端发送心跳")
			pingMsg := &msg.Ping{}
			if err := ctl.sessionCtx.Auth.Setter.SetPing(pingMsg); err != nil {
				xl.Warnf("ping 身份验证期间出错: %v，跳过发送 ping 消息", err)
				return false, err
			}
			_ = ctl.msgDispatcher.Send(pingMsg)
			return false, nil
		}

		go wait.BackoffUntil(sendHeartBeat,
			wait.NewFastBackoffManager(wait.FastBackoffOptions{
				Duration:           time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatInterval) * time.Second,
				InitDurationIfFail: time.Second,
				Factor:             2.0,
				Jitter:             0.1,
				MaxDuration:        time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatInterval) * time.Second,
			}),
			true, ctl.doneCh,
		)
	}

	// 检查心跳超时
	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 && ctl.sessionCtx.Common.Transport.HeartbeatTimeout > 0 {
		go wait.Until(func() {
			if time.Since(ctl.lastPong.Load().(time.Time)) > time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatTimeout)*time.Second {
				xl.Warnf("心跳超时")
				ctl.closeSession()
				return
			}
		}, time.Second, ctl.doneCh)
	}
}

// worker 控制器的主工作协程
func (ctl *Control) worker() {
	xl := ctl.xl
	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()

	<-ctl.msgDispatcher.Done()
	xl.Debugf("控制消息分发器已退出")
	ctl.closeSession()

	ctl.pm.Close()
	ctl.vm.Close()
	close(ctl.doneCh)
}

// UpdateAllConfigurer 更新所有代理和访问者配置
func (ctl *Control) UpdateAllConfigurer(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	ctl.vm.UpdateAll(visitorCfgs)
	ctl.pm.UpdateAll(proxyCfgs)
	return nil
}
