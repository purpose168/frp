// Copyright 2016 fatedier, fatedier@gmail.com
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

package msg

import (
	"net"
	"reflect"
)

const (
	// TypeLogin 登录消息类型
	TypeLogin = 'o'
	// TypeLoginResp 登录响应消息类型
	TypeLoginResp = '1'
	// TypeNewProxy 新建代理消息类型
	TypeNewProxy = 'p'
	// TypeNewProxyResp 新建代理响应消息类型
	TypeNewProxyResp = '2'
	// TypeCloseProxy 关闭代理消息类型
	TypeCloseProxy = 'c'
	// TypeNewWorkConn 新建工作连接消息类型
	TypeNewWorkConn = 'w'
	// TypeReqWorkConn 请求工作连接消息类型
	TypeReqWorkConn = 'r'
	// TypeStartWorkConn 启动工作连接消息类型
	TypeStartWorkConn = 's'
	// TypeNewVisitorConn 新建访问者连接消息类型
	TypeNewVisitorConn = 'v'
	// TypeNewVisitorConnResp 新建访问者连接响应消息类型
	TypeNewVisitorConnResp = '3'
	// TypePing 心跳消息类型
	TypePing = 'h'
	// TypePong 心跳响应消息类型
	TypePong = '4'
	// TypeUDPPacket UDP 数据包消息类型
	TypeUDPPacket = 'u'
	// TypeNatHoleVisitor NAT 穿透访问者消息类型
	TypeNatHoleVisitor = 'i'
	// TypeNatHoleClient NAT 穿透客户端消息类型
	TypeNatHoleClient = 'n'
	// TypeNatHoleResp NAT 穿透响应消息类型
	TypeNatHoleResp = 'm'
	// TypeNatHoleSid NAT 穿透会话 ID 消息类型
	TypeNatHoleSid = '5'
	// TypeNatHoleReport NAT 穿透报告消息类型
	TypeNatHoleReport = '6'
)

var msgTypeMap = map[byte]any{
	TypeLogin:              Login{},
	TypeLoginResp:          LoginResp{},
	TypeNewProxy:           NewProxy{},
	TypeNewProxyResp:       NewProxyResp{},
	TypeCloseProxy:         CloseProxy{},
	TypeNewWorkConn:        NewWorkConn{},
	TypeReqWorkConn:        ReqWorkConn{},
	TypeStartWorkConn:      StartWorkConn{},
	TypeNewVisitorConn:     NewVisitorConn{},
	TypeNewVisitorConnResp: NewVisitorConnResp{},
	TypePing:               Ping{},
	TypePong:               Pong{},
	TypeUDPPacket:          UDPPacket{},
	TypeNatHoleVisitor:     NatHoleVisitor{},
	TypeNatHoleClient:      NatHoleClient{},
	TypeNatHoleResp:        NatHoleResp{},
	TypeNatHoleSid:         NatHoleSid{},
	TypeNatHoleReport:      NatHoleReport{},
}

var TypeNameNatHoleResp = reflect.TypeOf(&NatHoleResp{}).Elem().Name()

// ClientSpec 客户端规格
type ClientSpec struct {
	// Type 由于支持虚拟客户端（VirtualClient），frps 需要知道客户端类型以便区分处理逻辑
	// 可选值：ssh-tunnel
	Type string `json:"type,omitempty"`
	// AlwaysAuthPass 如果值为 true，客户端将不需要身份验证
	AlwaysAuthPass bool `json:"always_auth_pass,omitempty"`
}

