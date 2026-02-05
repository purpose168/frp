// Copyright 2023 The frp Authors
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

package net

import (
	"context"
	"net"
)

// SetDefaultDNSAddress 设置默认 DNS 服务器地址
// 参数 dnsAddress 是 DNS 服务器地址
func SetDefaultDNSAddress(dnsAddress string) {
	if _, _, err := net.SplitHostPort(dnsAddress); err != nil {
		dnsAddress = net.JoinHostPort(dnsAddress, "53")
	}
	// 更改默认 DNS 服务器
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return net.Dial(network, dnsAddress)
		},
	}
}
