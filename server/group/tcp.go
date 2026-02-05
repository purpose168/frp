// Copyright 2018 fatedier, fatedier@gmail.com
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
	"net"
	"strconv"
	"sync"

	gerr "github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/server/ports"
)

// TCPGroupCtl 管理所有TCP组
type TCPGroupCtl struct {
	// groups 存储所有TCP组，键为组名
	groups map[string]*TCPGroup

	// portManager 用于管理端口的管理器
	portManager *ports.Manager
	// mu 保护groups的并发访问
	mu sync.Mutex
}

// NewTCPGroupCtl 创建一个新的TCP组控制器
// 参数portManager是端口管理器
func NewTCPGroupCtl(portManager *ports.Manager) *TCPGroupCtl {
	return &TCPGroupCtl{
		groups:      make(map[string]*TCPGroup),
		portManager: portManager,
	}
}

// Listen 为指定的TCP组创建监听器
// 如果组不存在，则创建新组
// 参数proxyName是代理名称
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数addr是监听地址
// 参数port是监听端口
// 返回值是监听器、实际监听端口和可能的错误
func (tgc *TCPGroupCtl) Listen(proxyName string, group string, groupKey string,
	addr string, port int,
) (l net.Listener, realPort int, err error) {
	tgc.mu.Lock()
	tcpGroup, ok := tgc.groups[group]
	if !ok {
		tcpGroup = NewTCPGroup(tgc)
		tgc.groups[group] = tcpGroup
	}
	tgc.mu.Unlock()

	return tcpGroup.Listen(proxyName, group, groupKey, addr, port)
}

// RemoveGroup 移除指定的TCP组
// 参数group是要移除的组名
func (tgc *TCPGroupCtl) RemoveGroup(group string) {
	tgc.mu.Lock()
	defer tgc.mu.Unlock()
	delete(tgc.groups, group)
}

// TCPGroup 表示一个TCP组
// 负责将连接路由到不同的代理
type TCPGroup struct {
	// group 组名
	group string
	// groupKey 组的密钥，用于验证组的身份
	groupKey string
	// addr 监听地址
	addr string
	// port 监听端口
	port int
	// realPort 实际监听的端口
	realPort int

	// acceptCh 用于接收新连接的通道
	acceptCh chan net.Conn
	// tcpLn 实际的TCP监听器
	tcpLn net.Listener
	// lns 组中的所有监听器
	lns []*TCPGroupListener
	// ctl 组控制器
	ctl *TCPGroupCtl
	// mu 保护组的并发访问
	mu sync.Mutex
}

// NewTCPGroup 创建一个新的TCP组
// 参数ctl是组控制器
func NewTCPGroup(ctl *TCPGroupCtl) *TCPGroup {
	return &TCPGroup{
		lns:      make([]*TCPGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

// Listen 为TCP组创建监听器
// 如果TCP组已经有监听器，则只需添加一个新的TCPGroupListener到队列中
// 否则，监听实际的地址
// 参数proxyName是代理名称
// 参数group是组名
// 参数groupKey是组的密钥，用于验证组的身份
// 参数addr是监听地址
// 参数port是监听端口
// 返回值是监听器、实际监听端口和可能的错误
func (tg *TCPGroup) Listen(proxyName string, group string, groupKey string, addr string, port int) (ln *TCPGroupListener, realPort int, err error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if len(tg.lns) == 0 {
		// 第一个监听器，监听实际地址
		realPort, err = tg.ctl.portManager.Acquire(proxyName, port)
		if err != nil {
			return
		}
		tcpLn, errRet := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
		if errRet != nil {
			err = errRet
			return
		}
		ln = newTCPGroupListener(group, tg, tcpLn.Addr())

		tg.group = group
		tg.groupKey = groupKey
		tg.addr = addr
		tg.port = port
		tg.realPort = realPort
		tg.tcpLn = tcpLn
		tg.lns = append(tg.lns, ln)
		if tg.acceptCh == nil {
			tg.acceptCh = make(chan net.Conn)
		}
		go tg.worker()
	} else {
		// 同一组的地址和端口必须相同
		if tg.group != group || tg.addr != addr {
			err = ErrGroupParamsInvalid
			return
		}
		if tg.port != port {
			err = ErrGroupDifferentPort
			return
		}
		if tg.groupKey != groupKey {
			err = ErrGroupAuthFailed
			return
		}
		ln = newTCPGroupListener(group, tg, tg.lns[0].Addr())
		realPort = tg.realPort
		tg.lns = append(tg.lns, ln)
	}
	return
}

// worker 当实际的TCP监听器创建后被调用
// 从实际的TCP监听器接收连接，并将其发送到acceptCh通道
func (tg *TCPGroup) worker() {
	for {
		c, err := tg.tcpLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			tg.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

// Accept 返回接收新连接的通道
func (tg *TCPGroup) Accept() <-chan net.Conn {
	return tg.acceptCh
}

// CloseListener 从TCP组中移除TCPGroupListener
// 如果组中没有监听器了，则关闭实际的TCP监听器并从控制器中移除组
// 参数ln是要关闭的监听器
func (tg *TCPGroup) CloseListener(ln *TCPGroupListener) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	for i, tmpLn := range tg.lns {
		if tmpLn == ln {
			tg.lns = append(tg.lns[:i], tg.lns[i+1:]...)
			break
		}
	}
	if len(tg.lns) == 0 {
		close(tg.acceptCh)
		tg.tcpLn.Close()
		tg.ctl.portManager.Release(tg.realPort)
		tg.ctl.RemoveGroup(tg.group)
	}
}

// TCPGroupListener 表示TCP组中的一个监听器
type TCPGroupListener struct {
	// groupName 组名
	groupName string
	// group 所属的TCP组
	group *TCPGroup

	// addr 监听器的地址
	addr net.Addr
	// closeCh 用于关闭监听器的通道
	closeCh chan struct{}
}

// newTCPGroupListener 创建一个新的TCP组监听器
// 参数name是组名
// 参数group是所属的TCP组
// 参数addr是监听器的地址
func newTCPGroupListener(name string, group *TCPGroup, addr net.Addr) *TCPGroupListener {
	return &TCPGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

// Accept 从TCP组接收新的连接
// 如果监听器已关闭，则返回错误
// 否则，从组的acceptCh通道接收连接
func (ln *TCPGroupListener) Accept() (c net.Conn, err error) {
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
func (ln *TCPGroupListener) Addr() net.Addr {
	return ln.addr
}

// Close 关闭监听器
// 关闭closeCh通道，并从组中移除自己
func (ln *TCPGroupListener) Close() (err error) {
	close(ln.closeCh)

	// 从TCP组中移除自己
	ln.group.CloseListener(ln)
	return
}
