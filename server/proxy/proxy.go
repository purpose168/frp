// Copyright 2017 fatedier, fatedier@gmail.com
//
// 依据 Apache License, Version 2.0 许可协议授权；
// 除非符合许可协议的规定，否则不得使用此文件。
// 您可以在以下网址获取许可协议的副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或者书面同意，否则本软件按"原样"分发，
// 不附带任何明示或暗示的担保或条件。
// 请参阅许可协议以了解管理权限和限制的特定语言。

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	"golang.org/x/time/rate"

	"github.com/purpose168/frp/pkg/config/types"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	plugin "github.com/purpose168/frp/pkg/plugin/server"
	"github.com/purpose168/frp/pkg/util/limit"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/server/controller"
	"github.com/purpose168/frp/server/metrics"
)

// proxyFactoryRegistry 代理工厂注册表
// 存储不同类型代理的工厂函数
var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy) Proxy{}

// RegisterProxyFactory 注册代理工厂
// 参数proxyConfType是代理配置类型
// 参数factory是代理工厂函数
func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

// GetWorkConnFn 获取工作连接的函数类型
type GetWorkConnFn func() (net.Conn, error)

// Proxy 代理接口
// 定义了代理的基本操作
type Proxy interface {
	// Context 返回代理的上下文
	Context() context.Context
	// Run 启动代理
	// 返回值是远程地址和可能的错误
	Run() (remoteAddr string, err error)
	// GetName 返回代理名称
	GetName() string
	// GetConfigurer 返回代理配置
	GetConfigurer() v1.ProxyConfigurer
	// GetWorkConnFromPool 从连接池获取工作连接
	// 参数src是源地址
	// 参数dst是目标地址
	// 返回值是工作连接和可能的错误
	GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error)
	// GetUsedPortsNum 返回使用的端口数量
	GetUsedPortsNum() int
	// GetResourceController 返回资源控制器
	GetResourceController() *controller.ResourceController
	// GetUserInfo 返回用户信息
	GetUserInfo() plugin.UserInfo
	// GetLimiter 返回速率限制器
	GetLimiter() *rate.Limiter
	// GetLoginMsg 返回登录消息
	GetLoginMsg() *msg.Login
	// Close 关闭代理
	Close()
}

// BaseProxy 基础代理
// 实现了Proxy接口的大部分方法，作为其他代理的基类
type BaseProxy struct {
	// name 代理名称
	name string
	// rc 资源控制器
	rc *controller.ResourceController
	// listeners 监听器列表
	listeners []net.Listener
	// usedPortsNum 使用的端口数量
	usedPortsNum int
	// poolCount 连接池大小
	poolCount int
	// getWorkConnFn 获取工作连接的函数
	getWorkConnFn GetWorkConnFn
	// serverCfg 服务器配置
	serverCfg *v1.ServerConfig
	// encryptionKey 加密密钥
	encryptionKey []byte
	// limiter 速率限制器
	limiter *rate.Limiter
	// userInfo 用户信息
	userInfo plugin.UserInfo
	// loginMsg 登录消息
	loginMsg *msg.Login
	// configurer 代理配置
	configurer v1.ProxyConfigurer

	// mu 读写锁
	mu sync.RWMutex
	// xl 日志记录器
	xl *xlog.Logger
	// ctx 上下文
	ctx context.Context
}

// GetName 返回代理名称
func (pxy *BaseProxy) GetName() string {
	return pxy.name
}

// Context 返回代理的上下文
func (pxy *BaseProxy) Context() context.Context {
	return pxy.ctx
}

// GetUsedPortsNum 返回使用的端口数量
func (pxy *BaseProxy) GetUsedPortsNum() int {
	return pxy.usedPortsNum
}

// GetResourceController 返回资源控制器
func (pxy *BaseProxy) GetResourceController() *controller.ResourceController {
	return pxy.rc
}

// GetUserInfo 返回用户信息
func (pxy *BaseProxy) GetUserInfo() plugin.UserInfo {
	return pxy.userInfo
}

// GetLoginMsg 返回登录消息
func (pxy *BaseProxy) GetLoginMsg() *msg.Login {
	return pxy.loginMsg
}

// GetLimiter 返回速率限制器
func (pxy *BaseProxy) GetLimiter() *rate.Limiter {
	return pxy.limiter
}

// GetConfigurer 返回代理配置
func (pxy *BaseProxy) GetConfigurer() v1.ProxyConfigurer {
	return pxy.configurer
}

// Close 关闭代理
// 关闭所有监听器
func (pxy *BaseProxy) Close() {
	xl := xlog.FromContextSafe(pxy.ctx)
	xl.Infof("代理正在关闭")
	for _, l := range pxy.listeners {
		l.Close()
	}
}

