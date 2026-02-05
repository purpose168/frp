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
	"sync"

	"github.com/fatedier/frp/pkg/util/util"
)

// TODO(fatedier): 由于当前 Go JSON 库的实现问题，自定义结构体的 UnmarshalJSON 方法
// 无法访问父级解码器的 DisallowUnknownFields 参数。
// 这里临时使用全局变量来控制是否允许未知字段。
// 一旦社区实现了 v2 版本，我们可以切换到标准化的方法。
//
// https://github.com/golang/go/issues/41144
// https://github.com/golang/go/discussions/63397
var (
	// DisallowUnknownFields 是否禁止未知字段
	DisallowUnknownFields = false
	// DisallowUnknownFieldsMu DisallowUnknownFields 的互斥锁
	DisallowUnknownFieldsMu sync.Mutex
)

// AuthScope 认证作用域类型
type AuthScope string

const (
	// AuthScopeHeartBeats 心跳作用域
	AuthScopeHeartBeats AuthScope = "HeartBeats"
	// AuthScopeNewWorkConns 新工作连接作用域
	AuthScopeNewWorkConns AuthScope = "NewWorkConns"
)

// AuthMethod 认证方法类型
type AuthMethod string

const (
	// AuthMethodToken Token 认证方法
	AuthMethodToken AuthMethod = "token"
	// AuthMethodOIDC OIDC 认证方法
	AuthMethodOIDC AuthMethod = "oidc"
)

// QUICOptions QUIC 协议选项
type QUICOptions struct {
	// KeepalivePeriod 保活周期（秒）
	KeepalivePeriod int `json:"keepalivePeriod,omitempty"`
	// MaxIdleTimeout 最大空闲超时（秒）
	MaxIdleTimeout int `json:"maxIdleTimeout,omitempty"`
	// MaxIncomingStreams 最大传入流数
	MaxIncomingStreams int `json:"maxIncomingStreams,omitempty"`
}

// Complete 填充 QUIC 选项的默认值
func (c *QUICOptions) Complete() {
	// 设置默认的保活周期
	c.KeepalivePeriod = util.EmptyOr(c.KeepalivePeriod, 10)
	// 设置默认的最大空闲超时
	c.MaxIdleTimeout = util.EmptyOr(c.MaxIdleTimeout, 30)
	// 设置默认的最大传入流数
	c.MaxIncomingStreams = util.EmptyOr(c.MaxIncomingStreams, 100000)
}

// WebServerConfig Web 服务器配置结构体
type WebServerConfig struct {
	// Addr 用于提供 Web 界面和 API 的网络绑定地址
	// 默认值为 "127.0.0.1"
	Addr string `json:"addr,omitempty"`
	// Port 指定 Web 服务器监听的端口
	// 如果此值为 0，则不会启动管理服务器
	Port int `json:"port,omitempty"`
	// User 指定 Web 服务器用于登录的用户名
	User string `json:"user,omitempty"`
	// Password 指定管理服务器用于登录的密码
	Password string `json:"password,omitempty"`
	// AssetsDir 指定管理服务器加载资源的本地目录
	// 如果此值为空，将从使用 embed 包打包的可执行文件中加载资源
	AssetsDir string `json:"assetsDir,omitempty"`
	// PprofEnable 启用 golang pprof 处理器
	PprofEnable bool `json:"pprofEnable,omitempty"`
	// TLS 如果 TLSConfig 不为 nil，则启用 TLS
	TLS *TLSConfig `json:"tls,omitempty"`
}

// Complete 填充 Web 服务器配置的默认值
func (c *WebServerConfig) Complete() {
	// 设置默认的绑定地址
	c.Addr = util.EmptyOr(c.Addr, "127.0.0.1")
}

// TLSConfig TLS 配置结构体
type TLSConfig struct {
	// CertFile 指定客户端将加载的证书文件路径
	CertFile string `json:"certFile,omitempty"`
	// KeyFile 指定客户端将加载的密钥文件路径
	KeyFile string `json:"keyFile,omitempty"`
	// TrustedCaFile 指定要加载的受信任 CA 文件路径
	TrustedCaFile string `json:"trustedCaFile,omitempty"`
	// ServerName 指定 TLS 证书的自定义服务器名称
	// 默认情况下，服务器名称与 ServerAddr 相同
	ServerName string `json:"serverName,omitempty"`
}

// NatTraversalConfig NAT 穿透配置选项
type NatTraversalConfig struct {
	// DisableAssistedAddrs 禁用本地网络接口进行 NAT 穿透期间的辅助连接
	// 启用后，将仅使用 STUN 发现的公共地址
	DisableAssistedAddrs bool `json:"disableAssistedAddrs,omitempty"`
}

// LogConfig 日志配置结构体
type LogConfig struct {
	// To 指定 frp 应该写入日志的目标
	// 如果使用 "console"，日志将打印到标准输出，否则将写入到指定文件
	// 默认值为 "console"
	To string `json:"to,omitempty"`
	// Level 指定最小日志级别
	// 有效值为 "trace"、"debug"、"info"、"warn" 和 "error"
	// 默认值为 "info"
	Level string `json:"level,omitempty"`
	// MaxDays 指定删除前存储日志信息的最大天数
	MaxDays int64 `json:"maxDays"`
	// DisablePrintColor 当 log.to 为 "console" 时禁用日志颜色
	DisablePrintColor bool `json:"disablePrintColor,omitempty"`
}

// Complete 填充日志配置的默认值
func (c *LogConfig) Complete() {
	// 设置默认的日志输出目标
	c.To = util.EmptyOr(c.To, "console")
	// 设置默认的日志级别
	c.Level = util.EmptyOr(c.Level, "info")
	// 设置默认的日志保留天数
	c.MaxDays = util.EmptyOr(c.MaxDays, 3)
}

// HTTPPluginOptions HTTP 插件选项结构体
type HTTPPluginOptions struct {
	// Name 插件名称
	Name string `json:"name"`
	// Addr 插件地址
	Addr string `json:"addr"`
	// Path 插件路径
	Path string `json:"path"`
	// Ops 插件操作列表
	Ops []string `json:"ops"`
	// TLSVerify 是否验证 TLS 证书
	TLSVerify bool `json:"tlsVerify,omitempty"`
}

// HeaderOperations HTTP 头部操作结构体
type HeaderOperations struct {
	// Set 要设置的 HTTP 头部映射
	Set map[string]string `json:"set,omitempty"`
}

// HTTPHeader HTTP 头部结构体
type HTTPHeader struct {
	// Name 头部名称
	Name string `json:"name"`
	// Value 头部值
	Value string `json:"value"`
}
