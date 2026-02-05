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
	"os"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

// ClientConfig 客户端配置结构体
type ClientConfig struct {
	ClientCommonConfig

	// Proxies 代理配置列表
	Proxies []TypedProxyConfig `json:"proxies,omitempty"`
	// Visitors 访问者配置列表
	Visitors []TypedVisitorConfig `json:"visitors,omitempty"`
}

// ClientCommonConfig 客户端通用配置结构体
type ClientCommonConfig struct {
	APIMetadata

	// Auth 认证配置
	Auth AuthClientConfig `json:"auth,omitempty"`
	// User 指定代理名称的前缀，用于区分不同的客户端
	// 如果此值不为空，代理名称将自动更改为 "{user}.{proxy_name}"
	User string `json:"user,omitempty"`
	// ClientID 唯一标识此 frpc 实例
	ClientID string `json:"clientID,omitempty"`

	// ServerAddr 指定要连接的服务器地址
	// 默认值为 "0.0.0.0"
	ServerAddr string `json:"serverAddr,omitempty"`
	// ServerPort 指定连接服务器的端口
	// 默认值为 7000
	ServerPort int `json:"serverPort,omitempty"`
	// NatHoleSTUNServer 用于帮助穿透 NAT 的 STUN 服务器
	NatHoleSTUNServer string `json:"natHoleStunServer,omitempty"`
	// DNSServer 指定 FRPC 使用的 DNS 服务器地址
	// 如果此值为空，将使用默认 DNS
	DNSServer string `json:"dnsServer,omitempty"`
	// LoginFailExit 控制客户端在登录失败后是否退出
	// 如果为 false，客户端将重试直到登录成功
	// 默认值为 true
	LoginFailExit *bool `json:"loginFailExit,omitempty"`
	// Start 指定按名称启用的代理集合
	// 如果此集合为空，所有提供的代理都将启用
	// 默认值为空集合
	Start []string `json:"start,omitempty"`

	// Log 日志配置
	Log LogConfig `json:"log,omitempty"`
	// WebServer Web 服务器配置
	WebServer WebServerConfig `json:"webServer,omitempty"`
	// Transport 传输层配置
	Transport ClientTransportConfig `json:"transport,omitempty"`
	// VirtualNet 虚拟网络配置
	VirtualNet VirtualNetConfig `json:"virtualNet,omitempty"`

	// FeatureGates 指定要启用或禁用的特性门控集合
	// 可用于启用 alpha/beta 特性或禁用默认特性
	FeatureGates map[string]bool `json:"featureGates,omitempty"`

	// UDPPacketSize 指定 UDP 数据包大小
	// 默认值为 1500
	UDPPacketSize int64 `json:"udpPacketSize,omitempty"`
	// Metadatas 客户端元数据信息
	Metadatas map[string]string `json:"metadatas,omitempty"`

	// IncludeConfigFiles 包含其他代理配置文件
	IncludeConfigFiles []string `json:"includes,omitempty"`
}

// Complete 填充客户端通用配置的默认值
func (c *ClientCommonConfig) Complete() error {
	// 设置默认的服务器地址
	c.ServerAddr = util.EmptyOr(c.ServerAddr, "0.0.0.0")
	// 设置默认的服务器端口
	c.ServerPort = util.EmptyOr(c.ServerPort, 7000)
	// 设置默认的登录失败退出行为
	c.LoginFailExit = util.EmptyOr(c.LoginFailExit, lo.ToPtr(true))
	// 设置默认的 NAT 穿透 STUN 服务器
	c.NatHoleSTUNServer = util.EmptyOr(c.NatHoleSTUNServer, "stun.easyvoip.com:3478")

	// 完成认证配置
	if err := c.Auth.Complete(); err != nil {
		return err
	}
	// 完成日志配置
	c.Log.Complete()
	// 完成传输层配置
	c.Transport.Complete()
	// 完成 Web 服务器配置
	c.WebServer.Complete()

	// 设置默认的 UDP 数据包大小
	c.UDPPacketSize = util.EmptyOr(c.UDPPacketSize, 1500)
	return nil
}

