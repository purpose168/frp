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

package v1

import (
	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/config/types"
	"github.com/fatedier/frp/pkg/util/util"
)

// ServerConfig 服务器配置结构体
type ServerConfig struct {
	APIMetadata

	Auth AuthServerConfig `json:"auth,omitempty"`
	// BindAddr 指定服务器绑定的地址。默认值为 "0.0.0.0"
	BindAddr string `json:"bindAddr,omitempty"`
	// BindPort 指定服务器监听的端口。默认值为 7000
	BindPort int `json:"bindPort,omitempty"`
	// KCPBindPort 指定服务器监听的 KCP 端口。如果此值为 0，服务器将不监听 KCP 连接
	KCPBindPort int `json:"kcpBindPort,omitempty"`
	// QUICBindPort 指定服务器监听的 QUIC 端口。将此值设置为 0 将禁用此功能
	QUICBindPort int `json:"quicBindPort,omitempty"`
	// ProxyBindAddr 指定代理绑定的地址。此值可能与 BindAddr 相同
	ProxyBindAddr string `json:"proxyBindAddr,omitempty"`
	// VhostHTTPPort 指定服务器监听 HTTP 虚拟主机请求的端口。如果此值为 0，服务器将不监听 HTTP 请求
	VhostHTTPPort int `json:"vhostHTTPPort,omitempty"`
	// VhostHTTPTimeout 指定虚拟主机 HTTP 服务器的响应头超时时间（以秒为单位）。默认值为 60
	VhostHTTPTimeout int64 `json:"vhostHTTPTimeout,omitempty"`
	// VhostHTTPSPort 指定服务器监听 HTTPS 虚拟主机请求的端口。如果此值为 0，服务器将不监听 HTTPS 请求
	VhostHTTPSPort int `json:"vhostHTTPSPort,omitempty"`
	// TCPMuxHTTPConnectPort 指定服务器监听 TCP HTTP CONNECT 请求的端口。如果值为 0，服务器将不在单个端口上复用 TCP 请求。如果不为 0，服务器将在此端口上监听 HTTP CONNECT 请求
	TCPMuxHTTPConnectPort int `json:"tcpmuxHTTPConnectPort,omitempty"`
	// TCPMuxPassthrough 如果为 true，frps 将不会对流量进行任何更新
	TCPMuxPassthrough bool `json:"tcpmuxPassthrough,omitempty"`
	// SubDomainHost 指定使用虚拟主机代理时将附加到客户端请求的子域名的域名。例如，如果此值设置为 "frps.com" 且客户端请求子域名 "test"，则生成的 URL 将为 "test.frps.com"
	SubDomainHost string `json:"subDomainHost,omitempty"`
	// Custom404Page 指定自定义 404 页面的路径以供显示。如果此值为 ""，将显示默认页面
	Custom404Page string `json:"custom404Page,omitempty"`

	SSHTunnelGateway SSHTunnelGateway `json:"sshTunnelGateway,omitempty"`

	WebServer WebServerConfig `json:"webServer,omitempty"`
	// EnablePrometheus 将在 webserver 地址的 /metrics API 上导出 Prometheus 指标
	EnablePrometheus bool `json:"enablePrometheus,omitempty"`

	Log LogConfig `json:"log,omitempty"`

	Transport ServerTransportConfig `json:"transport,omitempty"`

	// DetailedErrorsToClient 定义是否将特定错误（包含调试信息）发送到 frpc。默认值为 true
	DetailedErrorsToClient *bool `json:"detailedErrorsToClient,omitempty"`
	// MaxPortsPerClient 指定单个客户端可以代理的最大端口数。如果此值为 0，将不应用任何限制
	MaxPortsPerClient int64 `json:"maxPortsPerClient,omitempty"`
	// UserConnTimeout 指定等待工作连接的最大时间。默认值为 10
	UserConnTimeout int64 `json:"userConnTimeout,omitempty"`
	// UDPPacketSize 指定 UDP 数据包大小。默认值为 1500
	UDPPacketSize int64 `json:"udpPacketSize,omitempty"`
	// NatHoleAnalysisDataReserveHours 指定保留 NAT 穿透分析数据的小时数
	NatHoleAnalysisDataReserveHours int64 `json:"natholeAnalysisDataReserveHours,omitempty"`

	AllowPorts []types.PortsRange `json:"allowPorts,omitempty"`

	HTTPPlugins []HTTPPluginOptions `json:"httpPlugins,omitempty"`
}

// Complete 完成服务器配置，设置默认值
func (c *ServerConfig) Complete() error {
	if err := c.Auth.Complete(); err != nil {
		return err
	}
	c.Log.Complete()
	c.Transport.Complete()
	c.WebServer.Complete()
	c.SSHTunnelGateway.Complete()

	c.BindAddr = util.EmptyOr(c.BindAddr, "0.0.0.0")
	c.BindPort = util.EmptyOr(c.BindPort, 7000)
	if c.ProxyBindAddr == "" {
		c.ProxyBindAddr = c.BindAddr
	}

	if c.WebServer.Port > 0 {
		c.WebServer.Addr = util.EmptyOr(c.WebServer.Addr, "0.0.0.0")
	}

	c.VhostHTTPTimeout = util.EmptyOr(c.VhostHTTPTimeout, 60)
	c.DetailedErrorsToClient = util.EmptyOr(c.DetailedErrorsToClient, lo.ToPtr(true))
	c.UserConnTimeout = util.EmptyOr(c.UserConnTimeout, 10)
	c.UDPPacketSize = util.EmptyOr(c.UDPPacketSize, 1500)
	c.NatHoleAnalysisDataReserveHours = util.EmptyOr(c.NatHoleAnalysisDataReserveHours, 7*24)
	return nil
}

