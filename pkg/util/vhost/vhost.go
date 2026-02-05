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
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// RouteInfo 路由信息类型
type RouteInfo string

// 路由信息键常量
const (
	RouteInfoKey   RouteInfo = "routeInfo"
	RouteConfigKey RouteInfo = "routeConfig"
)

// RequestRouteInfo 请求路由信息
type RequestRouteInfo struct {
	URL        string
	Host       string
	HTTPUser   string
	RemoteAddr string
	URLHost    string
	Endpoint   string
}

// 函数类型定义
type (
	// 多路复用函数类型
	muxFunc func(net.Conn) (net.Conn, map[string]string, error)
	// 认证函数类型
	authFunc func(conn net.Conn, username, password string, reqInfoMap map[string]string) (bool, error)
	// 主机重写函数类型
	hostRewriteFunc func(net.Conn, string) (net.Conn, error)
	// 成功钩子函数类型
	successHookFunc func(net.Conn, map[string]string) error
	// 失败钩子函数类型
	failHookFunc func(net.Conn)
)

// Muxer 多路复用器，用于 https 和 tcpmux 代理
// 它接受连接并从连接数据的开头提取虚拟主机信息
// 然后将连接路由到适当的监听器
type Muxer struct {
	listener net.Listener
	timeout  time.Duration

	vhostFunc      muxFunc
	checkAuth      authFunc
	successHook    successHookFunc
	failHook       failHookFunc
	rewriteHost    hostRewriteFunc
	registryRouter *Routers
}

// NewMuxer 创建一个新的多路复用器
func NewMuxer(
	listener net.Listener,
	vhostFunc muxFunc,
	timeout time.Duration,
) (mux *Muxer, err error) {
	mux = &Muxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      vhostFunc,
		registryRouter: NewRouters(),
	}
	// 启动多路复用器
	go mux.run()
	return mux, nil
}

// SetCheckAuthFunc 设置认证函数
func (v *Muxer) SetCheckAuthFunc(f authFunc) *Muxer {
	v.checkAuth = f
	return v
}

// SetSuccessHookFunc 设置成功钩子函数
func (v *Muxer) SetSuccessHookFunc(f successHookFunc) *Muxer {
	v.successHook = f
	return v
}

// SetFailHookFunc 设置失败钩子函数
func (v *Muxer) SetFailHookFunc(f failHookFunc) *Muxer {
	v.failHook = f
	return v
}

// SetRewriteHostFunc 设置主机重写函数
func (v *Muxer) SetRewriteHostFunc(f hostRewriteFunc) *Muxer {
	v.rewriteHost = f
	return v
}

// Close 关闭多路复用器
func (v *Muxer) Close() error {
	return v.listener.Close()
}

// ChooseEndpointFunc 选择端点函数类型
type ChooseEndpointFunc func() (string, error)

// CreateConnFunc 创建连接函数类型
type CreateConnFunc func(remoteAddr string) (net.Conn, error)

// CreateConnByEndpointFunc 通过端点创建连接函数类型
type CreateConnByEndpointFunc func(endpoint, remoteAddr string) (net.Conn, error)

// RouteConfig 路由配置，用于匹配 HTTP 请求
type RouteConfig struct {
	Domain          string
	Location        string
	RewriteHost     string
	Username        string
	Password        string
	Headers         map[string]string
	ResponseHeaders map[string]string
	RouteByHTTPUser string

	CreateConnFn           CreateConnFunc
	ChooseEndpointFn       ChooseEndpointFunc
	CreateConnByEndpointFn CreateConnByEndpointFunc
}

// Listen 监听新的域名，如果 rewriteHost 不为空且 rewriteHost 函数不为 nil
// 则将主机头重写为 rewriteHost
func (v *Muxer) Listen(ctx context.Context, cfg *RouteConfig) (l *Listener, err error) {
	// 创建监听器
	l = &Listener{
		name:            cfg.Domain,
		location:        cfg.Location,
		routeByHTTPUser: cfg.RouteByHTTPUser,
		rewriteHost:     cfg.RewriteHost,
		username:        cfg.Username,
		password:        cfg.Password,
		mux:             v,
		accept:          make(chan net.Conn),
		ctx:             ctx,
	}
	// 添加到路由注册表
	err = v.registryRouter.Add(cfg.Domain, cfg.Location, cfg.RouteByHTTPUser, l)
	if err != nil {
		return
	}
	return l, nil
}

