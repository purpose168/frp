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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	baseTunName = "utun" // 基础 TUN 设备名称
	defaultMTU  = 1420   // 默认 MTU 值
)

// openTun 在 Linux 系统上打开 TUN 设备
// ctx: 上下文（未使用）
// addr: 地址字符串，格式为 CIDR
// 返回创建的 TUN 设备和可能的错误
func openTun(_ context.Context, addr string) (tun.Device, error) {
	// 查找下一个可用的 TUN 设备名称
	name, err := findNextTunName(baseTunName)
	if err != nil {
		// 如果失败，使用基于地址的哈希值作为备用名称
		name = getFallbackTunName(baseTunName, addr)
	}

	// 创建 TUN 设备
	tunDevice, err := tun.CreateTUN(name, defaultMTU)
	if err != nil {
		return nil, fmt.Errorf("创建 TUN 设备 '%s' 失败: %w", name, err)
	}

	// 获取实际设备名称
	actualName, err := tunDevice.Name()
	if err != nil {
		return nil, err
	}

	// 获取网络接口
	ifn, err := net.InterfaceByName(actualName)
	if err != nil {
		return nil, err
	}

	// 获取网络接口的 link
	link, err := netlink.LinkByName(actualName)
	if err != nil {
		return nil, err
	}

	// 解析地址
	ip, cidr, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	// 添加地址到接口
	if err := netlink.AddrAdd(link, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: cidr.Mask,
		},
	}); err != nil {
		return nil, err
	}

	// 启用接口
	if err := netlink.LinkSetUp(link); err != nil {
		return nil, err
	}

	// 添加路由
	if err = addRoutes(ifn, cidr); err != nil {
		return nil, err
	}
	return tunDevice, nil
}

// findNextTunName 查找下一个可用的 TUN 设备名称
// basename: 基础名称
// 返回下一个可用的设备名称和可能的错误
func findNextTunName(basename string) (string, error) {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("获取网络接口失败: %w", err)
	}
	maxSuffix := -1

	// 查找最大的后缀数字
	for _, iface := range interfaces {
		name := iface.Name
		if strings.HasPrefix(name, basename) {
			suffix := name[len(basename):]
			if suffix == "" {
				continue
			}

			numSuffix, err := strconv.Atoi(suffix)
			if err == nil && numSuffix > maxSuffix {
				maxSuffix = numSuffix
			}
		}
	}

	// 生成下一个名称
	nextSuffix := maxSuffix + 1
	name := fmt.Sprintf("%s%d", basename, nextSuffix)
	return name, nil
}

// addRoutes 添加系统路由
// ifn: 网络接口
// cidr: IP 网络
// 返回可能的错误
func addRoutes(ifn *net.Interface, cidr *net.IPNet) error {
	r := netlink.Route{
		Dst:       cidr,
		LinkIndex: ifn.Index,
	}
	if err := netlink.RouteReplace(&r); err != nil {
		return fmt.Errorf("添加路由到 %v 失败: %v", r.Dst, err)
	}
	return nil
}

// getFallbackTunName 基于基础名称和地址生成确定的备用 TUN 设备名称
// 使用哈希算法生成
// baseName: 基础名称
// addr: 地址字符串
// 返回生成的设备名称
func getFallbackTunName(baseName, addr string) string {
	hasher := sha256.New()
	hasher.Write([]byte(addr))
	hashBytes := hasher.Sum(nil)
	// 使用前 4 个字节生成 8 个十六进制字符，简洁且遵循 IFNAMSIZ 限制
	shortHash := hex.EncodeToString(hashBytes[:4])
	return fmt.Sprintf("%s%s", baseName, shortHash)
}
