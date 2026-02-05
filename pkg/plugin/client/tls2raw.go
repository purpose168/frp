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

//go:build !frps

package client

import (
	"context"
	"crypto/tls"
	"net"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/transport"
	netpkg "github.com/purpose168/frp/pkg/util/net"
	"github.com/purpose168/frp/pkg/util/xlog"
)

func init() {
	Register(v1.PluginTLS2Raw, NewTLS2RawPlugin)
}

// TLS2RawPlugin TLS到原始TCP连接插件
type TLS2RawPlugin struct {
	// opts 插件选项
	opts *v1.TLS2RawPluginOptions

	// tlsConfig TLS配置
	tlsConfig *tls.Config
}

// NewTLS2RawPlugin 创建TLS到原始TCP连接插件
func NewTLS2RawPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.TLS2RawPluginOptions)

	p := &TLS2RawPlugin{
		opts: opts,
	}

	tlsConfig, err := transport.NewServerTLSConfig(p.opts.CrtPath, p.opts.KeyPath, "")
	if err != nil {
		return nil, err
	}
	p.tlsConfig = tlsConfig
	return p, nil
}

// Handle 处理连接
func (p *TLS2RawPlugin) Handle(ctx context.Context, connInfo *ConnectionInfo) {
	xl := xlog.FromContextSafe(ctx)

	wrapConn := netpkg.WrapReadWriteCloserToConn(connInfo.Conn, connInfo.UnderlyingConn)
	tlsConn := tls.Server(wrapConn, p.tlsConfig)

	if err := tlsConn.Handshake(); err != nil {
		xl.Warnf("TLS握手错误: %v", err)
		return
	}
	rawConn, err := net.Dial("tcp", p.opts.LocalAddr)
	if err != nil {
		xl.Warnf("连接本地地址错误: %v", err)
		return
	}

	libio.Join(tlsConn, rawConn)
}

// Name 返回插件名称
func (p *TLS2RawPlugin) Name() string {
	return v1.PluginTLS2Raw
}

// Close 关闭插件
func (p *TLS2RawPlugin) Close() error {
	return nil
}