// AuthServerConfig 认证服务器配置结构体
type AuthServerConfig struct {
	Method           AuthMethod           `json:"method,omitempty"`
	AdditionalScopes []AuthScope          `json:"additionalScopes,omitempty"`
	Token            string               `json:"token,omitempty"`
	TokenSource      *ValueSource         `json:"tokenSource,omitempty"`
	OIDC             AuthOIDCServerConfig `json:"oidc,omitempty"`
}

// Complete 完成认证服务器配置，设置默认值
func (c *AuthServerConfig) Complete() error {
	c.Method = util.EmptyOr(c.Method, "token")
	return nil
}

// AuthOIDCServerConfig OIDC 认证服务器配置结构体
type AuthOIDCServerConfig struct {
	// Issuer 指定用于验证 OIDC 令牌的颁发者。此颁发者将用于加载公钥以验证签名，并将与 OIDC 令牌中的颁发者声明进行比较
	Issuer string `json:"issuer,omitempty"`
	// Audience 指定验证时 OIDC 令牌应包含的受众（Audience）。如果此值为空，将跳过受众（"客户端 ID"）验证
	Audience string `json:"audience,omitempty"`
	// SkipExpiryCheck 指定是否跳过检查 OIDC 令牌是否已过期
	SkipExpiryCheck bool `json:"skipExpiryCheck,omitempty"`
	// SkipIssuerCheck 指定是否跳过检查 OIDC 令牌的颁发者声明是否与 OidcIssuer 中指定的颁发者匹配
	SkipIssuerCheck bool `json:"skipIssuerCheck,omitempty"`
}

// ServerTransportConfig 服务器传输配置结构体
type ServerTransportConfig struct {
	// TCPMux 切换 TCP 流多路复用。这允许来自客户端的多个请求共享单个 TCP 连接。默认值为 true
	// $HideFromDoc
	TCPMux *bool `json:"tcpMux,omitempty"`
	// TCPMuxKeepaliveInterval 指定 TCP 流多路复用的保活间隔。如果 TCPMux 为 true，则不需要应用层心跳，因为它只能依赖 TCPMux 中的心跳
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	// TCPKeepAlive 指定 frpc 和 frps 之间活动网络连接的保活探测间隔。如果为负值，则禁用保活探测
	TCPKeepAlive int64 `json:"tcpKeepalive,omitempty"`
	// MaxPoolCount 指定每个代理的最大池大小。默认值为 5
	MaxPoolCount int64 `json:"maxPoolCount,omitempty"`
	// HeartBeatTimeout 指定在终止连接之前等待心跳的最长时间。不建议更改此值。默认值为 90。设置为负值可禁用它
	HeartbeatTimeout int64 `json:"heartbeatTimeout,omitempty"`
	// QUIC 选项
	QUIC QUICOptions `json:"quic,omitempty"`
	// TLS 指定来自客户端的连接的 TLS 设置
	TLS TLSServerConfig `json:"tls,omitempty"`
}

// Complete 完成服务器传输配置，设置默认值
func (c *ServerTransportConfig) Complete() {
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	c.TCPKeepAlive = util.EmptyOr(c.TCPKeepAlive, 7200)
	c.MaxPoolCount = util.EmptyOr(c.MaxPoolCount, 5)
	if lo.FromPtr(c.TCPMux) {
		// 如果启用了 TCPMux，则不需要应用层心跳，因为我们可以依赖 tcpmux 中的心跳
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	c.QUIC.Complete()
	if c.TLS.TrustedCaFile != "" {
		c.TLS.Force = true
	}
}

// TLSServerConfig TLS 服务器配置结构体
type TLSServerConfig struct {
	// Force 指定是否仅接受 TLS 加密的连接
	Force bool `json:"force,omitempty"`

	TLSConfig
}

// SSHTunnelGateway SSH 隧道网关配置结构体
type SSHTunnelGateway struct {
	BindPort              int    `json:"bindPort,omitempty"`
	PrivateKeyFile        string `json:"privateKeyFile,omitempty"`
	AutoGenPrivateKeyPath string `json:"autoGenPrivateKeyPath,omitempty"`
	AuthorizedKeysFile    string `json:"authorizedKeysFile,omitempty"`
}

// Complete 完成 SSH 隧道网关配置，设置默认值
func (c *SSHTunnelGateway) Complete() {
	c.AutoGenPrivateKeyPath = util.EmptyOr(c.AutoGenPrivateKeyPath, "./.autogen_ssh_key")
}
