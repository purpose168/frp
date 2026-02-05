// Copyright 2017 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vhost

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	httppkg "github.com/purpose168/frp/pkg/util/http"
	"github.com/purpose168/frp/pkg/util/log"
)

// ErrNoRouteFound 未找到路由错误
var ErrNoRouteFound = errors.New("no route found")

// HTTPReverseProxyOptions HTTP 反向代理选项
type HTTPReverseProxyOptions struct {
	ResponseHeaderTimeoutS int64
}

// HTTPReverseProxy HTTP 反向代理
type HTTPReverseProxy struct {
	proxy       http.Handler
	vhostRouter *Routers

	responseHeaderTimeout time.Duration
}

// NewHTTPReverseProxy 创建一个新的 HTTP 反向代理
func NewHTTPReverseProxy(option HTTPReverseProxyOptions, vhostRouter *Routers) *HTTPReverseProxy {
	// 设置默认响应头超时时间为 60 秒
	if option.ResponseHeaderTimeoutS <= 0 {
		option.ResponseHeaderTimeoutS = 60
	}
	rp := &HTTPReverseProxy{
		responseHeaderTimeout: time.Duration(option.ResponseHeaderTimeoutS) * time.Second,
		vhostRouter:           vhostRouter,
	}
	// 创建反向代理
	proxy := &httputil.ReverseProxy{
		// 根据路由策略修改传入的请求
		Rewrite: func(r *httputil.ProxyRequest) {
			// 保留原始的 X-Forwarded-For 头
			r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
			// 设置 X-Forwarded-* 头
			r.SetXForwarded()
			req := r.Out
			// 设置 URL 协议为 http
			req.URL.Scheme = "http"
			// 从上下文中获取路由信息
			reqRouteInfo := req.Context().Value(RouteInfoKey).(*RequestRouteInfo)
			// 获取规范化的主机名
			originalHost, _ := httppkg.CanonicalHost(reqRouteInfo.Host)

			// 从上下文中获取路由配置
			rc := req.Context().Value(RouteConfigKey).(*RouteConfig)
			if rc != nil {
				// 如果配置了重写主机，则修改请求的主机头
				if rc.RewriteHost != "" {
					req.Host = rc.RewriteHost
				}

				var endpoint string
				// 如果配置了端点选择函数，则选择端点
				if rc.ChooseEndpointFn != nil {
					// 忽略错误，稍后会使用 CreateConnFn
					endpoint, _ = rc.ChooseEndpointFn()
					reqRouteInfo.Endpoint = endpoint
					log.Tracef("为 http 请求选择端点名称 [%s] 主机 [%s] 路径 [%s] http用户 [%s]",
						endpoint, originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
				}
				// 设置 {domain}.{location}.{routeByHTTPUser}.{endpoint} 作为 URL 主机，以便 HTTP 传输重用连接
				req.URL.Host = rc.Domain + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.Location)) + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.RouteByHTTPUser)) + "." +
					base64.StdEncoding.EncodeToString([]byte(endpoint))

				// 设置自定义请求头
				for k, v := range rc.Headers {
					req.Header.Set(k, v)
				}
			} else {
				// 如果没有路由配置，使用原始主机
				req.URL.Host = req.Host
			}
		},
		// 修改响应
		ModifyResponse: func(r *http.Response) error {
			// 从上下文中获取路由配置
			rc := r.Request.Context().Value(RouteConfigKey).(*RouteConfig)
			if rc != nil {
				// 设置自定义响应头
				for k, v := range rc.ResponseHeaders {
					r.Header.Set(k, v)
				}
			}
			return nil
		},
		// 创建到路由策略指定的代理的连接
		Transport: &http.Transport{
			ResponseHeaderTimeout: rp.responseHeaderTimeout,
			IdleConnTimeout:       60 * time.Second,
			MaxIdleConnsPerHost:   5,
			// 拨号上下文函数
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 根据路由信息创建连接
				return rp.CreateConnection(ctx.Value(RouteInfoKey).(*RequestRouteInfo), true)
			},
			// 代理函数
			Proxy: func(req *http.Request) (*url.URL, error) {
				// 如果 HTTP 第一行中有主机，则使用代理模式
				// GET http://example.com/ HTTP/1.1
				// Host: example.com
				//
				// 正常情况：
				// GET / HTTP/1.1
				// Host: example.com
				urlHost := req.Context().Value(RouteInfoKey).(*RequestRouteInfo).URLHost
				if urlHost != "" {
					return req.URL, nil
				}
				return nil, nil
			},
		},
		BufferPool: pool.NewBuffer(32 * 1024),
		ErrorLog:   stdlog.New(log.NewWriteLogger(log.WarnLevel, 2), "", 0),
		// 错误处理函数
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Logf(log.WarnLevel, 1, "执行 http 代理请求 [主机: %s] 错误: %v", req.Host, err)
			if err != nil {
				// 如果是超时错误，返回 504 状态码
				if e, ok := err.(net.Error); ok && e.Timeout() {
					rw.WriteHeader(http.StatusGatewayTimeout)
					return
				}
			}
			// 返回 404 状态码
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write(getNotFoundPageContent())
		},
	}
	// 创建 HTTP/2 清晰文本处理器
	rp.proxy = h2c.NewHandler(proxy, &http2.Server{})
	return rp
}

