// Copyright 2019 fatedier, fatedier@gmail.com
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

package server

import (
	"github.com/purpose168/frp/pkg/msg"
)

// Request 请求结构
type Request struct {
	// Version 版本
	Version string `json:"version"`
	// Op 操作
	Op string `json:"op"`
	// Content 内容
	Content any `json:"content"`
}

// Response 响应结构
type Response struct {
	// Reject 是否拒绝
	Reject bool `json:"reject"`
	// RejectReason 拒绝原因
	RejectReason string `json:"reject_reason"`
	// Unchange 是否未改变
	Unchange bool `json:"unchange"`
	// Content 内容
	Content any `json:"content"`
}

// LoginContent 登录内容
type LoginContent struct {
	msg.Login

	// ClientAddress 客户端地址
	ClientAddress string `json:"client_address,omitempty"`
}

// UserInfo 用户信息
type UserInfo struct {
	// User 用户名
	User string `json:"user"`
	// Metas 元数据
	Metas map[string]string `json:"metas"`
	// RunID 运行ID
	RunID string `json:"run_id"`
}

// NewProxyContent 新建代理内容
type NewProxyContent struct {
	// User 用户信息
	User UserInfo `json:"user"`
	msg.NewProxy
}

// CloseProxyContent 关闭代理内容
type CloseProxyContent struct {
	// User 用户信息
	User UserInfo `json:"user"`
	msg.CloseProxy
}

// PingContent 心跳内容
type PingContent struct {
	// User 用户信息
	User UserInfo `json:"user"`
	msg.Ping
}

// NewWorkConnContent 新建工作连接内容
type NewWorkConnContent struct {
	// User 用户信息
	User UserInfo `json:"user"`
	msg.NewWorkConn
}

// NewUserConnContent 新建用户连接内容
type NewUserConnContent struct {
	// User 用户信息
	User UserInfo `json:"user"`
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name"`
	// ProxyType 代理类型
	ProxyType string `json:"proxy_type"`
	// RemoteAddr 远程地址
	RemoteAddr string `json:"remote_addr"`
}
