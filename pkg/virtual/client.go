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

package virtual

import (
	"context"
	"net"

	"github.com/purpose168/frp/client"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	netpkg "github.com/purpose168/frp/pkg/util/net"
)

// ClientOptions 虚拟客户端选项
type ClientOptions struct {
	Common           *v1.ClientCommonConfig                                       // 客户端通用配置
	Spec             *msg.ClientSpec                                              // 客户端规格
	HandleWorkConnCb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool // 处理工作连接的回调函数
}

// Client 虚拟客户端，用于创建和管理与服务端的连接
type Client struct {
	l   *netpkg.InternalListener // 内部监听器，用于接收连接
	svr *client.Service          // 客户端服务实例
}

// NewClient 创建一个新的虚拟客户端
// options: 客户端选项
// 返回创建的客户端实例和可能的错误
func NewClient(options ClientOptions) (*Client, error) {
	// 完成客户端通用配置
	if options.Common != nil {
		if err := options.Common.Complete(); err != nil {
			return nil, err
		}
	}

	// 创建内部监听器
	ln := netpkg.NewInternalListener()

	// 配置客户端服务选项
	serviceOptions := client.ServiceOptions{
		Common:     options.Common,
		ClientSpec: options.Spec,
		// 创建连接器，使用内部监听器作为对等方监听器
		ConnectorCreator: func(context.Context, *v1.ClientCommonConfig) client.Connector {
			return &pipeConnector{
				peerListener: ln,
			}
		},
		HandleWorkConnCb: options.HandleWorkConnCb,
	}
	// 创建客户端服务
	svr, err := client.NewService(serviceOptions)
	if err != nil {
		return nil, err
	}
	return &Client{
		l:   ln,
		svr: svr,
	}, nil
}

// PeerListener 获取对等方监听器
// 返回 net.Listener 接口
func (c *Client) PeerListener() net.Listener {
	return c.l
}

// UpdateProxyConfigurer 更新代理配置器
// proxyCfgs: 代理配置器列表
func (c *Client) UpdateProxyConfigurer(proxyCfgs []v1.ProxyConfigurer) {
	_ = c.svr.UpdateAllConfigurer(proxyCfgs, nil)
}

// Run 运行客户端服务
// ctx: 上下文，用于控制服务的生命周期
// 返回可能的错误
func (c *Client) Run(ctx context.Context) error {
	return c.svr.Run(ctx)
}

// Service 获取客户端服务实例
// 返回 *client.Service
func (c *Client) Service() *client.Service {
	return c.svr
}

// Close 关闭客户端，包括服务和监听器
func (c *Client) Close() {
	c.svr.Close()
	c.l.Close()
}

// pipeConnector 管道连接器，用于在内部创建连接
type pipeConnector struct {
	peerListener *netpkg.InternalListener // 对等方监听器
}

// Open 打开连接器
// 返回可能的错误
func (pc *pipeConnector) Open() error {
	return nil
}

// Connect 创建一个新的连接
// 返回创建的连接和可能的错误
func (pc *pipeConnector) Connect() (net.Conn, error) {
	// 创建一对互相连接的管道
	c1, c2 := net.Pipe()
	// 将其中一个连接放入对等方监听器
	if err := pc.peerListener.PutConn(c1); err != nil {
		// 如果失败，关闭两个连接
		c1.Close()
		c2.Close()
		return nil, err
	}
	// 返回另一个连接
	return c2, nil
}

// Close 关闭连接器
// 返回可能的错误
func (pc *pipeConnector) Close() error {
	// 关闭对等方监听器
	pc.peerListener.Close()
	return nil
}
