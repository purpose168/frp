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
	"context"
)

const (
	// APIVersion API版本
	APIVersion = "0.1.0"

	// OpLogin 登录操作
	OpLogin = "Login"
	// OpNewProxy 新建代理操作
	OpNewProxy = "NewProxy"
	// OpCloseProxy 关闭代理操作
	OpCloseProxy = "CloseProxy"
	// OpPing 心跳操作
	OpPing = "Ping"
	// OpNewWorkConn 新建工作连接操作
	OpNewWorkConn = "NewWorkConn"
	// OpNewUserConn 新建用户连接操作
	OpNewUserConn = "NewUserConn"
)

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// IsSupport 检查是否支持指定操作
	IsSupport(op string) bool
	// Handle 处理操作
	Handle(ctx context.Context, op string, content any) (res *Response, retContent any, err error)
}
