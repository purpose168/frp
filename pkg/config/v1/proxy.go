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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"

	"github.com/purpose168/frp/pkg/config/types"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/util/util"
)

// ProxyTransport 代理传输配置结构体
type ProxyTransport struct {
	// UseEncryption 控制与服务器的通信是否加密
	// 加密使用服务器和客户端配置中提供的令牌
	UseEncryption bool `json:"useEncryption,omitempty"`
	// UseCompression 控制与服务器的通信是否压缩
	UseCompression bool `json:"useCompression,omitempty"`
	// BandwidthLimit 带宽限制
	// 0 表示无限制
	BandwidthLimit types.BandwidthQuantity `json:"bandwidthLimit,omitempty"`
	// BandwidthLimitMode 指定在客户端还是服务端限制带宽
	// 有效值包括 "client" 和 "server"
	// 默认值为 "client"
	BandwidthLimitMode string `json:"bandwidthLimitMode,omitempty"`
	// ProxyProtocolVersion 指定使用哪个协议版本
	// 有效值包括 "v1"、"v2" 和 ""
	// 如果值为 ""，将自动选择协议版本
	// 默认值为 ""
	ProxyProtocolVersion string `json:"proxyProtocolVersion,omitempty"`
}

// LoadBalancerConfig 负载均衡器配置结构体
type LoadBalancerConfig struct {
	// Group 指定所属的组
	// 服务器将使用此信息对同一组中的代理进行负载均衡
	// 如果值为 ""，则不在任何组中
	Group string `json:"group"`
	// GroupKey 指定组密钥，同一组的代理应该具有相同的组密钥
	GroupKey string `json:"groupKey,omitempty"`
}

// ProxyBackend 代理后端配置结构体
type ProxyBackend struct {
	// LocalIP 指定后端的 IP 地址或主机名
	LocalIP string `json:"localIP,omitempty"`
	// LocalPort 指定后端的端口
	LocalPort int `json:"localPort,omitempty"`

	// Plugin 指定用于处理连接的插件
	// 如果设置了此值，将忽略 LocalIP 和 LocalPort 值
	Plugin TypedClientPluginOptions `json:"plugin,omitempty"`
}

// HealthCheckConfig 健康检查配置结构体
// 可用于负载均衡目的，以检测并移除到失败服务的代理
type HealthCheckConfig struct {
	// Type 指定用于健康检查的协议
	// 有效值包括 "tcp"、"http" 和 ""
	// 如果此值为 ""，则不执行健康检查
	//
	// 如果类型为 "tcp"，将尝试连接到目标服务器
	// 如果无法建立连接，则健康检查失败
	//
	// 如果类型为 "http"，将向 HealthCheckURL 指定的端点发送 GET 请求
	// 如果响应不是 200，则健康检查失败
	Type string `json:"type"` // tcp | http
	// TimeoutSeconds 指定等待健康检查尝试连接的秒数
	// 如果达到超时，则计为健康检查失败
	// 默认值为 3
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
	// MaxFailed 指定停止之前允许的失败次数
	// 默认值为 1
	MaxFailed int `json:"maxFailed,omitempty"`
	// IntervalSeconds 指定健康检查之间的时间间隔（秒）
	// 默认值为 10
	IntervalSeconds int `json:"intervalSeconds"`
	// Path 指定如果健康检查类型为 "http" 时发送健康检查的路径
	Path string `json:"path,omitempty"`
	// HTTPHeaders 指定如果健康检查类型为 "http" 时与健康请求一起发送的头部
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty"`
}

// DomainConfig 域名配置结构体
type DomainConfig struct {
	// CustomDomains 自定义域名列表
	CustomDomains []string `json:"customDomains,omitempty"`
	// SubDomain 子域名
	SubDomain string `json:"subdomain,omitempty"`
}

// ProxyBaseConfig 代理基础配置结构体
type ProxyBaseConfig struct {
	// Name 代理名称
	Name string `json:"name"`
	// Type 代理类型
	Type string `json:"type"`
	// Enabled 控制此代理是否启用
	// nil 或 true 表示启用，false 表示禁用
	// 这允许对每个代理进行单独控制，补充全局 "start" 字段
	Enabled *bool `json:"enabled,omitempty"`
	// Annotations 注解映射
	Annotations map[string]string `json:"annotations,omitempty"`
	// Transport 传输配置
	Transport ProxyTransport `json:"transport,omitempty"`
	// Metadatas 每个代理的元数据信息
	Metadatas map[string]string `json:"metadatas,omitempty"`
	// LoadBalancer 负载均衡器配置
	LoadBalancer LoadBalancerConfig `json:"loadBalancer,omitempty"`
	// HealthCheck 健康检查配置
	HealthCheck HealthCheckConfig `json:"healthCheck,omitempty"`
	ProxyBackend
}

