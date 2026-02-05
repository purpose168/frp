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

package visitor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netutil "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	Register(v1.VisitorPluginVirtualNet, NewVirtualNetPlugin)
}

// VirtualNetPlugin 虚拟网络插件
type VirtualNetPlugin struct {
	// pluginCtx 插件上下文
	pluginCtx PluginContext

	// routes 路由列表
	routes []net.IPNet

	// mu 互斥锁
	mu sync.Mutex
	// controllerConn 控制器连接
	controllerConn net.Conn
	// closeSignal 关闭信号通道
	closeSignal chan struct{}
	// consecutiveErrors 连续错误计数，用于指数退避
	consecutiveErrors int
	// ctx 上下文
	ctx context.Context
	// cancel 取消函数
	cancel context.CancelFunc
}

// NewVirtualNetPlugin 创建虚拟网络插件
func NewVirtualNetPlugin(pluginCtx PluginContext, options v1.VisitorPluginOptions) (Plugin, error) {
	opts := options.(*v1.VirtualNetVisitorPluginOptions)
	p := &VirtualNetPlugin{
		pluginCtx: pluginCtx,
		routes:    make([]net.IPNet, 0),
	}
	p.ctx, p.cancel = context.WithCancel(pluginCtx.Ctx)

	if opts.DestinationIP == "" {
		return nil, errors.New("目标IP地址是必需的")
	}

	// 解析DestinationIP并创建主机路由
	ip := net.ParseIP(opts.DestinationIP)
	if ip == nil {
		return nil, fmt.Errorf("无效的目标IP地址 [%s]", opts.DestinationIP)
	}

	var mask net.IPMask
	if ip.To4() != nil {
		mask = net.CIDRMask(32, 32) // IPv4使用/32
	} else {
		mask = net.CIDRMask(128, 128) // IPv6使用/128
	}
	p.routes = append(p.routes, net.IPNet{IP: ip, Mask: mask})
	return p, nil
}

// Name 返回插件名称
func (p *VirtualNetPlugin) Name() string {
	return v1.VisitorPluginVirtualNet
}

// Start 启动插件
func (p *VirtualNetPlugin) Start() {
	xl := xlog.FromContextSafe(p.pluginCtx.Ctx)
	if p.pluginCtx.VnetController == nil {
		return
	}

	routeStr := "unknown"
	if len(p.routes) > 0 {
		routeStr = p.routes[0].String()
	}
	xl.Infof("正在为访问者 [%s] 启动 VirtualNetPlugin，尝试注册路由 %s", p.pluginCtx.Name, routeStr)
	go p.run()
}

// run 运行插件主循环
func (p *VirtualNetPlugin) run() {
	xl := xlog.FromContextSafe(p.ctx)
	for {
		currentCloseSignal := make(chan struct{})
		p.mu.Lock()
		p.closeSignal = currentCloseSignal
		p.mu.Unlock()

		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin 运行循环为访问者 [%s] 停止（在管道创建前上下文已取消）。", p.pluginCtx.Name)
			p.cleanupControllerConn(xl)
			return
		default:
		}

		controllerConn, pluginConn := net.Pipe()

		p.mu.Lock()
		p.controllerConn = controllerConn
		p.mu.Unlock()

		// 使用CloseNotifyConn包装，支持关闭通知和错误记录
		var closeErr error
		pluginNotifyConn := netutil.WrapCloseNotifyConn(pluginConn, func(err error) {
			closeErr = err
			close(currentCloseSignal) // 通知运行循环关闭
		})

		xl.Infof("正在尝试为访问者 [%s] 注册客户端路由", p.pluginCtx.Name)
		p.pluginCtx.VnetController.RegisterClientRoute(p.ctx, p.pluginCtx.Name, p.routes, controllerConn)
		xl.Infof("成功为访问者 [%s] 注册客户端路由。正在启动连接处理器，使用 CloseNotifyConn。", p.pluginCtx.Name)
		// 将CloseNotifyConn传递给访问者处理
		// 访问者可以调用CloseWithError来记录失败原因
		p.pluginCtx.SendConnToVisitor(pluginNotifyConn)

		// 等待上下文取消或连接关闭
		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin 运行循环为访问者 [%s] 停止（在等待时上下文已取消）。", p.pluginCtx.Name)
			p.cleanupControllerConn(xl)
			return
		case <-currentCloseSignal:
			// 根据错误确定重连延迟，使用指数退避
			var reconnectDelay time.Duration
			if closeErr != nil {
				p.consecutiveErrors++
				xl.Warnf("访问者 [%s] 的连接因错误关闭（连续错误：%d）：%v",
					p.pluginCtx.Name, p.consecutiveErrors, closeErr)
				// 指数退避：60s、120s、240s、300s（上限）
				baseDelay := 60 * time.Second
				reconnectDelay = baseDelay * time.Duration(1<<uint(p.consecutiveErrors-1))
				if reconnectDelay > 300*time.Second {
					reconnectDelay = 300 * time.Second
				}
			} else {
				// 成功连接时重置连续错误计数
				if p.consecutiveErrors > 0 {
					xl.Infof("访问者 [%s] 的连接正常关闭，重置错误计数（之前为 %d）",
						p.pluginCtx.Name, p.consecutiveErrors)
					p.consecutiveErrors = 0
				} else {
					xl.Infof("访问者 [%s] 的连接正常关闭", p.pluginCtx.Name)
				}
				reconnectDelay = 10 * time.Second
			}

			// 访问者关闭了插件端。关闭控制器端
			p.cleanupControllerConn(xl)

			xl.Infof("正在等待 %v 后尝试为访问者 [%s] 重新连接...", reconnectDelay, p.pluginCtx.Name)
			select {
			case <-time.After(reconnectDelay):
			case <-p.ctx.Done():
				xl.Infof("访问者 [%s] 的重连延迟被中断", p.pluginCtx.Name)
				return
			}
		}

		xl.Infof("正在为访问者 [%s] 重新建立虚拟连接...", p.pluginCtx.Name)
	}
}

// cleanupControllerConn 关闭当前的controllerConn（如果存在），在锁保护下
func (p *VirtualNetPlugin) cleanupControllerConn(xl *xlog.Logger) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.controllerConn != nil {
		xl.Debugf("正在为访问者 [%s] 清理 controllerConn", p.pluginCtx.Name)
		p.controllerConn.Close()
		p.controllerConn = nil
	}
	p.closeSignal = nil
}

// Close 启动插件关闭
func (p *VirtualNetPlugin) Close() error {
	xl := xlog.FromContextSafe(p.pluginCtx.Ctx)
	xl.Infof("正在为访问者 [%s] 关闭 VirtualNetPlugin", p.pluginCtx.Name)

	// 通知运行循环goroutine停止
	p.cancel()

	// 从控制器注销路由
	if p.pluginCtx.VnetController != nil {
		p.pluginCtx.VnetController.UnregisterClientRoute(p.pluginCtx.Name)
		xl.Infof("已为访问者 [%s] 注销客户端路由", p.pluginCtx.Name)
	}

	// 显式关闭管道的控制器端
	// 这确保即使运行循环卡住或访问者未关闭其端，管道也会断开
	p.cleanupControllerConn(xl)
	xl.Infof("已为访问者 [%s] 清理连接", p.pluginCtx.Name)

	return nil
}
