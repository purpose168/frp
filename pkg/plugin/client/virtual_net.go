// Copyright 2025 The frp Authors
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

//go:build !frps

package client

import (
	"context"
	"io"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	Register(v1.PluginVirtualNet, NewVirtualNetPlugin)
}

// VirtualNetPlugin 虚拟网络插件
type VirtualNetPlugin struct {
	// pluginCtx 插件上下文
	pluginCtx PluginContext
	// opts 插件选项
	opts *v1.VirtualNetPluginOptions
	// mu 互斥锁
	mu sync.Mutex
	// conns 连接映射
	conns map[io.ReadWriteCloser]struct{}
}

// NewVirtualNetPlugin 创建虚拟网络插件
func NewVirtualNetPlugin(pluginCtx PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.VirtualNetPluginOptions)

	p := &VirtualNetPlugin{
		pluginCtx: pluginCtx,
		opts:      opts,
	}
	return p, nil
}

// Handle 处理连接
func (p *VirtualNetPlugin) Handle(ctx context.Context, connInfo *ConnectionInfo) {
	// 验证虚拟网络控制器是否可用
	if p.pluginCtx.VnetController == nil {
		return
	}

	// 在启动读取循环之前添加连接，以避免竞态条件
	// 即RemoveConn可能在连接添加之前被调用
	p.mu.Lock()
	if p.conns == nil {
		p.conns = make(map[io.ReadWriteCloser]struct{})
	}
	p.conns[connInfo.Conn] = struct{}{}
	p.mu.Unlock()

	// 向控制器注册连接并传递清理函数
	p.pluginCtx.VnetController.StartServerConnReadLoop(ctx, connInfo.Conn, func() {
		p.RemoveConn(connInfo.Conn)
	})
}

// RemoveConn 移除连接
func (p *VirtualNetPlugin) RemoveConn(conn io.ReadWriteCloser) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 检查映射是否存在，因为Close可能并发地将其设置为nil
	if p.conns != nil {
		delete(p.conns, conn)
	}
}

// Name 返回插件名称
func (p *VirtualNetPlugin) Name() string {
	return v1.PluginVirtualNet
}

// Close 关闭插件
func (p *VirtualNetPlugin) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 关闭所有剩余的连接
	for conn := range p.conns {
		_ = conn.Close()
	}
	p.conns = nil
	return nil
}