// getListener 获取监听器
func (v *Muxer) getListener(name, path, httpUser string) (*Listener, bool) {
	// 查找路由的函数
	findRouter := func(inName, inPath, inHTTPUser string) (*Listener, bool) {
		// 尝试获取路由
		vr, ok := v.registryRouter.Get(inName, inPath, inHTTPUser)
		if ok {
			return vr.payload.(*Listener), true
		}
		// 尝试检查是否有未指定 routerByHTTPUser 的代理，这意味着匹配所有
		vr, ok = v.registryRouter.Get(inName, inPath, "")
		if ok {
			return vr.payload.(*Listener), true
		}
		return nil, false
	}

	// 首先检查完整的主机名
	// 如果不存在，则检查通配符域名，如 *.example.com
	l, ok := findRouter(name, path, httpUser)
	if ok {
		return l, true
	}

	// 例如：name = test.example.com，尝试匹配通配符域名
	// *.example.com
	// *.com
	domainSplit := strings.Split(name, ".")
	for len(domainSplit) >= 3 {
		domainSplit[0] = "*"
		name = strings.Join(domainSplit, ".")

		l, ok = findRouter(name, path, httpUser)
		if ok {
			return l, true
		}
		domainSplit = domainSplit[1:]
	}
	// 最后，尝试检查是否有域名为 "*" 的代理，这意味着匹配所有域名
	l, ok = findRouter("*", path, httpUser)
	if ok {
		return l, true
	}
	return nil, false
}

// run 运行多路复用器
func (v *Muxer) run() {
	for {
		// 接受连接
		conn, err := v.listener.Accept()
		if err != nil {
			return
		}
		// 处理连接
		go v.handle(conn)
	}
}

// handle 处理连接
func (v *Muxer) handle(c net.Conn) {
	// 设置超时
	if err := c.SetDeadline(time.Now().Add(v.timeout)); err != nil {
		_ = c.Close()
		return
	}

	// 提取虚拟主机信息
	sConn, reqInfoMap, err := v.vhostFunc(c)
	if err != nil {
		log.Debugf("从 http/https 请求中获取主机名错误: %v", err)
		_ = c.Close()
		return
	}

	// 获取监听器
	name := strings.ToLower(reqInfoMap["Host"])
	path := strings.ToLower(reqInfoMap["Path"])
	httpUser := reqInfoMap["HTTPUser"]
	l, ok := v.getListener(name, path, httpUser)
	if !ok {
		log.Debugf("未找到主机 [%s] 路径 [%s] HTTP 用户 [%s] 的监听器", name, path, httpUser)
		// 调用失败钩子
		v.failHook(sConn)
		return
	}

	// 获取日志记录器
	xl := xlog.FromContextSafe(l.ctx)
	// 调用成功钩子
	if v.successHook != nil {
		if err := v.successHook(c, reqInfoMap); err != nil {
			xl.Infof("成功函数在 vhost 连接上失败: %v", err)
			_ = c.Close()
			return
		}
	}

	// 如果存在认证函数且设置了用户名/密码
	// 则验证用户访问权限
	if l.mux.checkAuth != nil && l.username != "" {
		ok, err := l.mux.checkAuth(c, l.username, l.password, reqInfoMap)
		if !ok || err != nil {
			xl.Debugf("认证失败，用户: %s", l.username)
			_ = c.Close()
			return
		}
	}

	// 清除超时
	if err = sConn.SetDeadline(time.Time{}); err != nil {
		_ = c.Close()
		return
	}
	c = sConn

	// 记录请求
	xl.Debugf("新请求主机 [%s] 路径 [%s] HTTP 用户 [%s]", name, path, httpUser)
	// 将连接发送到监听器
	err = errors.PanicToError(func() {
		l.accept <- c
	})
	if err != nil {
		xl.Warnf("监听器已关闭，忽略此请求: %v", err)
	}
}

// Listener 监听器
type Listener struct {
	name            string
	location        string
	routeByHTTPUser string
	rewriteHost     string
	username        string
	password        string
	mux             *Muxer // 用于关闭多路复用器
	accept          chan net.Conn
	ctx             context.Context
}

// Accept 接受连接
func (l *Listener) Accept() (net.Conn, error) {
	// 获取日志记录器
	xl := xlog.FromContextSafe(l.ctx)
	// 从通道接收连接
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("监听器已关闭")
	}

	// 如果存在主机重写函数
	// 则使用修改后的主机头重写 HTTP 请求
	// 如果 l.rewriteHost 为空，则无需操作
	if l.mux.rewriteHost != nil {
		// 重写主机头
		sConn, err := l.mux.rewriteHost(conn, l.rewriteHost)
		if err != nil {
			xl.Warnf("主机头重写失败: %v", err)
			return nil, fmt.Errorf("主机头重写失败")
		}
		xl.Debugf("主机头重写成功，重写为 [%s]", l.rewriteHost)
		conn = sConn
	}
	// 创建带上下文的连接
	return netpkg.NewContextConn(l.ctx, conn), nil
}

// Close 关闭监听器
func (l *Listener) Close() error {
	// 从路由注册表中删除
	l.mux.registryRouter.Del(l.name, l.location, l.routeByHTTPUser)
	// 关闭接受通道
	close(l.accept)
	return nil
}

// Name 返回监听器名称
func (l *Listener) Name() string {
	return l.name
}

// Addr 返回监听器地址
func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}