// Login 当 frpc 启动时，客户端发送此消息登录到服务器
type Login struct {
	// Version 版本号
	Version string `json:"version,omitempty"`
	// Hostname 主机名
	Hostname string `json:"hostname,omitempty"`
	// Os 操作系统
	Os string `json:"os,omitempty"`
	// Arch 架构
	Arch string `json:"arch,omitempty"`
	// User 用户名
	User string `json:"user,omitempty"`
	// PrivilegeKey 特权密钥
	PrivilegeKey string `json:"privilege_key,omitempty"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
	// RunID 运行 ID
	RunID string `json:"run_id,omitempty"`
	// ClientID 客户端 ID
	ClientID string `json:"client_id,omitempty"`
	// Metas 元数据
	Metas map[string]string `json:"metas,omitempty"`

	// ClientSpec 目前仅对虚拟客户端有效
	ClientSpec ClientSpec `json:"client_spec,omitempty"`

	// PoolCount 连接池数量，一些全局配置
	PoolCount int `json:"pool_count,omitempty"`
}

// LoginResp 登录响应
type LoginResp struct {
	// Version 版本号
	Version string `json:"version,omitempty"`
	// RunID 运行 ID
	RunID string `json:"run_id,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// NewProxy 当 frpc 登录成功后，发送此消息到 frps 以运行新的代理
type NewProxy struct {
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// ProxyType 代理类型
	ProxyType string `json:"proxy_type,omitempty"`
	// UseEncryption 是否使用加密
	UseEncryption bool `json:"use_encryption,omitempty"`
	// UseCompression 是否使用压缩
	UseCompression bool `json:"use_compression,omitempty"`
	// BandwidthLimit 带宽限制
	BandwidthLimit string `json:"bandwidth_limit,omitempty"`
	// BandwidthLimitMode 带宽限制模式
	BandwidthLimitMode string `json:"bandwidth_limit_mode,omitempty"`
	// Group 分组
	Group string `json:"group,omitempty"`
	// GroupKey 分组密钥
	GroupKey string `json:"group_key,omitempty"`
	// Metas 元数据
	Metas map[string]string `json:"metas,omitempty"`
	// Annotations 注解
	Annotations map[string]string `json:"annotations,omitempty"`

	// RemotePort 远程端口，仅 tcp 和 udp 有效
	RemotePort int `json:"remote_port,omitempty"`

	// CustomDomains 自定义域名，仅 http 和 https 有效
	CustomDomains []string `json:"custom_domains,omitempty"`
	// SubDomain 子域名
	SubDomain string `json:"subdomain,omitempty"`
	// Locations 路径
	Locations []string `json:"locations,omitempty"`
	// HTTPUser HTTP 用户名
	HTTPUser string `json:"http_user,omitempty"`
	// HTTPPwd HTTP 密码
	HTTPPwd string `json:"http_pwd,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"host_header_rewrite,omitempty"`
	// Headers 请求头
	Headers map[string]string `json:"headers,omitempty"`
	// ResponseHeaders 响应头
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	// RouteByHTTPUser 按 HTTP 用户路由
	RouteByHTTPUser string `json:"route_by_http_user,omitempty"`

	// Sk 密钥，仅 stcp、sudp、xtcp 有效
	Sk string `json:"sk,omitempty"`
	// AllowUsers 允许的用户列表
	AllowUsers []string `json:"allow_users,omitempty"`

	// Multiplexer 多路复用器，仅 tcpmux 有效
	Multiplexer string `json:"multiplexer,omitempty"`
}

// NewProxyResp 新建代理响应
type NewProxyResp struct {
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// RemoteAddr 远程地址
	RemoteAddr string `json:"remote_addr,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// CloseProxy 关闭代理
type CloseProxy struct {
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
}

// NewWorkConn 新建工作连接
type NewWorkConn struct {
	// RunID 运行 ID
	RunID string `json:"run_id,omitempty"`
	// PrivilegeKey 特权密钥
	PrivilegeKey string `json:"privilege_key,omitempty"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// ReqWorkConn 请求工作连接
type ReqWorkConn struct{}

// StartWorkConn 启动工作连接
type StartWorkConn struct {
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// SrcAddr 源地址
	SrcAddr string `json:"src_addr,omitempty"`
	// DstAddr 目标地址
	DstAddr string `json:"dst_addr,omitempty"`
	// SrcPort 源端口
	SrcPort uint16 `json:"src_port,omitempty"`
	// DstPort 目标端口
	DstPort uint16 `json:"dst_port,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// NewVisitorConn 新建访问者连接
type NewVisitorConn struct {
	// RunID 运行 ID
	RunID string `json:"run_id,omitempty"`
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// SignKey 签名密钥
	SignKey string `json:"sign_key,omitempty"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
	// UseEncryption 是否使用加密
	UseEncryption bool `json:"use_encryption,omitempty"`
	// UseCompression 是否使用压缩
	UseCompression bool `json:"use_compression,omitempty"`
}

// NewVisitorConnResp 新建访问者连接响应
type NewVisitorConnResp struct {
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// Ping 心跳
type Ping struct {
	// PrivilegeKey 特权密钥
	PrivilegeKey string `json:"privilege_key,omitempty"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// Pong 心跳响应
type Pong struct {
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// UDPPacket UDP 数据包
type UDPPacket struct {
	// Content 内容
	Content string `json:"c,omitempty"`
	// LocalAddr 本地地址
	LocalAddr *net.UDPAddr `json:"l,omitempty"`
	// RemoteAddr 远程地址
	RemoteAddr *net.UDPAddr `json:"r,omitempty"`
}

// NatHoleVisitor NAT 穿透访问者
type NatHoleVisitor struct {
	// TransactionID 事务 ID
	TransactionID string `json:"transaction_id,omitempty"`
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// PreCheck 是否预检查
	PreCheck bool `json:"pre_check,omitempty"`
	// Protocol 协议
	Protocol string `json:"protocol,omitempty"`
	// SignKey 签名密钥
	SignKey string `json:"sign_key,omitempty"`
	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
	// MappedAddrs 映射地址列表
	MappedAddrs []string `json:"mapped_addrs,omitempty"`
	// AssistedAddrs 辅助地址列表
	AssistedAddrs []string `json:"assisted_addrs,omitempty"`
}

// NatHoleClient NAT 穿透客户端
type NatHoleClient struct {
	// TransactionID 事务 ID
	TransactionID string `json:"transaction_id,omitempty"`
	// ProxyName 代理名称
	ProxyName string `json:"proxy_name,omitempty"`
	// Sid 会话 ID
	Sid string `json:"sid,omitempty"`
	// MappedAddrs 映射地址列表
	MappedAddrs []string `json:"mapped_addrs,omitempty"`
	// AssistedAddrs 辅助地址列表
	AssistedAddrs []string `json:"assisted_addrs,omitempty"`
}

// PortsRange 端口范围
type PortsRange struct {
	// From 起始端口
	From int `json:"from,omitempty"`
	// To 结束端口
	To int `json:"to,omitempty"`
}

// NatHoleDetectBehavior NAT 穿透检测行为
type NatHoleDetectBehavior struct {
	// Role 角色，sender 或 receiver
	Role string `json:"role,omitempty"`
	// Mode 模式，0、1、2...
	Mode int `json:"mode,omitempty"`
	// TTL 生存时间
	TTL int `json:"ttl,omitempty"`
	// SendDelayMs 发送延迟（毫秒）
	SendDelayMs int `json:"send_delay_ms,omitempty"`
	// ReadTimeoutMs 读取超时（毫秒）
	ReadTimeoutMs int `json:"read_timeout,omitempty"`
	// CandidatePorts 候选端口列表
	CandidatePorts []PortsRange `json:"candidate_ports,omitempty"`
	// SendRandomPorts 发送随机端口数量
	SendRandomPorts int `json:"send_random_ports,omitempty"`
	// ListenRandomPorts 监听随机端口数量
	ListenRandomPorts int `json:"listen_random_ports,omitempty"`
}

// NatHoleResp NAT 穿透响应
type NatHoleResp struct {
	// TransactionID 事务 ID
	TransactionID string `json:"transaction_id,omitempty"`
	// Sid 会话 ID
	Sid string `json:"sid,omitempty"`
	// Protocol 协议
	Protocol string `json:"protocol,omitempty"`
	// CandidateAddrs 候选地址列表
	CandidateAddrs []string `json:"candidate_addrs,omitempty"`
	// AssistedAddrs 辅助地址列表
	AssistedAddrs []string `json:"assisted_addrs,omitempty"`
	// DetectBehavior 检测行为
	DetectBehavior NatHoleDetectBehavior `json:"detect_behavior,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// NatHoleSid NAT 穿透会话 ID
type NatHoleSid struct {
	// TransactionID 事务 ID
	TransactionID string `json:"transaction_id,omitempty"`
	// Sid 会话 ID
	Sid string `json:"sid,omitempty"`
	// Response 是否为响应
	Response bool `json:"response,omitempty"`
	// Nonce 随机数
	Nonce string `json:"nonce,omitempty"`
}

// NatHoleReport NAT 穿透报告
type NatHoleReport struct {
	// Sid 会话 ID
	Sid string `json:"sid,omitempty"`
	// Success 是否成功
	Success bool `json:"success,omitempty"`
}