// Register 注册路由配置到反向代理
// 反向代理将使用 routeCfg 中的 CreateConnFn 来创建到远程服务的连接
func (rp *HTTPReverseProxy) Register(routeCfg RouteConfig) error {
	// 添加路由到虚拟主机路由器
	err := rp.vhostRouter.Add(routeCfg.Domain, routeCfg.Location, routeCfg.RouteByHTTPUser, &routeCfg)
	if err != nil {
		return err
	}
	return nil
}

// UnRegister 根据域名和位置注销路由配置
func (rp *HTTPReverseProxy) UnRegister(routeCfg RouteConfig) {
	// 从虚拟主机路由器中删除路由
	rp.vhostRouter.Del(routeCfg.Domain, routeCfg.Location, routeCfg.RouteByHTTPUser)
}

// GetRouteConfig 获取路由配置
func (rp *HTTPReverseProxy) GetRouteConfig(domain, location, routeByHTTPUser string) *RouteConfig {
	// 获取虚拟主机路由
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		log.Debugf("为 http 请求获取路由配置 [域名: %s] [位置: %s] [http用户: %s]", domain, location, routeByHTTPUser)
		return vr.payload.(*RouteConfig)
	}
	return nil
}

// CreateConnection 根据路由配置创建新连接
func (rp *HTTPReverseProxy) CreateConnection(reqRouteInfo *RequestRouteInfo, byEndpoint bool) (net.Conn, error) {
	// 获取规范化的主机名
	host, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	// 获取虚拟主机路由
	vr, ok := rp.getVhost(host, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
	if ok {
		// 如果通过端点创建连接
		if byEndpoint {
			fn := vr.payload.(*RouteConfig).CreateConnByEndpointFn
			if fn != nil {
				// 使用端点和远程地址创建连接
				return fn(reqRouteInfo.Endpoint, reqRouteInfo.RemoteAddr)
			}
		}
		// 使用远程地址创建连接
		fn := vr.payload.(*RouteConfig).CreateConnFn
		if fn != nil {
			return fn(reqRouteInfo.RemoteAddr)
		}
	}
	return nil, fmt.Errorf("%v: %s %s %s", ErrNoRouteFound, host, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
}

// CheckAuth 检查认证信息
func (rp *HTTPReverseProxy) CheckAuth(domain, location, routeByHTTPUser, user, passwd string) bool {
	// 获取虚拟主机路由
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		// 获取配置的用户名和密码
		checkUser := vr.payload.(*RouteConfig).Username
		checkPasswd := vr.payload.(*RouteConfig).Password
		// 如果配置了用户名或密码，则验证
		if (checkUser != "" || checkPasswd != "") && (checkUser != user || checkPasswd != passwd) {
			return false
		}
	}
	return true
}

// getVhost 尝试根据路由策略获取虚拟主机路由
func (rp *HTTPReverseProxy) getVhost(domain, location, routeByHTTPUser string) (*Router, bool) {
	// 查找路由的函数
	findRouter := func(inDomain, inLocation, inRouteByHTTPUser string) (*Router, bool) {
		// 尝试获取路由
		vr, ok := rp.vhostRouter.Get(inDomain, inLocation, inRouteByHTTPUser)
		if ok {
			return vr, ok
		}
		// 尝试检查是否有未指定 routerByHTTPUser 的代理，这意味着匹配所有
		vr, ok = rp.vhostRouter.Get(inDomain, inLocation, "")
		if ok {
			return vr, ok
		}
		return nil, false
	}

	// 首先检查完整的主机名
	// 如果不存在，则检查通配符域名，如 *.example.com
	vr, ok := findRouter(domain, location, routeByHTTPUser)
	if ok {
		return vr, ok
	}

	// 例如：domain = test.example.com，尝试匹配通配符域名
	// *.example.com
	// *.com
	domainSplit := strings.Split(domain, ".")
	for len(domainSplit) >= 3 {
		domainSplit[0] = "*"
		domain = strings.Join(domainSplit, ".")
		vr, ok = findRouter(domain, location, routeByHTTPUser)
		if ok {
			return vr, true
		}
		domainSplit = domainSplit[1:]
	}

	// 最后，尝试检查是否有域名为 "*" 的代理，这意味着匹配所有域名
	vr, ok = findRouter("*", location, routeByHTTPUser)
	if ok {
		return vr, true
	}
	return nil, false
}

// connectHandler 处理 CONNECT 方法请求
func (rp *HTTPReverseProxy) connectHandler(rw http.ResponseWriter, req *http.Request) {
	// 获取连接劫持器
	hj, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 劫持连接
	client, _, err := hj.Hijack()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 创建到后端的连接
	remote, err := rp.CreateConnection(req.Context().Value(RouteInfoKey).(*RequestRouteInfo), false)
	if err != nil {
		// 返回 404 响应
		_ = NotFoundResponse().Write(client)
		client.Close()
		return
	}
	// 将请求写入后端连接
	_ = req.Write(remote)
	// 双向转发数据
	go libio.Join(remote, client)
}

// parseBasicAuth 解析基本认证
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// 不区分大小写的前缀匹配
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	// 解码 base64
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// injectRequestInfoToCtx 将请求信息注入到上下文中
func (rp *HTTPReverseProxy) injectRequestInfoToCtx(req *http.Request) *http.Request {
	user := ""
	// 如果 URL 主机不为空，则是代理请求。从 Proxy-Authorization 头获取 HTTP 用户
	if req.URL.Host != "" {
		proxyAuth := req.Header.Get("Proxy-Authorization")
		if proxyAuth != "" {
			user, _, _ = parseBasicAuth(proxyAuth)
		}
	}
	// 否则从基本认证获取用户
	if user == "" {
		user, _, _ = req.BasicAuth()
	}

	// 创建请求路由信息
	reqRouteInfo := &RequestRouteInfo{
		URL:        req.URL.Path,
		Host:       req.Host,
		HTTPUser:   user,
		RemoteAddr: req.RemoteAddr,
		URLHost:    req.URL.Host,
	}

	// 获取路由配置
	originalHost, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	rc := rp.GetRouteConfig(originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser)

	// 将路由信息注入到上下文中
	newctx := req.Context()
	newctx = context.WithValue(newctx, RouteInfoKey, reqRouteInfo)
	newctx = context.WithValue(newctx, RouteConfigKey, rc)
	return req.Clone(newctx)
}

// ServeHTTP 处理 HTTP 请求
func (rp *HTTPReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// 获取域名和路径
	domain, _ := httppkg.CanonicalHost(req.Host)
	location := req.URL.Path
	// 获取基本认证信息
	user, passwd, _ := req.BasicAuth()
	// 检查认证
	if !rp.CheckAuth(domain, location, user, user, passwd) {
		rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// 将请求信息注入到上下文中
	newreq := rp.injectRequestInfoToCtx(req)
	// 如果是 CONNECT 方法，使用连接处理器
	if req.Method == http.MethodConnect {
		rp.connectHandler(rw, newreq)
	} else {
		// 否则使用反向代理
		rp.proxy.ServeHTTP(rw, newreq)
	}
}
