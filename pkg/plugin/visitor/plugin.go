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

package visitor

import (
	"context"
	"fmt"
	"net"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/vnet"
)

// PluginContext 插件上下文，为访问者插件提供必要的上下文和回调
type PluginContext struct {
	// Name 是此访问者的唯一标识符，用于日志记录和路由
	Name string

	// Ctx 管理插件的生命周期并携带用于结构化日志记录的日志记录器
	Ctx context.Context

	// VnetController 管理TUN设备路由。如果禁用了虚拟网络，可能为nil
	VnetController *vnet.Controller

	// SendConnToVisitor 将连接发送到访问者的内部处理队列
	// 不返回错误；失败通过关闭连接来处理
	SendConnToVisitor func(net.Conn)
}

// Creators 用于创建插件以处理连接
var creators = make(map[string]CreatorFn)

// CreatorFn 创建插件的函数类型
type CreatorFn func(pluginCtx PluginContext, options v1.VisitorPluginOptions) (Plugin, error)

// Register 注册插件创建函数
func Register(name string, fn CreatorFn) {
	if _, exist := creators[name]; exist {
		panic(fmt.Sprintf("plugin [%s] is already registered", name))
	}
	creators[name] = fn
}

// Create 创建插件实例
func Create(pluginName string, pluginCtx PluginContext, options v1.VisitorPluginOptions) (p Plugin, err error) {
	if fn, ok := creators[pluginName]; ok {
		p, err = fn(pluginCtx, options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", pluginName)
	}
	return
}

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// Start 启动插件
	Start()
	// Close 关闭插件
	Close() error
}
