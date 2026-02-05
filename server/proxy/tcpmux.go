// Copyright 2020 guylewin, guy@lewin.co.il
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
	"fmt"
	"net"
	"reflect"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

// init 注册TCPMux代理工厂
func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.TCPMuxProxyConfig{}), NewTCPMuxProxy)
}

// TCPMuxProxy TCP多路复用代理
// 实现了Proxy接口，用于处理TCP多路复用代理请求
type TCPMuxProxy struct {
	*BaseProxy
	// cfg TCPMux代理配置
	cfg *v1.TCPMuxProxyConfig
}

// NewTCPMuxProxy 创建一个新的TCPMux代理
// 参数baseProxy是基础代理
func NewTCPMuxProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.TCPMuxProxyConfig)
	if !ok {
		return nil
	}
	return &TCPMuxProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// httpConnectListen 监听HTTP连接
// 参数domain是域名
// 参数routeByHTTPUser是按HTTP用户路由
// 参数httpUser是HTTP用户名
// 参数httpPwd是HTTP密码
// 参数addrs是地址列表
// 返回值是新的地址列表和可能的错误
func (pxy *TCPMuxProxy) httpConnectListen(
	domain, routeByHTTPUser, httpUser, httpPwd string, addrs []string) ([]string, error,
) {
	var l net.Listener
	var err error
	routeConfig := &vhost.RouteConfig{
		Domain:          domain,
		RouteByHTTPUser: routeByHTTPUser,
		Username:        httpUser,
		Password:        httpPwd,
	}
	if pxy.cfg.LoadBalancer.Group != "" {
		l, err = pxy.rc.TCPMuxGroupCtl.Listen(pxy.ctx, pxy.cfg.Multiplexer,
			pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, *routeConfig)
	} else {
		l, err = pxy.rc.TCPMuxHTTPConnectMuxer.Listen(pxy.ctx, routeConfig)
	}
	if err != nil {
		return nil, err
	}
	pxy.xl.Infof("tcpmux httpconnect多路复用器监听主机 [%s], 组 [%s] 按HTTP用户路由 [%s]",
		domain, pxy.cfg.LoadBalancer.Group, pxy.cfg.RouteByHTTPUser)
	pxy.listeners = append(pxy.listeners, l)
	return append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.TCPMuxHTTPConnectPort)), nil
}

// httpConnectRun 启动HTTP连接
// 返回值是远程地址和可能的错误
func (pxy *TCPMuxProxy) httpConnectRun() (remoteAddr string, err error) {
	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		if domain == "" {
			continue
		}

		addrs, err = pxy.httpConnectListen(domain, pxy.cfg.RouteByHTTPUser, pxy.cfg.HTTPUser, pxy.cfg.HTTPPassword, addrs)
		if err != nil {
			return "", err
		}
	}

	if pxy.cfg.SubDomain != "" {
		addrs, err = pxy.httpConnectListen(pxy.cfg.SubDomain+"."+pxy.serverCfg.SubDomainHost,
			pxy.cfg.RouteByHTTPUser, pxy.cfg.HTTPUser, pxy.cfg.HTTPPassword, addrs)
		if err != nil {
			return "", err
		}
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = strings.Join(addrs, ",")
	return remoteAddr, err
}

// Run 启动TCPMux代理
// 返回值是远程地址和可能的错误
func (pxy *TCPMuxProxy) Run() (remoteAddr string, err error) {
	switch v1.TCPMultiplexerType(pxy.cfg.Multiplexer) {
	case v1.TCPMultiplexerHTTPConnect:
		remoteAddr, err = pxy.httpConnectRun()
	default:
		err = fmt.Errorf("未知的多路复用器 [%s]", pxy.cfg.Multiplexer)
	}

	if err != nil {
		pxy.Close()
	}
	return remoteAddr, err
}

// Close 关闭TCPMux代理
func (pxy *TCPMuxProxy) Close() {
	pxy.BaseProxy.Close()
}
