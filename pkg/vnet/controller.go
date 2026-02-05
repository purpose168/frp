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

package vnet

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/golib/pool"
	"github.com/songgao/water/waterutil"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/util/log"
	"github.com/purpose168/frp/pkg/util/xlog"
)

const (
	maxPacketSize = 1420 // 最大数据包大小
)

// Controller 虚拟网络控制器，负责管理 TUN 设备和路由
type Controller struct {
	addr string

	tun          io.ReadWriteCloser // TUN 设备接口
	clientRouter *clientRouter      // 基于目标 IP 路由（客户端模式）
	serverRouter *serverRouter      // 基于源 IP 路由（服务器模式）
}

// NewController 创建一个新的虚拟网络控制器
// cfg: 虚拟网络配置，包含地址等信息
// 返回创建的控制器实例
func NewController(cfg v1.VirtualNetConfig) *Controller {
	return &Controller{
		addr:         cfg.Address,
		clientRouter: newClientRouter(),
		serverRouter: newServerRouter(),
	}
}

// Init 初始化控制器，打开 TUN 设备
// 返回可能的错误
func (c *Controller) Init() error {
	// 打开 TUN 设备
	tunDevice, err := OpenTun(context.Background(), c.addr)
	if err != nil {
		return err
	}
	c.tun = tunDevice
	return nil
}

// Run 运行控制器主循环，从 TUN 设备读取数据包并处理
// 返回可能的错误
func (c *Controller) Run() error {
	conn := c.tun

	for {
		// 从池中获取缓冲区
		buf := pool.GetBuf(maxPacketSize)
		n, err := conn.Read(buf)
		if err != nil {
			pool.PutBuf(buf)
			log.Warnf("vnet 从 tun 设备读取错误: %v", err)
			return err
		}

		// 处理数据包
		c.handlePacket(buf[:n])
		pool.PutBuf(buf)
	}
}

// handlePacket 处理单个数据包的路由和转发
// 调用方负责管理缓冲区
func (c *Controller) handlePacket(buf []byte) {
	// 记录收到的数据包
	log.Tracef("vnet 从 tun 读取 [%d]: %s", len(buf), base64.StdEncoding.EncodeToString(buf))

	var src, dst net.IP // 源 IP 和目标 IP

	// 根据 IP 版本进行不同的处理
	switch {
	case waterutil.IsIPv4(buf):
		// 解析 IPv4 头
		header, err := ipv4.ParseHeader(buf)
		if err != nil {
			log.Warnf("解析 IPv4 头错误: %v", err)
			return
		}
		src = header.Src
		dst = header.Dst
		log.Tracef("%s >> %s %d/%-4d %-4x %d",
			header.Src, header.Dst,
			header.Len, header.TotalLen, header.ID, header.Flags)
	case waterutil.IsIPv6(buf):
		// 解析 IPv6 头
		header, err := ipv6.ParseHeader(buf)
		if err != nil {
			log.Warnf("解析 IPv6 头错误: %v", err)
			return
		}
		src = header.Src
		dst = header.Dst
		log.Tracef("%s >> %s %d %d",
			header.Src, header.Dst,
			header.PayloadLen, header.TrafficClass)
	default:
		log.Tracef("未知数据包，已丢弃(%d)", len(buf))
		return
	}

	// 首先尝试根据目标 IP 查找客户端连接
	targetConn, err := c.clientRouter.findConn(dst)
	if err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("写入客户端目标连接错误: %v", err)
		}
		return
	}

	// 如果客户端路由没有找到，尝试服务器端路由
	targetConn, err = c.serverRouter.findConnBySrc(dst)
	if err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("写入服务器目标连接错误: %v", err)
		}
		return
	}

	log.Tracef("没有找到从 %s 到 %s 的数据包路由", src, dst)
}

// Stop 停止控制器，关闭 TUN 设备
// 返回可能的错误
func (c *Controller) Stop() error {
	return c.tun.Close()
}

