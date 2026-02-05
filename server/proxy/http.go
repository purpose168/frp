// Copyright 2019 fatedier, fatedier@gmail.com
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
	"io"
	"net"
	"reflect"
	"strings"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/util/limit"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/util"
	"github.com/purpose168/frp/pkg/util/vhost"
	"github.com/purpose168/frp/server/metrics"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.HTTPProxyConfig{}), NewHTTPProxy)
}

// HTTPProxy HTTP代理
// 实现了Proxy接口，用于处理HTTP代理请求
type HTTPProxy struct {
	*BaseProxy
	// cfg HTTP代理配置
	cfg *v1.HTTPProxyConfig

	// closeFuncs 关闭函数列表，用于在代理关闭时执行清理操作
	closeFuncs []func()
}

// NewHTTPProxy 创建一个新的HTTP代理
// 参数baseProxy是基础代理
func NewHTTPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.HTTPProxyConfig)
	if !ok {
		return nil
	}
	return &HTTPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// Run 启动HTTP代理
// 返回值是远程地址和可能的错误
func (pxy *HTTPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	routeConfig := vhost.RouteConfig{
		RewriteHost:     pxy.cfg.HostHeaderRewrite,
		RouteByHTTPUser: pxy.cfg.RouteByHTTPUser,
		Headers:         pxy.cfg.RequestHeaders.Set,
		ResponseHeaders: pxy.cfg.ResponseHeaders.Set,
		Username:        pxy.cfg.HTTPUser,
		Password:        pxy.cfg.HTTPPassword,
		CreateConnFn:    pxy.GetRealConn,
	}

	locations := pxy.cfg.Locations
	if len(locations) == 0 {
		locations = []string{""}
	}

	defer func() {
		if err != nil {
			pxy.Close()
		}
	}()

	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		if domain == "" {
			continue
		}

		routeConfig.Domain = domain
		for _, location := range locations {
			routeConfig.Location = location

			tmpRouteConfig := routeConfig

			// 处理组
			if pxy.cfg.LoadBalancer.Group != "" {
				err = pxy.rc.HTTPGroupCtl.Register(pxy.name, pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, routeConfig)
				if err != nil {
					return
				}

				pxy.closeFuncs = append(pxy.closeFuncs, func() {
					pxy.rc.HTTPGroupCtl.UnRegister(pxy.name, pxy.cfg.LoadBalancer.Group, tmpRouteConfig)
				})
			} else {
				// 无组
				err = pxy.rc.HTTPReverseProxy.Register(routeConfig)
				if err != nil {
					return
				}
				pxy.closeFuncs = append(pxy.closeFuncs, func() {
					pxy.rc.HTTPReverseProxy.UnRegister(tmpRouteConfig)
				})
			}
			addrs = append(addrs, util.CanonicalAddr(routeConfig.Domain, pxy.serverCfg.VhostHTTPPort))
			xl.Infof("HTTP代理监听主机 [%s] 路径 [%s] 组 [%s], 按HTTP用户路由 [%s]",
				routeConfig.Domain, routeConfig.Location, pxy.cfg.LoadBalancer.Group, pxy.cfg.RouteByHTTPUser)
		}
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + pxy.serverCfg.SubDomainHost
		for _, location := range locations {
			routeConfig.Location = location

			tmpRouteConfig := routeConfig

			// 处理组
			if pxy.cfg.LoadBalancer.Group != "" {
				err = pxy.rc.HTTPGroupCtl.Register(pxy.name, pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, routeConfig)
				if err != nil {
					return
				}

				pxy.closeFuncs = append(pxy.closeFuncs, func() {
					pxy.rc.HTTPGroupCtl.UnRegister(pxy.name, pxy.cfg.LoadBalancer.Group, tmpRouteConfig)
				})
			} else {
				// 无组
				err = pxy.rc.HTTPReverseProxy.Register(routeConfig)
				if err != nil {
					return
				}
				pxy.closeFuncs = append(pxy.closeFuncs, func() {
					pxy.rc.HTTPReverseProxy.UnRegister(tmpRouteConfig)
				})
			}
			addrs = append(addrs, util.CanonicalAddr(tmpRouteConfig.Domain, pxy.serverCfg.VhostHTTPPort))

			xl.Infof("HTTP代理监听主机 [%s] 路径 [%s] 组 [%s], 按HTTP用户路由 [%s]",
				routeConfig.Domain, routeConfig.Location, pxy.cfg.LoadBalancer.Group, pxy.cfg.RouteByHTTPUser)
		}
	}
	remoteAddr = strings.Join(addrs, ",")
	return
}

// GetRealConn 获取实际连接
// 参数remoteAddr是远程地址
// 返回值是工作连接和可能的错误
func (pxy *HTTPProxy) GetRealConn(remoteAddr string) (workConn net.Conn, err error) {
	xl := pxy.xl
	rAddr, errRet := net.ResolveTCPAddr("tcp", remoteAddr)
	if errRet != nil {
		xl.Warnf("解析TCP地址 [%s] 错误: %v", remoteAddr, errRet)
		// 这里不返回错误，因为对于未启用代理协议的代理，remoteAddr不是必需的
	}

	tmpConn, errRet := pxy.GetWorkConnFromPool(rAddr, nil)
	if errRet != nil {
		err = errRet
		return
	}

	var rwc io.ReadWriteCloser = tmpConn
	if pxy.cfg.Transport.UseEncryption {
		rwc, err = libio.WithEncryption(rwc, pxy.encryptionKey)
		if err != nil {
			xl.Errorf("创建加密流错误: %v", err)
			return
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

	workConn = netpkg.WrapReadWriteCloserToConn(rwc, tmpConn)
	workConn = netpkg.WrapStatsConn(workConn, pxy.updateStatsAfterClosedConn)
	metrics.Server.OpenConnection(pxy.GetName(), pxy.GetConfigurer().GetBaseConfig().Type)
	return
}

// updateStatsAfterClosedConn 连接关闭后更新统计信息
// 参数totalRead是读取的总字节数
// 参数totalWrite是写入的总字节数
func (pxy *HTTPProxy) updateStatsAfterClosedConn(totalRead, totalWrite int64) {
	name := pxy.GetName()
	proxyType := pxy.GetConfigurer().GetBaseConfig().Type
	metrics.Server.CloseConnection(name, proxyType)
	metrics.Server.AddTrafficIn(name, proxyType, totalWrite)
	metrics.Server.AddTrafficOut(name, proxyType, totalRead)
}

// Close 关闭HTTP代理
func (pxy *HTTPProxy) Close() {
	pxy.BaseProxy.Close()
	for _, closeFn := range pxy.closeFuncs {
		closeFn()
	}
}
