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

//go:build !darwin && !linux

package vnet

import (
	"context"
	"fmt"
	"runtime"

	"golang.zx2c4.com/wireguard/tun"
)

// openTun 在不支持的平台上打开 TUN 设备
// 此函数仅在非 Darwin 和非 Linux 系统上编译
// ctx: 上下文（未使用）
// addr: 地址字符串（未使用）
// 返回 nil 和错误信息
func openTun(_ context.Context, _ string) (tun.Device, error) {
	return nil, fmt.Errorf("虚拟网络在此平台 (%s/%s) 上不支持", runtime.GOOS, runtime.GOARCH)
}
