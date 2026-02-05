// Copyright 2025 The frp Authors
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

package group

import (
	"context"
	"net"
	"sync"

	gerr "github.com/fatedier/golib/errors"

	"github.com/purpose168/frp/pkg/util/vhost"
)

// HTTPSGroupController 管理多个HTTPS组
// 负责创建、维护和删除HTTPS组
// 每个组对应一个域名和路由配置
type HTTPSGroupController struct {
	// groups 存储所有HTTPS组，键为组名
	groups map[string]*HTTPSGroup

	// httpsMuxer 用于多路复用HTTPS连接的虚拟主机多路复用器
	httpsMuxer *vhost.HTTPSMuxer

	// mu 保护groups的并发访问
	mu sync.Mutex
}

// NewHTTPSGroupController 创建一个新的HTTPS组控制器
// 参数httpsMuxer是用于多路复用HTTPS连接的虚拟主机多路复用器
func NewHTTPSGroupController(httpsMuxer *vhost.HTTPSMuxer) *HTTPSGroupController {
	return &HTTPSGroupController{
		groups:     make(map[string]*HTTPSGroup),
		httpsMuxer: httpsMuxer,
	}
}

// Listen 为指定的HTTPS组创建监听器
// 如果组不存在，则创建新组
// 参数ctx用于控制操作的生命周期
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数routeConfig是路由配置，包含域名和其他路由信息
// 返回值是监听器和可能的错误
func (ctl *HTTPSGroupController) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (l net.Listener, err error) {
	indexKey := group
	ctl.mu.Lock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		g = NewHTTPSGroup(ctl)
		ctl.groups[indexKey] = g
	}
	ctl.mu.Unlock()

	return g.Listen(ctx, group, groupKey, routeConfig)
}

// RemoveGroup 移除指定的HTTPS组
// 参数group是要移除的组名
func (ctl *HTTPSGroupController) RemoveGroup(group string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	delete(ctl.groups, group)
}

// HTTPSGroup 表示一个HTTPS组
// 每个组包含多个监听器，共享同一个域名和路由配置
type HTTPSGroup struct {
	// group 组名
	group string
	// groupKey 组的密钥，用于验证组的身份
	groupKey string
	// domain 组对应的域名
	domain string

	// acceptCh 用于接收新连接的通道
	acceptCh chan net.Conn
	// httpsLn 实际的HTTPS监听器
	httpsLn *vhost.Listener
	// lns 组中的所有监听器
	lns []*HTTPSGroupListener
	// ctl 组控制器
	ctl *HTTPSGroupController
	// mu 保护组的并发访问
	mu sync.Mutex
}

// NewHTTPSGroup 创建一个新的HTTPS组
// 参数ctl是组控制器
func NewHTTPSGroup(ctl *HTTPSGroupController) *HTTPSGroup {
	return &HTTPSGroup{
		lns:      make([]*HTTPSGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

// Listen 为HTTPS组创建监听器
// 如果是组中的第一个监听器，则创建实际的HTTPS监听器
// 否则，验证组参数并创建新的监听器
// 参数ctx用于控制操作的生命周期
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数routeConfig是路由配置，包含域名和其他路由信息
// 返回值是监听器和可能的错误
func (g *HTTPSGroup) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (ln *HTTPSGroupListener, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.lns) == 0 {
		// 第一个监听器，监听实际地址
		httpsLn, errRet := g.ctl.httpsMuxer.Listen(ctx, &routeConfig)
		if errRet != nil {
			return nil, errRet
		}
		ln = newHTTPSGroupListener(group, g, httpsLn.Addr())

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
		g.httpsLn = httpsLn
		g.lns = append(g.lns, ln)
		go g.worker()
	} else {
		// 同一组的路由配置必须相同
		if g.group != group || g.domain != routeConfig.Domain {
			return nil, ErrGroupParamsInvalid
		}
		if g.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = newHTTPSGroupListener(group, g, g.lns[0].Addr())
		g.lns = append(g.lns, ln)
	}
	return
}

// worker 处理HTTPS连接的协程
// 从实际的HTTPS监听器接收连接，并将其发送到acceptCh通道
func (g *HTTPSGroup) worker() {
	for {
		c, err := g.httpsLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			g.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

// Accept 返回接收新连接的通道
func (g *HTTPSGroup) Accept() <-chan net.Conn {
	return g.acceptCh
}

// CloseListener 关闭组中的一个监听器
// 如果组中没有监听器了，则关闭实际的HTTPS监听器并从控制器中移除组
// 参数ln是要关闭的监听器
func (g *HTTPSGroup) CloseListener(ln *HTTPSGroupListener) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, tmpLn := range g.lns {
		if tmpLn == ln {
			g.lns = append(g.lns[:i], g.lns[i+1:]...)
			break
		}
	}
	if len(g.lns) == 0 {
		close(g.acceptCh)
		if g.httpsLn != nil {
			g.httpsLn.Close()
		}
		g.ctl.RemoveGroup(g.group)
	}
}

// HTTPSGroupListener 表示HTTPS组中的一个监听器
// 每个监听器对应一个客户端的连接请求
type HTTPSGroupListener struct {
	// groupName 组名
	groupName string
	// group 所属的HTTPS组
	group *HTTPSGroup

	// addr 监听器的地址
	addr net.Addr
	// closeCh 用于关闭监听器的通道
	closeCh chan struct{}
}

// newHTTPSGroupListener 创建一个新的HTTPS组监听器
// 参数name是组名
// 参数group是所属的HTTPS组
// 参数addr是监听器的地址
func newHTTPSGroupListener(name string, group *HTTPSGroup, addr net.Addr) *HTTPSGroupListener {
	return &HTTPSGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

// Accept 接收新的连接
// 如果监听器已关闭，则返回错误
// 否则，从组的acceptCh通道接收连接
func (ln *HTTPSGroupListener) Accept() (c net.Conn, err error) {
	var ok bool
	select {
	case <-ln.closeCh:
		return nil, ErrListenerClosed
	case c, ok = <-ln.group.Accept():
		if !ok {
			return nil, ErrListenerClosed
		}
		return c, nil
	}
}

// Addr 返回监听器的地址
func (ln *HTTPSGroupListener) Addr() net.Addr {
	return ln.addr
}

// Close 关闭监听器
// 关闭closeCh通道，并从组中移除自己
func (ln *HTTPSGroupListener) Close() (err error) {
	close(ln.closeCh)

	// 从HTTPS组中移除自己
	ln.group.CloseListener(ln)
	return
}