// GetWorkConnFromPool 从连接池获取工作连接
// 为了快速响应，从连接池取出连接后立即向frpc发送StartWorkConn消息
func (pxy *BaseProxy) GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error) {
	xl := xlog.FromContextSafe(pxy.ctx)
	// 尝试从连接池获取所有连接
	for i := 0; i < pxy.poolCount+1; i++ {
		if workConn, err = pxy.getWorkConnFn(); err != nil {
			xl.Warnf("获取工作连接失败: %v", err)
			return
		}
		xl.Debugf("获取新的工作连接: [%s]", workConn.RemoteAddr().String())
		xl.Spawn().AppendPrefix(pxy.GetName())
		workConn = netpkg.NewContextConn(pxy.ctx, workConn)

		var (
			srcAddr    string
			dstAddr    string
			srcPortStr string
			dstPortStr string
			srcPort    uint64
			dstPort    uint64
		)

		if src != nil {
			srcAddr, srcPortStr, _ = net.SplitHostPort(src.String())
			srcPort, _ = strconv.ParseUint(srcPortStr, 10, 16)
		}
		if dst != nil {
			dstAddr, dstPortStr, _ = net.SplitHostPort(dst.String())
			dstPort, _ = strconv.ParseUint(dstPortStr, 10, 16)
		}
		err := msg.WriteMsg(workConn, &msg.StartWorkConn{
			ProxyName: pxy.GetName(),
			SrcAddr:   srcAddr,
			SrcPort:   uint16(srcPort),
			DstAddr:   dstAddr,
			DstPort:   uint16(dstPort),
			Error:     "",
		})
		if err != nil {
			xl.Warnf("向工作连接发送消息失败: %v, 尝试次数: %d", err, i)
			workConn.Close()
		} else {
			break
		}
	}

	if err != nil {
		xl.Errorf("最终获取工作连接失败")
		return
	}
	return
}

// startCommonTCPListenersHandler 为每个监听器启动一个goroutine处理连接
func (pxy *BaseProxy) startCommonTCPListenersHandler() {
	xl := xlog.FromContextSafe(pxy.ctx)
	for _, listener := range pxy.listeners {
		go func(l net.Listener) {
			var tempDelay time.Duration // 接受连接失败时的休眠时间

			for {
				// 阻塞等待连接
				// 如果监听器被关闭，返回错误
				c, err := l.Accept()
				if err != nil {
					if err, ok := err.(interface{ Temporary() bool }); ok && err.Temporary() {
						if tempDelay == 0 {
							tempDelay = 5 * time.Millisecond
						} else {
							tempDelay *= 2
						}
						if maxTime := 1 * time.Second; tempDelay > maxTime {
							tempDelay = maxTime
						}
						xl.Infof("遇到临时错误: %s, 休眠 %s ...", err, tempDelay)
						time.Sleep(tempDelay)
						continue
					}

					xl.Warnf("监听器已关闭: %s", err)
					return
				}
				xl.Infof("获取用户连接 [%s]", c.RemoteAddr().String())
				go pxy.handleUserTCPConnection(c)
			}
		}(listener)
	}
}

// handleUserTCPConnection 处理用户的TCP连接
func (pxy *BaseProxy) handleUserTCPConnection(userConn net.Conn) {
	xl := xlog.FromContextSafe(pxy.Context())
	defer userConn.Close()

	cfg := pxy.configurer.GetBaseConfig()
	// 服务器插件钩子
	rc := pxy.GetResourceController()
	content := &plugin.NewUserConnContent{
		User:       pxy.GetUserInfo(),
		ProxyName:  pxy.GetName(),
		ProxyType:  cfg.Type,
		RemoteAddr: userConn.RemoteAddr().String(),
	}
	_, err := rc.PluginManager.NewUserConn(content)
	if err != nil {
		xl.Warnf("用户连接 [%s] 被拒绝, 错误:%v", content.RemoteAddr, err)
		return
	}

	// 尝试从连接池获取连接
	workConn, err := pxy.GetWorkConnFromPool(userConn.RemoteAddr(), userConn.LocalAddr())
	if err != nil {
		return
	}
	defer workConn.Close()

	var local io.ReadWriteCloser = workConn
	xl.Tracef("处理用户TCP连接, 使用加密: %t, 使用压缩: %t",
		cfg.Transport.UseEncryption, cfg.Transport.UseCompression)
	if cfg.Transport.UseEncryption {
		local, err = libio.WithEncryption(local, pxy.encryptionKey)
		if err != nil {
			xl.Errorf("创建加密流错误: %v", err)
			return
		}
	}
	if cfg.Transport.UseCompression {
		var recycleFn func()
		local, recycleFn = libio.WithCompressionFromPool(local)
		defer recycleFn()
	}

	if pxy.GetLimiter() != nil {
		local = libio.WrapReadWriteCloser(limit.NewReader(local, pxy.GetLimiter()), limit.NewWriter(local, pxy.GetLimiter()), func() error {
			return local.Close()
		})
	}

	xl.Debugf("连接两个连接, workConn(本地[%s] 远程[%s]) userConn(本地[%s] 远程[%s])", workConn.LocalAddr().String(),
		workConn.RemoteAddr().String(), userConn.LocalAddr().String(), userConn.RemoteAddr().String())

	name := pxy.GetName()
	proxyType := cfg.Type
	metrics.Server.OpenConnection(name, proxyType)
	inCount, outCount, _ := libio.Join(local, userConn)
	metrics.Server.CloseConnection(name, proxyType)
	metrics.Server.AddTrafficIn(name, proxyType, inCount)
	metrics.Server.AddTrafficOut(name, proxyType, outCount)
	xl.Debugf("连接已关闭")
}