// ClientTransportConfig 客户端传输层配置结构体
type ClientTransportConfig struct {
	// Protocol 指定与服务器交互时使用的协议
	// 有效值为 "tcp"、"kcp"、"quic"、"websocket" 和 "wss"
	// 默认值为 "tcp"
	Protocol string `json:"protocol,omitempty"`
	// DialServerTimeout 指定连接服务器时等待连接完成的最大时间（秒）
	DialServerTimeout int64 `json:"dialServerTimeout,omitempty"`
	// DialServerKeepAlive 指定 frpc 和 frps 之间活动网络连接的保活探测间隔（秒）
	// 如果为负值，则禁用保活探测
	DialServerKeepAlive int64 `json:"dialServerKeepalive,omitempty"`
	// ConnectServerLocalIP 指定客户端连接到服务器时绑定的地址
	// 注意：此值仅用于 TCP/Websocket 协议，不支持 KCP 协议
	ConnectServerLocalIP string `json:"connectServerLocalIP,omitempty"`
	// ProxyURL 指定用于连接服务器的代理地址
	// 如果此值为空，将直接连接到服务器
	// 默认值从 "http_proxy" 环境变量读取
	ProxyURL string `json:"proxyURL,omitempty"`
	// PoolCount 指定客户端预先与服务器建立的连接数
	PoolCount int `json:"poolCount,omitempty"`
	// TCPMux 切换 TCP 流多路复用
	// 允许来自客户端的多个请求共享单个 TCP 连接
	// 如果为 true，服务器也必须启用 TCP 多路复用
	// 默认值为 true
	TCPMux *bool `json:"tcpMux,omitempty"`
	// TCPMuxKeepaliveInterval 指定 TCP 流多路复用的保活间隔（秒）
	// 如果 TCPMux 为 true，则不需要应用层心跳，因为它可以仅依赖 TCPMux 中的心跳
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	// QUIC QUIC 协议选项
	QUIC QUICOptions `json:"quic,omitempty"`
	// HeartBeatInterval 指定向服务器发送心跳的间隔（秒）
	// 不建议更改此值
	// 默认值为 30，设置为负值可禁用
	HeartbeatInterval int64 `json:"heartbeatInterval,omitempty"`
	// HeartBeatTimeout 指定连接终止前允许的最大心跳响应延迟（秒）
	// 不建议更改此值
	// 默认值为 90，设置为负值可禁用
	HeartbeatTimeout int64 `json:"heartbeatTimeout,omitempty"`
	// TLS 指定与服务器的连接的 TLS 设置
	TLS TLSClientConfig `json:"tls,omitempty"`
}

// Complete 填充客户端传输层配置的默认值
func (c *ClientTransportConfig) Complete() {
	// 设置默认协议
	c.Protocol = util.EmptyOr(c.Protocol, "tcp")
	// 设置默认的连接超时
	c.DialServerTimeout = util.EmptyOr(c.DialServerTimeout, 10)
	// 设置默认的保活间隔
	c.DialServerKeepAlive = util.EmptyOr(c.DialServerKeepAlive, 7200)
	// 设置默认的代理 URL
	c.ProxyURL = util.EmptyOr(c.ProxyURL, os.Getenv("http_proxy"))
	// 设置默认的连接池数量
	c.PoolCount = util.EmptyOr(c.PoolCount, 1)
	// 设置默认的 TCP 多路复用
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	// 设置默认的 TCP 多路复用保活间隔
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	// 根据 TCP 多路复用设置配置心跳
	if lo.FromPtr(c.TCPMux) {
		// 如果启用 TCP 多路复用，不需要应用层心跳，因为我们可以依赖 tcpmux 中的心跳
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, -1)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, 30)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	// 完成 QUIC 配置
	c.QUIC.Complete()
	// 完成 TLS 配置
	c.TLS.Complete()
}

// TLSClientConfig 客户端 TLS 配置结构体
type TLSClientConfig struct {
	// TLSEnable 指定与服务器通信时是否使用 TLS
	// 如果 "tls.certFile" 和 "tls.keyFile" 有效，客户端将加载提供的 TLS 配置
	// 自 v0.50.0 起，默认值已更改为 true，默认启用 TLS
	Enable *bool `json:"enable,omitempty"`
	// DisableCustomTLSFirstByte 如果设置为 false，当启用 TLS 时，frpc 将使用第一个自定义字节与 frps 建立连接
	// 自 v0.50.0 起，默认值已更改为 true，默认禁用第一个自定义字节
	DisableCustomTLSFirstByte *bool `json:"disableCustomTLSFirstByte,omitempty"`

	TLSConfig
}

