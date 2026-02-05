// 版权所有 2017 fatedier, fatedier@gmail.com
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

package visitor

import (
	"context"
	"net"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	plugin "github.com/fatedier/frp/pkg/plugin/visitor"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

// Helper 封装了一些供访问者使用的函数
type Helper interface {
	// ConnectServer 直接连接到 frp 服务器
	ConnectServer() (net.Conn, error)
	// TransferConn 将连接转移到另一个访问者
	TransferConn(string, net.Conn) error
	// MsgTransporter 返回消息传输器，用于通过控制器向 frp 服务器发送和接收消息
	MsgTransporter() transport.MessageTransporter
	// VNetController 返回用于管理虚拟网络的 vnet 控制器
	VNetController() *vnet.Controller
	// RunID 返回当前控制器的运行 ID
	RunID() string
}

// Visitor 用于将流量从本地端口转发到远程服务
type Visitor interface {
	Run() error
	AcceptConn(conn net.Conn) error
	Close()
}

// NewVisitor 创建新的访问者实例
func NewVisitor(
	ctx context.Context,
	cfg v1.VisitorConfigurer,
	clientCfg *v1.ClientCommonConfig,
	helper Helper,
) (Visitor, error) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseConfig().Name)
	ctx = xlog.NewContext(ctx, xl)
	var visitor Visitor
	baseVisitor := BaseVisitor{
		clientCfg:  clientCfg,
		helper:     helper,
		ctx:        ctx,
		internalLn: netpkg.NewInternalListener(),
	}
	if cfg.GetBaseConfig().Plugin.Type != "" {
		p, err := plugin.Create(
			cfg.GetBaseConfig().Plugin.Type,
			plugin.PluginContext{
				Name:           cfg.GetBaseConfig().Name,
				Ctx:            ctx,
				VnetController: helper.VNetController(),
				SendConnToVisitor: func(conn net.Conn) {
					_ = baseVisitor.AcceptConn(conn)
				},
			},
			cfg.GetBaseConfig().Plugin.VisitorPluginOptions,
		)
		if err != nil {
			return nil, err
		}
		baseVisitor.plugin = p
	}
	switch cfg := cfg.(type) {
	case *v1.STCPVisitorConfig:
		visitor = &STCPVisitor{
			BaseVisitor: &baseVisitor,
			cfg:         cfg,
		}
	case *v1.XTCPVisitorConfig:
		visitor = &XTCPVisitor{
			BaseVisitor:   &baseVisitor,
			cfg:           cfg,
			startTunnelCh: make(chan struct{}),
		}
	case *v1.SUDPVisitorConfig:
		visitor = &SUDPVisitor{
			BaseVisitor:  &baseVisitor,
			cfg:          cfg,
			checkCloseCh: make(chan struct{}),
		}
	}
	return visitor, nil
}

// BaseVisitor 所有访问者的基础结构
type BaseVisitor struct {
	clientCfg  *v1.ClientCommonConfig
	helper     Helper
	l          net.Listener
	internalLn *netpkg.InternalListener
	plugin     plugin.Plugin

	mu  sync.RWMutex
	ctx context.Context
}

// AcceptConn 接受连接
func (v *BaseVisitor) AcceptConn(conn net.Conn) error {
	return v.internalLn.PutConn(conn)
}

// Close 关闭访问者
func (v *BaseVisitor) Close() {
	if v.l != nil {
		v.l.Close()
	}
	if v.internalLn != nil {
		v.internalLn.Close()
	}
	if v.plugin != nil {
		v.plugin.Close()
	}
}