// Options 代理选项
// 用于创建代理的配置选项
type Options struct {
	// UserInfo 用户信息
	UserInfo plugin.UserInfo
	// LoginMsg 登录消息
	LoginMsg *msg.Login
	// PoolCount 连接池大小
	PoolCount int
	// ResourceController 资源控制器
	ResourceController *controller.ResourceController
	// GetWorkConnFn 获取工作连接的函数
	GetWorkConnFn GetWorkConnFn
	// Configurer 代理配置
	Configurer v1.ProxyConfigurer
	// ServerCfg 服务器配置
	ServerCfg *v1.ServerConfig
	// EncryptionKey 加密密钥
	EncryptionKey []byte
}

// NewProxy 创建一个新的代理
// 参数ctx是上下文
// 参数options是代理选项
// 返回值是创建的代理和可能的错误
func NewProxy(ctx context.Context, options *Options) (pxy Proxy, err error) {
	configurer := options.Configurer
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(configurer.GetBaseConfig().Name)

	var limiter *rate.Limiter
	limitBytes := configurer.GetBaseConfig().Transport.BandwidthLimit.Bytes()
	if limitBytes > 0 && configurer.GetBaseConfig().Transport.BandwidthLimitMode == types.BandwidthLimitModeServer {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	basePxy := BaseProxy{
		name:          configurer.GetBaseConfig().Name,
		rc:            options.ResourceController,
		listeners:     make([]net.Listener, 0),
		poolCount:     options.PoolCount,
		getWorkConnFn: options.GetWorkConnFn,
		serverCfg:     options.ServerCfg,
		encryptionKey: options.EncryptionKey,
		limiter:       limiter,
		xl:            xl,
		ctx:           xlog.NewContext(ctx, xl),
		userInfo:      options.UserInfo,
		loginMsg:      options.LoginMsg,
		configurer:    configurer,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(configurer)]
	if factory == nil {
		return pxy, fmt.Errorf("不支持的代理类型")
	}
	pxy = factory(&basePxy)
	if pxy == nil {
		return nil, fmt.Errorf("代理创建失败")
	}
	return pxy, nil
}

// Manager 代理管理器
// 负责管理所有代理的创建、删除和查找
type Manager struct {
	// pxys 代理列表，键为代理名称
	pxys map[string]Proxy

	// mu 读写锁
	mu sync.RWMutex
}

// NewManager 创建一个新的代理管理器
func NewManager() *Manager {
	return &Manager{
		pxys: make(map[string]Proxy),
	}
}

// Add 添加一个代理
// 参数name是代理名称
// 参数pxy是代理实例
// 返回值是可能的错误
func (pm *Manager) Add(name string, pxy Proxy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.pxys[name]; ok {
		return fmt.Errorf("代理名称 [%s] 已被使用", name)
	}

	pm.pxys[name] = pxy
	return nil
}

// Exist 检查代理是否存在
// 参数name是代理名称
// 返回值是代理是否存在
func (pm *Manager) Exist(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, ok := pm.pxys[name]
	return ok
}

// Del 删除一个代理
// 参数name是代理名称
func (pm *Manager) Del(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.pxys, name)
}

// GetByName 根据名称获取代理
// 参数name是代理名称
// 返回值是代理实例和是否存在
func (pm *Manager) GetByName(name string) (pxy Proxy, ok bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pxy, ok = pm.pxys[name]
	return
}