// Complete 填充客户端 TLS 配置的默认值
func (c *TLSClientConfig) Complete() {
	// 设置默认启用 TLS
	c.Enable = util.EmptyOr(c.Enable, lo.ToPtr(true))
	// 设置默认禁用第一个自定义字节
	c.DisableCustomTLSFirstByte = util.EmptyOr(c.DisableCustomTLSFirstByte, lo.ToPtr(true))
}

// AuthClientConfig 客户端认证配置结构体
type AuthClientConfig struct {
	// Method 指定用于 frpc 与 frps 认证的认证方法
	// 如果指定 "token" - 将读取 token 到登录消息中
	// 如果指定 "oidc" - 将使用 OIDC 设置颁发 OIDC（Open ID Connect）令牌
	// 默认值为 "token"
	Method AuthMethod `json:"method,omitempty"`
	// AdditionalScopes 指定是否在额外作用域中包含认证信息
	// 当前支持的作用域有："HeartBeats"、"NewWorkConns"
	AdditionalScopes []AuthScope `json:"additionalScopes,omitempty"`
	// Token 指定用于创建发送到服务器的密钥的授权令牌
	// 服务器必须具有匹配的令牌才能授权成功
	// 默认值为 ""
	Token string `json:"token,omitempty"`
	// TokenSource 指定授权令牌的动态源
	// 与 Token 字段互斥
	TokenSource *ValueSource `json:"tokenSource,omitempty"`
	// OIDC OIDC 认证配置
	OIDC AuthOIDCClientConfig `json:"oidc,omitempty"`
}

// Complete 填充客户端认证配置的默认值
func (c *AuthClientConfig) Complete() error {
	// 设置默认的认证方法
	c.Method = util.EmptyOr(c.Method, "token")
	return nil
}

// AuthOIDCClientConfig 客户端 OIDC 认证配置结构体
type AuthOIDCClientConfig struct {
	// ClientID 指定在 OIDC 认证中用于获取令牌的客户端 ID
	ClientID string `json:"clientID,omitempty"`
	// ClientSecret 指定在 OIDC 认证中用于获取令牌的客户端密钥
	ClientSecret string `json:"clientSecret,omitempty"`
	// Audience 指定 OIDC 认证中令牌的受众
	Audience string `json:"audience,omitempty"`
	// Scope 指定 OIDC 认证中令牌的作用域
	Scope string `json:"scope,omitempty"`
	// TokenEndpointURL 指定实现 OIDC 令牌端点的 URL
	// 将用于获取 OIDC 令牌
	TokenEndpointURL string `json:"tokenEndpointURL,omitempty"`
	// AdditionalEndpointParams 指定要发送的额外参数
	// 此字段将在 OIDC 令牌生成器中转换为 map[string][]string
	AdditionalEndpointParams map[string]string `json:"additionalEndpointParams,omitempty"`

	// TrustedCaFile 指定用于验证 OIDC 令牌端点 TLS 证书的自定义 CA 证书文件路径
	TrustedCaFile string `json:"trustedCaFile,omitempty"`
	// InsecureSkipVerify 禁用 OIDC 令牌端点的 TLS 证书验证
	// 仅用于调试，不建议在生产环境中使用
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
	// ProxyURL 指定连接到 OIDC 令牌端点时使用的代理
	// 支持 http、https、socks5 和 socks5h 代理协议
	// 如果为空，则不对 OIDC 连接使用代理
	ProxyURL string `json:"proxyURL,omitempty"`

	// TokenSource 指定授权令牌的自定义动态源
	// 与此结构的所有其他字段互斥
	TokenSource *ValueSource `json:"tokenSource,omitempty"`
}

// VirtualNetConfig 虚拟网络配置结构体
type VirtualNetConfig struct {
	// Address 虚拟网络地址
	Address string `json:"address,omitempty"`
}
