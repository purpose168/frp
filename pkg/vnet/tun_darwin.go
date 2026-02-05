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
	"fmt"
	"net"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

const (
	defaultTunName = "utun" // 默认 TUN 设备名称
	defaultMTU     = 1420   // 默认 MTU 值
)

// openTun 在 Darwin/macOS 系统上打开 TUN 设备
// ctx: 上下文（未使用）
// addr: 地址字符串，格式为 CIDR
// 返回创建的 TUN 设备和可能的错误
func openTun(_ context.Context, addr string) (tun.Device, error) {
	// 创建 TUN 设备
	dev, err := tun.CreateTUN(defaultTunName, defaultMTU)
	if err != nil {
		return nil, err
	}

	// 获取设备名称
	name, err := dev.Name()
	if err != nil {
		return nil, err
	}

	// 解析地址
	ip, ipNet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	// 为点对点隧道生成对端 IP
	peerIP := generatePeerIP(ip)

	// 使用 ifconfig 配置接口，设置为点对点寻址
	if err = exec.Command("ifconfig", name, "inet", ip.String(), peerIP.String(), "mtu", fmt.Sprint(defaultMTU), "up").Run(); err != nil {
		return nil, err
	}

	// 添加隧道子网的默认路由
	routes := []net.IPNet{*ipNet}
	if err = addRoutes(name, routes); err != nil {
		return nil, err
	}
	return dev, nil
}

// generatePeerIP 为点对点隧道生成对端 IP
// 通过递增 IP 的最后一个八位组来生成
// ip: 原始 IP 地址
// 返回生成的对端 IP 地址
func generatePeerIP(ip net.IP) net.IP {
	// 复制以避免修改原始 IP
	peerIP := make(net.IP, len(ip))
	copy(peerIP, ip)

	// 递增最后一个八位组
	peerIP[len(peerIP)-1]++

	return peerIP
}

// addRoutes 配置 TUN 接口的系统路由
// ifName: 接口名称
// routes: 要添加的路由列表
// 返回可能的错误
func addRoutes(ifName string, routes []net.IPNet) error {
	for _, route := range routes {
		routeStr := route.String()
		// 使用 route 命令添加路由
		if err := exec.Command("route", "add", "-net", routeStr, "-interface", ifName).Run(); err != nil {
			return err
		}
	}
	return nil
}
