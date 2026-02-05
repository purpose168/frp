// Copyright 2019 fatedier, fatedier@gmail.com
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

package visitor

import (
	"fmt"
	"io"
	"net"
	"slices"
	"sync"

	libio "github.com/fatedier/golib/io"

	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/util"
)

// listenerBundle 监听器捆绑包
type listenerBundle struct {
	// l 内部监听器
	l *netpkg.InternalListener
	// sk 密钥
	sk string
	// allowUsers 允许的用户列表
	allowUsers []string
}

// Manager 访问者监听器管理器
type Manager struct {
	// listeners 监听器映射
	listeners map[string]*listenerBundle

	// mu 读写锁
	mu sync.RWMutex
}

// NewManager 创建一个新的访问者管理器
func NewManager() *Manager {
	return &Manager{
		listeners: make(map[string]*listenerBundle),
	}
}

// Listen 为访问者创建一个新的监听器
// 参数name是监听器名称
// 参数sk是密钥
// 参数allowUsers是允许的用户列表
// 返回值是内部监听器和可能的错误
func (vm *Manager) Listen(name string, sk string, allowUsers []string) (*netpkg.InternalListener, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, ok := vm.listeners[name]; ok {
		return nil, fmt.Errorf("[%s]的自定义监听器重复", name)
	}

	l := netpkg.NewInternalListener()
	vm.listeners[name] = &listenerBundle{
		l:          l,
		sk:         sk,
		allowUsers: allowUsers,
	}
	return l, nil
}

// NewConn 为访问者创建一个新的连接
// 参数name是监听器名称
// 参数conn是网络连接
// 参数timestamp是时间戳
// 参数signKey是签名密钥
// 参数useEncryption是否使用加密
// 参数useCompression是否使用压缩
// 参数visitorUser是访问者用户名
// 返回值是可能的错误
func (vm *Manager) NewConn(name string, conn net.Conn, timestamp int64, signKey string,
	useEncryption bool, useCompression bool, visitorUser string,
) (err error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if l, ok := vm.listeners[name]; ok {
		if util.GetAuthKey(l.sk, timestamp) != signKey {
			err = fmt.Errorf("[%s]的访问者连接认证失败", name)
			return
		}

		if !slices.Contains(l.allowUsers, visitorUser) && !slices.Contains(l.allowUsers, "*") {
			err = fmt.Errorf("[%s]的访问者连接用户 [%s] 不被允许", name, visitorUser)
			return
		}

		var rwc io.ReadWriteCloser = conn
		if useEncryption {
			if rwc, err = libio.WithEncryption(rwc, []byte(l.sk)); err != nil {
				err = fmt.Errorf("创建加密连接失败: %v", err)
				return
			}
		}
		if useCompression {
			rwc = libio.WithCompression(rwc)
		}
		err = l.l.PutConn(netpkg.WrapReadWriteCloserToConn(rwc, conn))
	} else {
		err = fmt.Errorf("[%s]的自定义监听器不存在", name)
		return
	}
	return
}

// CloseListener 关闭指定名称的监听器
// 参数name是监听器名称
func (vm *Manager) CloseListener(name string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	delete(vm.listeners, name)
}
