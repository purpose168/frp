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
	"net"
	"reflect"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.HTTPSProxyConfig{}), NewHTTPSProxy)
}

// HTTPSProxy HTTPS代理
// 实现了Proxy接口，用于处理HTTPS代理请求
type HTTPSProxy struct {
	*BaseProxy
	// cfg HTTPS代理配置
	cfg *v1.HTTPSProxyConfig
}

// NewHTTPSProxy 创建一个新的HTTPS代理
// 参数baseProxy是基础代理
func NewHTTPSProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.HTTPSProxyConfig)
	if !ok {
		return nil
	}
	return &HTTPSProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// Run 启动HTTPS代理
// 返回值是远程地址和可能的错误
func (pxy *HTTPSProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	routeConfig := &vhost.RouteConfig{}

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

		l, err := pxy.listenForDomain(routeConfig, domain)
		if err != nil {
			return "", err
		}
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.VhostHTTPSPort))
		xl.Infof("HTTPS代理监听主机 [%s] 组 [%s]", domain, pxy.cfg.LoadBalancer.Group)
	}

	if pxy.cfg.SubDomain != "" {
		domain := pxy.cfg.SubDomain + "." + pxy.serverCfg.SubDomainHost
		l, err := pxy.listenForDomain(routeConfig, domain)
		if err != nil {
			return "", err
		}
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.VhostHTTPSPort))
		xl.Infof("HTTPS代理监听主机 [%s] 组 [%s]", domain, pxy.cfg.LoadBalancer.Group)
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = strings.Join(addrs, ",")
	return
}

// Close 关闭HTTPS代理
func (pxy *HTTPSProxy) Close() {
	pxy.BaseProxy.Close()
}

// listenForDomain 为指定域名创建监听器
// 参数routeConfig是路由配置
// 参数domain是域名
// 返回值是监听器和可能的错误
func (pxy *HTTPSProxy) listenForDomain(routeConfig *vhost.RouteConfig, domain string) (net.Listener, error) {
	tmpRouteConfig := *routeConfig
	tmpRouteConfig.Domain = domain

	if pxy.cfg.LoadBalancer.Group != "" {
		return pxy.rc.HTTPSGroupCtl.Listen(
			pxy.ctx,
			pxy.cfg.LoadBalancer.Group,
			pxy.cfg.LoadBalancer.GroupKey,
			tmpRouteConfig,
		)
	}
	return pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, &tmpRouteConfig)
}
