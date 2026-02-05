// Copyright 2020 guylewin, guy@lewin.co.il
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
	"fmt"
	"net"
	"sync"

	gerr "github.com/fatedier/golib/errors"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/util/tcpmux"
	"github.com/purpose168/frp/pkg/util/vhost"
)

// TCPMuxGroupCtl 管理所有TCPMux组
type TCPMuxGroupCtl struct {
	// groups 存储所有TCPMux组，键为组名
	groups map[string]*TCPMuxGroup

	// tcpMuxHTTPConnectMuxer 用于管理HTTP Connect类型的TCP多路复用器
	tcpMuxHTTPConnectMuxer *tcpmux.HTTPConnectTCPMuxer
	// mu 保护groups的并发访问
	mu sync.Mutex
}

// NewTCPMuxGroupCtl 创建一个新的TCPMux组控制器
// 参数tcpMuxHTTPConnectMuxer是HTTP Connect类型的TCP多路复用器
func NewTCPMuxGroupCtl(tcpMuxHTTPConnectMuxer *tcpmux.HTTPConnectTCPMuxer) *TCPMuxGroupCtl {
	return &TCPMuxGroupCtl{
		groups:                 make(map[string]*TCPMuxGroup),
		tcpMuxHTTPConnectMuxer: tcpMuxHTTPConnectMuxer,
	}
}

