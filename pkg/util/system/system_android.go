// Copyright 2024 The frp Authors
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

package system

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"
)

// EnableCompatibilityMode 启用兼容模式，用于修复 Android 设备上的特定问题
func EnableCompatibilityMode() {
	// 修复时区问题
	fixTimezone()
	// 修复 DNS 解析器问题
	fixDNSResolver()
}

// fixTimezone 用于尝试修复某些 Android 设备上的时区问题
func fixTimezone() {
	// 通过 Android 系统属性获取时区设置
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	// 加载获取到的时区位置信息
	loc, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	// 设置本地时区
	time.Local = loc
}

// fixDNSResolver 首先尝试解析 google.com 以检查当前 DNS 是否可用
// 如果不可用，则默认使用 8.8.8.8 作为 DNS 服务器
// 这是针对 Go 语言在 Android 上无法获取默认 DNS 服务器问题的解决方案
func fixDNSResolver() {
	// 首先，尝试解析域名。如果解析成功，则无需进行修改
	// 在实际场景中，用户可能已经配置了 /etc/resolv.conf，或者直接在 Android 环境中编译
	// 而不是使用交叉编译，因此不会出现此问题
	if net.DefaultResolver != nil {
		// 设置超时上下文，避免长时间阻塞
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// 尝试解析 google.com 域名
		_, err := net.DefaultResolver.LookupHost(timeoutCtx, "google.com")
		if err == nil {
			return
		}
	}
	// 如果解析失败，则使用 8.8.8.8 作为 DNS 服务器
	// 注意：如果有其他方法可以获取默认 DNS 服务器，应优先使用默认 DNS 服务器
	net.DefaultResolver = &net.Resolver{
		// 优先使用 Go 原生解析器
		PreferGo: true,
		// 自定义拨号函数，用于替换默认的 DNS 服务器地址
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 如果地址是本地回环地址，则替换为 Google 公共 DNS 服务器
			if addr == "127.0.0.1:53" || addr == "[::1]:53" {
				addr = "8.8.8.8:53"
			}
			// 创建拨号器并建立连接
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
	}
}
