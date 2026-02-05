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

package proxy

import (
	"context"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	libnet "github.com/fatedier/golib/net"
	"golang.org/x/time/rate"

	"github.com/purpose168/frp/pkg/config/types"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	plugin "github.com/purpose168/frp/pkg/plugin/client"
	"github.com/purpose168/frp/pkg/transport"
	"github.com/purpose168/frp/pkg/util/limit"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/pkg/vnet"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy, v1.ProxyConfigurer) Proxy{}

// RegisterProxyFactory 注册代理工厂函数，用于创建特定类型的代理实例
func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy, v1.ProxyConfigurer) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

// Proxy 定义如何为不同代理类型处理工作连接
type Proxy interface {
	Run() error
	// InWorkConn 接受注册到服务器的工作连接
	InWorkConn(net.Conn, *msg.StartWorkConn)
	SetInWorkConnCallback(func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) /* continue */ bool)
	Close()
}

// NewProxy 创建新的代理实例
func NewProxy(
	ctx context.Context,
	pxyConf v1.ProxyConfigurer,
	clientCfg *v1.ClientCommonConfig,
	encryptionKey []byte,
	msgTransporter transport.MessageTransporter,
	vnetController *vnet.Controller,
) (pxy Proxy) {
	var limiter *rate.Limiter
	limitBytes := pxyConf.GetBaseConfig().Transport.BandwidthLimit.Bytes()
	if limitBytes > 0 && pxyConf.GetBaseConfig().Transport.BandwidthLimitMode == types.BandwidthLimitModeClient {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	baseProxy := BaseProxy{
		baseCfg:        pxyConf.GetBaseConfig(),
		clientCfg:      clientCfg,
		encryptionKey:  encryptionKey,
		limiter:        limiter,
		msgTransporter: msgTransporter,
		vnetController: vnetController,
		xl:             xlog.FromContextSafe(ctx),
		ctx:            ctx,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(pxyConf)]
	if factory == nil {
		return nil
	}
	return factory(&baseProxy, pxyConf)
}

// BaseProxy 所有代理类型的基础结构
type BaseProxy struct {
	baseCfg        *v1.ProxyBaseConfig
	clientCfg      *v1.ClientCommonConfig
	encryptionKey  []byte
	msgTransporter transport.MessageTransporter
	vnetController *vnet.Controller
	limiter        *rate.Limiter
	// proxyPlugin 用于处理连接而不是拨号到本地服务
	// 目前仅对 TCP 协议有效
	proxyPlugin        plugin.Plugin
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) /* continue */ bool

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

// Run 运行代理，初始化插件
func (pxy *BaseProxy) Run() error {
	if pxy.baseCfg.Plugin.Type != "" {
		p, err := plugin.Create(pxy.baseCfg.Plugin.Type, plugin.PluginContext{
			Name:           pxy.baseCfg.Name,
			VnetController: pxy.vnetController,
		}, pxy.baseCfg.Plugin.ClientPluginOptions)
		if err != nil {
			return err
		}
		pxy.proxyPlugin = p
	}
	return nil
}

// Close 关闭代理并释放资源
func (pxy *BaseProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

// SetInWorkConnCallback 设置工作连接回调函数
func (pxy *BaseProxy) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pxy.inWorkConnCallback = cb
}

// InWorkConn 处理工作连接
func (pxy *BaseProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	if pxy.inWorkConnCallback != nil {
		if !pxy.inWorkConnCallback(pxy.baseCfg, conn, m) {
			return
		}
	}
	pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey)
}

// HandleTCPWorkConnection TCP 工作连接的通用处理器
func (pxy *BaseProxy) HandleTCPWorkConnection(workConn net.Conn, m *msg.StartWorkConn, encKey []byte) {
	xl := pxy.xl
	baseCfg := pxy.baseCfg
	var (
		remote io.ReadWriteCloser
		err    error
	)
	remote = workConn
	if pxy.limiter != nil {
		remote = libio.WrapReadWriteCloser(limit.NewReader(workConn, pxy.limiter), limit.NewWriter(workConn, pxy.limiter), func() error {
			return workConn.Close()
		})
	}

	xl.Tracef("处理 TCP 工作连接，使用加密: %t, 使用压缩: %t",
		baseCfg.Transport.UseEncryption, baseCfg.Transport.UseCompression)
	if baseCfg.Transport.UseEncryption {
		remote, err = libio.WithEncryption(remote, encKey)
		if err != nil {
			workConn.Close()
			xl.Errorf("创建加密流错误: %v", err)
			return
		}
	}
	var compressionResourceRecycleFn func()
	if baseCfg.Transport.UseCompression {
		remote, compressionResourceRecycleFn = libio.WithCompressionFromPool(remote)
	}

	// 检查是否需要发送代理协议信息
	var connInfo plugin.ConnectionInfo
	if m.SrcAddr != "" && m.SrcPort != 0 {
		if m.DstAddr == "" {
			m.DstAddr = "127.0.0.1"
		}
		srcAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.SrcAddr, strconv.Itoa(int(m.SrcPort))))
		dstAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.DstAddr, strconv.Itoa(int(m.DstPort))))
		connInfo.SrcAddr = srcAddr
		connInfo.DstAddr = dstAddr
	}

	if baseCfg.Transport.ProxyProtocolVersion != "" && m.SrcAddr != "" && m.SrcPort != 0 {
		// 使用通用的代理协议构建函数
		header := netpkg.BuildProxyProtocolHeaderStruct(connInfo.SrcAddr, connInfo.DstAddr, baseCfg.Transport.ProxyProtocolVersion)
		connInfo.ProxyProtocolHeader = header
	}
	connInfo.Conn = remote
	connInfo.UnderlyingConn = workConn

	if pxy.proxyPlugin != nil {
		// 如果设置了插件，让插件首先处理连接
		xl.Debugf("由插件处理: %s", pxy.proxyPlugin.Name())
		pxy.proxyPlugin.Handle(pxy.ctx, &connInfo)
		xl.Debugf("插件处理完成")
		return
	}

	localConn, err := libnet.Dial(
		net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort)),
		libnet.WithTimeout(10*time.Second),
	)
	if err != nil {
		workConn.Close()
		xl.Errorf("连接到本地服务 [%s:%d] 错误: %v", baseCfg.LocalIP, baseCfg.LocalPort, err)
		return
	}

	xl.Debugf("连接本地服务，localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())

	if connInfo.ProxyProtocolHeader != nil {
		if _, err := connInfo.ProxyProtocolHeader.WriteTo(localConn); err != nil {
			workConn.Close()
			xl.Errorf("向本地连接写入代理协议头错误: %v", err)
			return
		}
	}

	_, _, errs := libio.Join(localConn, remote)
	xl.Debugf("连接已关闭")
	if len(errs) > 0 {
		xl.Tracef("连接错误: %v", errs)
	}
	if compressionResourceRecycleFn != nil {
		compressionResourceRecycleFn()
	}
}