// Listen 为指定的TCPMux组创建监听器
// 如果组不存在，则创建新组
// 参数ctx用于控制操作的生命周期
// 参数multiplexer是多路复用器类型
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数routeConfig是路由配置，包含域名和其他路由信息
// 返回值是监听器和可能的错误
func (tmgc *TCPMuxGroupCtl) Listen(
	ctx context.Context,
	multiplexer, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (l net.Listener, err error) {
	tmgc.mu.Lock()
	tcpMuxGroup, ok := tmgc.groups[group]
	if !ok {
		tcpMuxGroup = NewTCPMuxGroup(tmgc)
		tmgc.groups[group] = tcpMuxGroup
	}
	tmgc.mu.Unlock()

	switch v1.TCPMultiplexerType(multiplexer) {
	case v1.TCPMultiplexerHTTPConnect:
		return tcpMuxGroup.HTTPConnectListen(ctx, group, groupKey, routeConfig)
	default:
		err = fmt.Errorf("未知的多路复用器类型 [%s]", multiplexer)
		return
	}
}

// RemoveGroup 移除指定的TCPMux组
// 参数group是要移除的组名
func (tmgc *TCPMuxGroupCtl) RemoveGroup(group string) {
	tmgc.mu.Lock()
	defer tmgc.mu.Unlock()
	delete(tmgc.groups, group)
}

// TCPMuxGroup 表示一个TCPMux组
// 负责将连接路由到不同的代理
type TCPMuxGroup struct {
	// group 组名
	group string
	// groupKey 组的密钥，用于验证组的身份
	groupKey string
	// domain 组对应的域名
	domain string
	// routeByHTTPUser 按HTTP用户路由的标识
	routeByHTTPUser string
	// username 用户名，用于认证
	username string
	// password 密码，用于认证
	password string

	// acceptCh 用于接收新连接的通道
	acceptCh chan net.Conn
	// tcpMuxLn 实际的TCPMux监听器
	tcpMuxLn net.Listener
	// lns 组中的所有监听器
	lns []*TCPMuxGroupListener
	// ctl 组控制器
	ctl *TCPMuxGroupCtl
	// mu 保护组的并发访问
	mu sync.Mutex
}

// NewTCPMuxGroup 创建一个新的TCPMux组
// 参数ctl是组控制器
func NewTCPMuxGroup(ctl *TCPMuxGroupCtl) *TCPMuxGroup {
	return &TCPMuxGroup{
		lns:      make([]*TCPMuxGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

// HTTPConnectListen 为TCPMux组创建HTTP Connect类型的监听器
// 如果是组中的第一个监听器，则创建实际的TCPMux监听器
// 否则，验证组参数并创建新的监听器
// 参数ctx用于控制操作的生命周期
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数routeConfig是路由配置，包含域名和其他路由信息
// 返回值是监听器和可能的错误
func (tmg *TCPMuxGroup) HTTPConnectListen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (ln *TCPMuxGroupListener, err error) {
	tmg.mu.Lock()
	defer tmg.mu.Unlock()
	if len(tmg.lns) == 0 {
		// 第一个监听器，监听实际地址
		tcpMuxLn, errRet := tmg.ctl.tcpMuxHTTPConnectMuxer.Listen(ctx, &routeConfig)
		if errRet != nil {
			return nil, errRet
		}
		ln = newTCPMuxGroupListener(group, tmg, tcpMuxLn.Addr())

		tmg.group = group
		tmg.groupKey = groupKey
		tmg.domain = routeConfig.Domain
		tmg.routeByHTTPUser = routeConfig.RouteByHTTPUser
		tmg.username = routeConfig.Username
		tmg.password = routeConfig.Password
		tmg.tcpMuxLn = tcpMuxLn
		tmg.lns = append(tmg.lns, ln)
		if tmg.acceptCh == nil {
			tmg.acceptCh = make(chan net.Conn)
		}
		go tmg.worker()
	} else {
		// 同一组的路由配置必须相同
		if tmg.group != group || tmg.domain != routeConfig.Domain ||
			tmg.routeByHTTPUser != routeConfig.RouteByHTTPUser ||
			tmg.username != routeConfig.Username ||
			tmg.password != routeConfig.Password {
			return nil, ErrGroupParamsInvalid
		}
		if tmg.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = newTCPMuxGroupListener(group, tmg, tmg.lns[0].Addr())
		tmg.lns = append(tmg.lns, ln)
	}
	return
}

// worker 当实际的TCPMux监听器创建后被调用
// 从实际的TCPMux监听器接收连接，并将其发送到acceptCh通道
func (tmg *TCPMuxGroup) worker() {
	for {
		c, err := tmg.tcpMuxLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			tmg.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

// Accept 返回接收新连接的通道
func (tmg *TCPMuxGroup) Accept() <-chan net.Conn {
	return tmg.acceptCh
}

// CloseListener 关闭组中的一个监听器
// 如果组中没有监听器了，则关闭实际的TCPMux监听器并从控制器中移除组
// 参数ln是要关闭的监听器
func (tmg *TCPMuxGroup) CloseListener(ln *TCPMuxGroupListener) {
	tmg.mu.Lock()
	defer tmg.mu.Unlock()
	for i, tmpLn := range tmg.lns {
		if tmpLn == ln {
			tmg.lns = append(tmg.lns[:i], tmg.lns[i+1:]...)
			break
		}
	}
	if len(tmg.lns) == 0 {
		close(tmg.acceptCh)
		tmg.tcpMuxLn.Close()
		tmg.ctl.RemoveGroup(tmg.group)
	}
}

// TCPMuxGroupListener 表示TCPMux组中的一个监听器
type TCPMuxGroupListener struct {
	// groupName 组名
	groupName string
	// group 所属的TCPMux组
	group *TCPMuxGroup

	// addr 监听器的地址
	addr net.Addr
	// closeCh 用于关闭监听器的通道
	closeCh chan struct{}
}

// newTCPMuxGroupListener 创建一个新的TCPMux组监听器
// 参数name是组名
// 参数group是所属的TCPMux组
// 参数addr是监听器的地址
func newTCPMuxGroupListener(name string, group *TCPMuxGroup, addr net.Addr) *TCPMuxGroupListener {
	return &TCPMuxGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

// Accept 从TCPMux组接收新的连接
// 如果监听器已关闭，则返回错误
// 否则，从组的acceptCh通道接收连接
func (ln *TCPMuxGroupListener) Accept() (c net.Conn, err error) {
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
func (ln *TCPMuxGroupListener) Addr() net.Addr {
	return ln.addr
}

// Close 关闭监听器
// 关闭closeCh通道，并从组中移除自己
func (ln *TCPMuxGroupListener) Close() (err error) {
	close(ln.closeCh)

	// 从TCPMux组中移除自己
	ln.group.CloseListener(ln)
	return
}