// GetBaseConfig 获取基础配置
func (c *ProxyBaseConfig) GetBaseConfig() *ProxyBaseConfig {
	return c
}

// Complete 填充代理基础配置的默认值
func (c *ProxyBaseConfig) Complete(namePrefix string) {
	// 设置代理名称前缀
	c.Name = lo.Ternary(namePrefix == "", "", namePrefix+".") + c.Name
	// 设置默认的本地 IP
	c.LocalIP = util.EmptyOr(c.LocalIP, "127.0.0.1")
	// 设置默认的带宽限制模式
	c.Transport.BandwidthLimitMode = util.EmptyOr(c.Transport.BandwidthLimitMode, types.BandwidthLimitModeClient)

	// 如果配置了插件，完成插件配置
	if c.Plugin.ClientPluginOptions != nil {
		c.Plugin.Complete()
	}
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
// 此函数将在 frpc 端调用
func (c *ProxyBaseConfig) MarshalToMsg(m *msg.NewProxy) {
	m.ProxyName = c.Name
	m.ProxyType = c.Type
	m.UseEncryption = c.Transport.UseEncryption
	m.UseCompression = c.Transport.UseCompression
	m.BandwidthLimit = c.Transport.BandwidthLimit.String()
	// 留空以使用默认值以减少流量
	if c.Transport.BandwidthLimitMode != "client" {
		m.BandwidthLimitMode = c.Transport.BandwidthLimitMode
	}
	m.Group = c.LoadBalancer.Group
	m.GroupKey = c.LoadBalancer.GroupKey
	m.Metas = c.Metadatas
	m.Annotations = c.Annotations
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
// 此函数将在 frps 端调用
func (c *ProxyBaseConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.Name = m.ProxyName
	c.Type = m.ProxyType
	c.Transport.UseEncryption = m.UseEncryption
	c.Transport.UseCompression = m.UseCompression
	if m.BandwidthLimit != "" {
		c.Transport.BandwidthLimit, _ = types.NewBandwidthQuantity(m.BandwidthLimit)
	}
	if m.BandwidthLimitMode != "" {
		c.Transport.BandwidthLimitMode = m.BandwidthLimitMode
	}
	c.LoadBalancer.Group = m.Group
	c.LoadBalancer.GroupKey = m.GroupKey
	c.Metadatas = m.Metas
	c.Annotations = m.Annotations
}

// TypedProxyConfig 类型化代理配置结构体
type TypedProxyConfig struct {
	// Type 代理类型
	Type string `json:"type"`
	ProxyConfigurer
}

// UnmarshalJSON 自定义 JSON 反序列化方法
func (c *TypedProxyConfig) UnmarshalJSON(b []byte) error {
	// 处理 null 值
	if len(b) == 4 && string(b) == "null" {
		return errors.New("类型是必需的")
	}

	// 解析代理类型
	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	// 根据类型创建对应的配置器
	configurer := NewProxyConfigurerByType(ProxyType(typeStruct.Type))
	if configurer == nil {
		return fmt.Errorf("未知的代理类型: %s", typeStruct.Type)
	}
	// 创建 JSON 解码器
	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	// 解码配置器
	if err := decoder.Decode(configurer); err != nil {
		return fmt.Errorf("反序列化 ProxyConfig 错误: %v", err)
	}
	c.ProxyConfigurer = configurer
	return nil
}

// MarshalJSON 自定义 JSON 序列化方法
func (c *TypedProxyConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ProxyConfigurer)
}

// ProxyConfigurer 代理配置器接口
type ProxyConfigurer interface {
	// Complete 填充配置的默认值
	Complete(namePrefix string)
	// GetBaseConfig 获取基础配置
	GetBaseConfig() *ProxyBaseConfig
	// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
	// 此函数将在 frpc 端调用
	MarshalToMsg(*msg.NewProxy)
	// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
	// 此函数将在 frps 端调用
	UnmarshalFromMsg(*msg.NewProxy)
}

// ProxyType 代理类型
type ProxyType string

const (
	// ProxyTypeTCP TCP 代理类型
	ProxyTypeTCP ProxyType = "tcp"
	// ProxyTypeUDP UDP 代理类型
	ProxyTypeUDP ProxyType = "udp"
	// ProxyTypeTCPMUX TCP 多路复用代理类型
	ProxyTypeTCPMUX ProxyType = "tcpmux"
	// ProxyTypeHTTP HTTP 代理类型
	ProxyTypeHTTP ProxyType = "http"
	// ProxyTypeHTTPS HTTPS 代理类型
	ProxyTypeHTTPS ProxyType = "https"
	// ProxyTypeSTCP STCP 代理类型
	ProxyTypeSTCP ProxyType = "stcp"
	// ProxyTypeXTCP XTCP 代理类型
	ProxyTypeXTCP ProxyType = "xtcp"
	// ProxyTypeSUDP SUDP 代理类型
	ProxyTypeSUDP ProxyType = "sudp"
)

// proxyConfigTypeMap 代理配置类型映射
var proxyConfigTypeMap = map[ProxyType]reflect.Type{
	ProxyTypeTCP:    reflect.TypeOf(TCPProxyConfig{}),
	ProxyTypeUDP:    reflect.TypeOf(UDPProxyConfig{}),
	ProxyTypeHTTP:   reflect.TypeOf(HTTPProxyConfig{}),
	ProxyTypeHTTPS:  reflect.TypeOf(HTTPSProxyConfig{}),
	ProxyTypeTCPMUX: reflect.TypeOf(TCPMuxProxyConfig{}),
	ProxyTypeSTCP:   reflect.TypeOf(STCPProxyConfig{}),
	ProxyTypeXTCP:   reflect.TypeOf(XTCPProxyConfig{}),
	ProxyTypeSUDP:   reflect.TypeOf(SUDPProxyConfig{}),
}

// NewProxyConfigurerByType 根据代理类型创建代理配置器
func NewProxyConfigurerByType(proxyType ProxyType) ProxyConfigurer {
	v, ok := proxyConfigTypeMap[proxyType]
	if !ok {
		return nil
	}
	pc := reflect.New(v).Interface().(ProxyConfigurer)
	pc.GetBaseConfig().Type = string(proxyType)
	return pc
}

var _ ProxyConfigurer = &TCPProxyConfig{}

// TCPProxyConfig TCP 代理配置结构体
type TCPProxyConfig struct {
	ProxyBaseConfig

	// RemotePort 远程端口
	RemotePort int `json:"remotePort,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *TCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.RemotePort = c.RemotePort
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *TCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.RemotePort = m.RemotePort
}

var _ ProxyConfigurer = &UDPProxyConfig{}

// UDPProxyConfig UDP 代理配置结构体
type UDPProxyConfig struct {
	ProxyBaseConfig

	// RemotePort 远程端口
	RemotePort int `json:"remotePort,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *UDPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.RemotePort = c.RemotePort
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *UDPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.RemotePort = m.RemotePort
}

var _ ProxyConfigurer = &HTTPProxyConfig{}

// HTTPProxyConfig HTTP 代理配置结构体
type HTTPProxyConfig struct {
	ProxyBaseConfig
	DomainConfig

	// Locations 位置列表
	Locations []string `json:"locations,omitempty"`
	// HTTPUser HTTP 用户名
	HTTPUser string `json:"httpUser,omitempty"`
	// HTTPPassword HTTP 密码
	HTTPPassword string `json:"httpPassword,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"hostHeaderRewrite,omitempty"`
	// RequestHeaders 请求头操作
	RequestHeaders HeaderOperations `json:"requestHeaders,omitempty"`
	// ResponseHeaders 响应头操作
	ResponseHeaders HeaderOperations `json:"responseHeaders,omitempty"`
	// RouteByHTTPUser 按 HTTP 用户路由
	RouteByHTTPUser string `json:"routeByHTTPUser,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *HTTPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
	m.Locations = c.Locations
	m.HostHeaderRewrite = c.HostHeaderRewrite
	m.HTTPUser = c.HTTPUser
	m.HTTPPwd = c.HTTPPassword
	m.Headers = c.RequestHeaders.Set
	m.ResponseHeaders = c.ResponseHeaders.Set
	m.RouteByHTTPUser = c.RouteByHTTPUser
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *HTTPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
	c.Locations = m.Locations
	c.HostHeaderRewrite = m.HostHeaderRewrite
	c.HTTPUser = m.HTTPUser
	c.HTTPPassword = m.HTTPPwd
	c.RequestHeaders.Set = m.Headers
	c.ResponseHeaders.Set = m.ResponseHeaders
	c.RouteByHTTPUser = m.RouteByHTTPUser
}

var _ ProxyConfigurer = &HTTPSProxyConfig{}

// HTTPSProxyConfig HTTPS 代理配置结构体
type HTTPSProxyConfig struct {
	ProxyBaseConfig
	DomainConfig
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *HTTPSProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *HTTPSProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
}

// TCPMultiplexerType TCP 多路复用器类型
type TCPMultiplexerType string

const (
	// TCPMultiplexerHTTPConnect HTTP CONNECT 多路复用器
	TCPMultiplexerHTTPConnect TCPMultiplexerType = "httpconnect"
)

var _ ProxyConfigurer = &TCPMuxProxyConfig{}

// TCPMuxProxyConfig TCP 多路复用代理配置结构体
type TCPMuxProxyConfig struct {
	ProxyBaseConfig
	DomainConfig

	// HTTPUser HTTP 用户名
	HTTPUser string `json:"httpUser,omitempty"`
	// HTTPPassword HTTP 密码
	HTTPPassword string `json:"httpPassword,omitempty"`
	// RouteByHTTPUser 按 HTTP 用户路由
	RouteByHTTPUser string `json:"routeByHTTPUser,omitempty"`
	// Multiplexer 多路复用器
	Multiplexer string `json:"multiplexer,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *TCPMuxProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
	m.Multiplexer = c.Multiplexer
	m.HTTPUser = c.HTTPUser
	m.HTTPPwd = c.HTTPPassword
	m.RouteByHTTPUser = c.RouteByHTTPUser
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *TCPMuxProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
	c.Multiplexer = m.Multiplexer
	c.HTTPPassword = m.HTTPPwd
	c.RouteByHTTPUser = m.RouteByHTTPUser
}

var _ ProxyConfigurer = &STCPProxyConfig{}

// STCPProxyConfig STCP 代理配置结构体
type STCPProxyConfig struct {
	ProxyBaseConfig

	// Secretkey 密钥
	Secretkey string `json:"secretKey,omitempty"`
	// AllowUsers 允许的用户列表
	AllowUsers []string `json:"allowUsers,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *STCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.Secretkey
	m.AllowUsers = c.AllowUsers
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *STCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.Secretkey = m.Sk
	c.AllowUsers = m.AllowUsers
}

var _ ProxyConfigurer = &XTCPProxyConfig{}

// XTCPProxyConfig XTCP 代理配置结构体
type XTCPProxyConfig struct {
	ProxyBaseConfig

	// Secretkey 密钥
	Secretkey string `json:"secretKey,omitempty"`
	// AllowUsers 允许的用户列表
	AllowUsers []string `json:"allowUsers,omitempty"`

	// NatTraversal NAT 穿透配置
	NatTraversal *NatTraversalConfig `json:"natTraversal,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *XTCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.Secretkey
	m.AllowUsers = c.AllowUsers
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *XTCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.Secretkey = m.Sk
	c.AllowUsers = m.AllowUsers
}

var _ ProxyConfigurer = &SUDPProxyConfig{}

// SUDPProxyConfig SUDP 代理配置结构体
type SUDPProxyConfig struct {
	ProxyBaseConfig

	// Secretkey 密钥
	Secretkey string `json:"secretKey,omitempty"`
	// AllowUsers 允许的用户列表
	AllowUsers []string `json:"allowUsers,omitempty"`
}

// MarshalToMsg 将此配置序列化为 msg.NewProxy 消息
func (c *SUDPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.Secretkey
	m.AllowUsers = c.AllowUsers
}

// UnmarshalFromMsg 将 msg.NewProxy 消息反序列化到此配置
func (c *SUDPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.Secretkey = m.Sk
	c.AllowUsers = m.AllowUsers
}
