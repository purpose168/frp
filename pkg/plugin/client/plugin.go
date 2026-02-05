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

package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/golib/errors"
	pp "github.com/pires/go-proxyproto"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/vnet"
)

// PluginContext 插件上下文
type PluginContext struct {
	// Name 名称
	Name string
	// VnetController 虚拟网络控制器
	VnetController *vnet.Controller
}

// Creators 用于创建插件以处理连接
var creators = make(map[string]CreatorFn)

// CreatorFn 创建插件的函数类型
type CreatorFn func(pluginCtx PluginContext, options v1.ClientPluginOptions) (Plugin, error)

// Register 注册插件创建函数
func Register(name string, fn CreatorFn) {
	if _, exist := creators[name]; exist {
		panic(fmt.Sprintf("plugin [%s] is already registered", name))
	}
	creators[name] = fn
}

// Create 创建插件实例
func Create(pluginName string, pluginCtx PluginContext, options v1.ClientPluginOptions) (p Plugin, err error) {
	if fn, ok := creators[pluginName]; ok {
		p, err = fn(pluginCtx, options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", pluginName)
	}
	return
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	// Conn 连接对象
	Conn io.ReadWriteCloser
	// UnderlyingConn 底层连接
	UnderlyingConn net.Conn

	// ProxyProtocolHeader 代理协议头
	ProxyProtocolHeader *pp.Header
	// SrcAddr 源地址
	SrcAddr net.Addr
	// DstAddr 目标地址
	DstAddr net.Addr
}

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// Handle 处理连接
	Handle(ctx context.Context, connInfo *ConnectionInfo)
	// Close 关闭插件
	Close() error
}

// Listener 监听器
type Listener struct {
	// conns 连接通道
	conns chan net.Conn
	// closed 是否已关闭
	closed bool
	// mu 互斥锁
	mu sync.Mutex
}

// NewProxyListener 创建代理监听器
func NewProxyListener() *Listener {
	return &Listener{
		conns: make(chan net.Conn, 64),
	}
}

// Accept 接受连接
func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

// PutConn 放入连接
func (l *Listener) PutConn(conn net.Conn) error {
	err := errors.PanicToError(func() {
		l.conns <- conn
	})
	return err
}

// Close 关闭监听器
func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		close(l.conns)
		l.closed = true
	}
	return nil
}

// Addr 返回地址
func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}