// readLoopClient 客户端连接读取循环
// ctx: 上下文，用于获取日志记录器
// conn: 要读取的连接
func (c *Controller) readLoopClient(ctx context.Context, conn io.ReadWriteCloser) {
	// 从上下文获取日志记录器
	xl := xlog.FromContextSafe(ctx)
	defer func() {
		// 读取循环结束时移除路由（连接关闭）
		c.clientRouter.removeConnRoute(conn)
		conn.Close()
	}()

	for {
		// 读取消息
		data, err := ReadMessage(conn)
		if err != nil {
			xl.Warnf("客户端读取错误: %v", err)
			return
		}

		// 忽略空数据
		if len(data) == 0 {
			continue
		}

		// 根据 IP 版本解析头部并记录
		switch {
		case waterutil.IsIPv4(data):
			header, err := ipv4.ParseHeader(data)
			if err != nil {
				xl.Warnf("解析 IPv4 头错误: %v", err)
				continue
			}
			xl.Tracef("%s >> %s %d/%-4d %-4x %d",
				header.Src, header.Dst,
				header.Len, header.TotalLen, header.ID, header.Flags)
		case waterutil.IsIPv6(data):
			header, err := ipv6.ParseHeader(data)
			if err != nil {
				xl.Warnf("解析 IPv6 头错误: %v", err)
				continue
			}
			xl.Tracef("%s >> %s %d %d",
				header.Src, header.Dst,
				header.PayloadLen, header.TrafficClass)
		default:
			xl.Tracef("未知数据包，已丢弃(%d)", len(data))
			continue
		}

		// 写入 TUN 设备
		xl.Tracef("vnet 写入 tun (客户端) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		_, err = c.tun.Write(data)
		if err != nil {
			xl.Warnf("客户端写入 tun 错误: %v", err)
		}
	}
}

// readLoopServer 服务器连接读取循环
// ctx: 上下文，用于获取日志记录器
// conn: 要读取的连接
// onClose: 连接关闭时的回调函数
func (c *Controller) readLoopServer(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	xl := xlog.FromContextSafe(ctx)
	defer func() {
		// 连接关闭时清理所有关联的 IP 映射
		c.serverRouter.cleanupConnIPs(conn)
		// 调用关闭回调
		if onClose != nil {
			onClose()
		}
		conn.Close()
	}()

	for {
		data, err := ReadMessage(conn)
		if err != nil {
			xl.Warnf("服务器读取错误: %v", err)
			return
		}

		// 忽略空数据
		if len(data) == 0 {
			continue
		}

		// 注册源 IP 到连接的映射
		if waterutil.IsIPv4(data) || waterutil.IsIPv6(data) {
			var src net.IP
			if waterutil.IsIPv4(data) {
				header, err := ipv4.ParseHeader(data)
				if err == nil {
					src = header.Src
					c.serverRouter.registerSrcIP(src, conn)
				}
			} else {
				header, err := ipv6.ParseHeader(data)
				if err == nil {
					src = header.Src
					c.serverRouter.registerSrcIP(src, conn)
				}
			}
		}

		xl.Tracef("vnet 写入 tun (服务器) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		_, err = c.tun.Write(data)
		if err != nil {
			xl.Warnf("服务器写入 tun 错误: %v", err)
		}
	}
}

// RegisterClientRoute 注册客户端路由（基于目标 IP CIDR）
// 并启动读取循环
// ctx: 上下文
// name: 路由名称
// routes: IP 网络列表
// conn: 关联的连接
func (c *Controller) RegisterClientRoute(ctx context.Context, name string, routes []net.IPNet, conn io.ReadWriteCloser) {
	c.clientRouter.addRoute(name, routes, conn)
	go c.readLoopClient(ctx, conn)
}

// UnregisterClientRoute 从路由表中移除客户端路由
// name: 要移除的路由名称
func (c *Controller) UnregisterClientRoute(name string) {
	c.clientRouter.delRoute(name)
}

// StartServerConnReadLoop 启动服务器连接的读取循环
// （动态关联源 IP）
// ctx: 上下文
// conn: 要读取的连接
// onClose: 连接关闭时的回调函数
func (c *Controller) StartServerConnReadLoop(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	go c.readLoopServer(ctx, conn, onClose)
}

// ParseRoutes 将路由字符串转换为 IPNet 对象
// routeStrings: 路由字符串数组
// 返回 IPNet 数组和可能的错误
func ParseRoutes(routeStrings []string) ([]net.IPNet, error) {
	routes := make([]net.IPNet, 0, len(routeStrings))
	for _, r := range routeStrings {
		_, ipNet, err := net.ParseCIDR(r)
		if err != nil {
			return nil, fmt.Errorf("解析路由 %s 错误: %v", r, err)
		}
		routes = append(routes, *ipNet)
	}
	return routes, nil
}

// Client router（基于目标 IP 路由）
type clientRouter struct {
	routes map[string]*routeElement // 路由映射表
	mu     sync.RWMutex             // 读写锁
}

func newClientRouter() *clientRouter {
	return &clientRouter{
		routes: make(map[string]*routeElement),
	}
}

// addRoute 添加客户端路由
// name: 路由名称
// routes: IP 网络列表
// conn: 关联的连接
func (r *clientRouter) addRoute(name string, routes []net.IPNet, conn io.ReadWriteCloser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[name] = &routeElement{
		name:   name,
		routes: routes,
		conn:   conn,
	}
}

// findConn 根据目标 IP 查找连接
// dst: 目标 IP 地址
// 返回找到的连接和可能的错误
func (r *clientRouter) findConn(dst net.IP) (io.Writer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, re := range r.routes {
		for _, route := range re.routes {
			// 检查目标 IP 是否在路由范围内
			if route.Contains(dst) {
				return re.conn, nil
			}
		}
	}
	return nil, fmt.Errorf("没有找到目标 %s 的路由", dst)
}

// delRoute 删除指定名称的路由
// name: 要删除的路由名称
func (r *clientRouter) delRoute(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, name)
}

// removeConnRoute 移除与指定连接关联的路由
// conn: 要移除的连接
func (r *clientRouter) removeConnRoute(conn io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, re := range r.routes {
		if re.conn == conn {
			delete(r.routes, name)
			return
		}
	}
}

// Server router（仅基于源 IP 路由）
type serverRouter struct {
	srcIPConns map[string]io.Writer // 源 IP 字符串到连接的映射
	mu         sync.RWMutex         // 读写锁
}

func newServerRouter() *serverRouter {
	return &serverRouter{
		srcIPConns: make(map[string]io.Writer),
	}
}

// findConnBySrc 根据源 IP 查找连接
// src: 源 IP 地址
// 返回找到的连接和可能的错误
func (r *serverRouter) findConnBySrc(src net.IP) (io.Writer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, exists := r.srcIPConns[src.String()]
	if !exists {
		return nil, fmt.Errorf("没有找到源 %s 的路由", src)
	}
	return conn, nil
}

// registerSrcIP 注册源 IP 到连接的映射
// src: 源 IP 地址
// conn: 关联的连接
func (r *serverRouter) registerSrcIP(src net.IP, conn io.Writer) {
	key := src.String()

	r.mu.RLock()
	existingConn, ok := r.srcIPConns[key]
	r.mu.RUnlock()

	// 如果条目存在且连接相同，则无需执行任何操作
	if ok && existingConn == conn {
		return
	}

	// 获取写锁以更新映射
	r.mu.Lock()
	defer r.mu.Unlock()

	// 获取写锁后再次检查，以处理潜在的竞态条件
	existingConn, ok = r.srcIPConns[key]
	if ok && existingConn == conn {
		return
	}

	r.srcIPConns[key] = conn
}

// cleanupConnIPs 移除与指定连接关联的所有 IP 映射
// conn: 要清理的连接
func (r *serverRouter) cleanupConnIPs(conn io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 查找并删除所有指向此连接的 IP 映射
	for ip, mappedConn := range r.srcIPConns {
		if mappedConn == conn {
			delete(r.srcIPConns, ip)
		}
	}
}

// routeElement 路由元素，存储路由名称、IP 网络列表和关联的连接
type routeElement struct {
	name   string             // 路由名称
	routes []net.IPNet        // IP 网络列表
	conn   io.ReadWriteCloser // 关联的连接
}
