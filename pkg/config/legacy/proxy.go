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

// 版权所有 2023 The frp Authors
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"基础分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的管理权限和限制，请参阅许可证。

package legacy

import (
	"fmt"
	"reflect"

	"gopkg.in/ini.v1"

	"github.com/purpose168/frp/pkg/config/types"
)

type ProxyType string

const (
	ProxyTypeTCP    ProxyType = "tcp"
	ProxyTypeUDP    ProxyType = "udp"
	ProxyTypeTCPMUX ProxyType = "tcpmux"
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSTCP   ProxyType = "stcp"
	ProxyTypeXTCP   ProxyType = "xtcp"
	ProxyTypeSUDP   ProxyType = "sudp"
)

// 代理
var (
	proxyConfTypeMap = map[ProxyType]reflect.Type{
		ProxyTypeTCP:    reflect.TypeOf(TCPProxyConf{}),
		ProxyTypeUDP:    reflect.TypeOf(UDPProxyConf{}),
		ProxyTypeTCPMUX: reflect.TypeOf(TCPMuxProxyConf{}),
		ProxyTypeHTTP:   reflect.TypeOf(HTTPProxyConf{}),
		ProxyTypeHTTPS:  reflect.TypeOf(HTTPSProxyConf{}),
		ProxyTypeSTCP:   reflect.TypeOf(STCPProxyConf{}),
		ProxyTypeXTCP:   reflect.TypeOf(XTCPProxyConf{}),
		ProxyTypeSUDP:   reflect.TypeOf(SUDPProxyConf{}),
	}
)

type ProxyConf interface {
	// GetBaseConfig 返回此配置的 BaseProxyConf。
	GetBaseConfig() *BaseProxyConf
	// UnmarshalFromIni 将 ini.Section 解组到此配置。此函数将在 frpc 端调用。
	UnmarshalFromIni(string, string, *ini.Section) error
}

func NewConfByType(proxyType ProxyType) ProxyConf {
	v, ok := proxyConfTypeMap[proxyType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(ProxyConf)
	return cfg
}

// 代理配置加载器
// DefaultProxyConf 通过 proxyType 创建一个空的 ProxyConf 对象。
// 如果 proxyType 不存在，则返回 nil。
func DefaultProxyConf(proxyType ProxyType) ProxyConf {
	return NewConfByType(proxyType)
}

// 从 ini 加载的代理
func NewProxyConfFromIni(prefix, name string, section *ini.Section) (ProxyConf, error) {
	// section.Key: 如果键不存在，section 将使用默认值设置它。
	proxyType := ProxyType(section.Key("type").String())
	if proxyType == "" {
		proxyType = ProxyTypeTCP
	}

	conf := DefaultProxyConf(proxyType)
	if conf == nil {
		return nil, fmt.Errorf("无效的类型 [%s]", proxyType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, err
	}
	return conf, nil
}

// LocalSvrConf 配置客户端将连接到的位置，或将使用的插件。
type LocalSvrConf struct {
	// LocalIP 指定要连接的 IP 地址或主机名。
	LocalIP string `ini:"local_ip" json:"local_ip"`
	// LocalPort 指定要连接的端口。
	LocalPort int `ini:"local_port" json:"local_port"`

	// Plugin 指定应使用什么插件。如果设置了此值，则将忽略 LocalIp 和 LocalPort 值。
	// 默认情况下，此值为 ""。
	Plugin string `ini:"plugin" json:"plugin"`
	// PluginParams 指定要传递给插件的参数（如果正在使用插件）。
	// 默认情况下，此值为空映射。
	PluginParams map[string]string `ini:"-"`
}

// HealthCheckConf 配置健康检查。这对于负载平衡目的很有用，可以检测并删除到失败服务的代理。
type HealthCheckConf struct {
	// HealthCheckType 指定用于健康检查的协议。
	// 有效值包括 "tcp"、"http" 和 ""。如果此值为 ""，则不会执行健康检查。
	// 默认情况下，此值为 ""。
	//
	// 如果类型为 "tcp"，将尝试连接到目标服务器。如果无法建立连接，则健康检查失败。
	//
	// 如果类型为 "http"，将向 HealthCheckURL 指定的端点发出 GET 请求。
	// 如果响应不是 200，则健康检查失败。
	HealthCheckType string `ini:"health_check_type" json:"health_check_type"` // tcp | http
	// HealthCheckTimeoutS 指定等待健康检查尝试连接的秒数。
	// 如果达到超时，则计为健康检查失败。默认情况下，此值为 3。
	HealthCheckTimeoutS int `ini:"health_check_timeout_s" json:"health_check_timeout_s"`
	// HealthCheckMaxFailed 指定停止之前允许的失败次数。
	// 默认情况下，此值为 1。
	HealthCheckMaxFailed int `ini:"health_check_max_failed" json:"health_check_max_failed"`
	// HealthCheckIntervalS 指定健康检查之间的时间（以秒为单位）。
	// 默认情况下，此值为 10。
	HealthCheckIntervalS int `ini:"health_check_interval_s" json:"health_check_interval_s"`
	// HealthCheckURL 指定如果健康检查类型为 "http" 时发送健康检查的地址。
	HealthCheckURL string `ini:"health_check_url" json:"health_check_url"`
	// HealthCheckAddr 指定如果健康检查类型为 "tcp" 时连接的地址。
	HealthCheckAddr string `ini:"-"`
}

// BaseProxyConf 提供所有类型通用的配置信息。
type BaseProxyConf struct {
	// ProxyName 是此代理的名称
	ProxyName string `ini:"name" json:"name"`
	// ProxyType 指定此代理的类型。有效值包括 "tcp"、"udp"、"http"、"https"、"stcp" 和 "xtcp"。
	// 默认情况下，此值为 "tcp"。
	ProxyType string `ini:"type" json:"type"`

	// UseEncryption 控制是否加密与服务器的通信。
	// 加密使用服务器和客户端配置中提供的令牌完成。
	// 默认情况下，此值为 false。
	UseEncryption bool `ini:"use_encryption" json:"use_encryption"`
	// UseCompression 控制是否压缩与服务器的通信。
	// 默认情况下，此值为 false。
	UseCompression bool `ini:"use_compression" json:"use_compression"`
	// Group 指定此代理所属的组。服务器将使用此信息对同一组中的代理进行负载平衡。
	// 如果值为 ""，则不会在组中。默认情况下，此值为 ""。
	Group string `ini:"group" json:"group"`
	// GroupKey 指定组密钥，该密钥在同一组的代理之间应该相同。
	// 默认情况下，此值为 ""。
	GroupKey string `ini:"group_key" json:"group_key"`

	// ProxyProtocolVersion 指定要使用的协议版本。
	// 有效值包括 "v1"、"v2" 和 ""。如果值为 ""，则将自动选择协议版本。
	// 默认情况下，此值为 ""。
	ProxyProtocolVersion string `ini:"proxy_protocol_version" json:"proxy_protocol_version"`

	// BandwidthLimit 限制带宽
	// 0 表示无限制
	BandwidthLimit types.BandwidthQuantity `ini:"bandwidth_limit" json:"bandwidth_limit"`
	// BandwidthLimitMode 指定是在客户端还是服务器端限制带宽。
	// 有效值包括 "client" 和 "server"。默认情况下，此值为 "client"。
	BandwidthLimitMode string `ini:"bandwidth_limit_mode" json:"bandwidth_limit_mode"`

	// 每个代理的元信息
	Metas map[string]string `ini:"-" json:"metas"`

	LocalSvrConf    `ini:",extends"`
	HealthCheckConf `ini:",extends"`
}

// Base
func (cfg *BaseProxyConf) GetBaseConfig() *BaseProxyConf {
	return cfg
}

// BaseProxyConf 应用自定义逻辑更改。
func (cfg *BaseProxyConf) decorate(_ string, name string, section *ini.Section) error {
	cfg.ProxyName = name
	// metas_xxx
	cfg.Metas = GetMapWithoutPrefix(section.KeysHash(), "meta_")

	// bandwidth_limit
	if bandwidth, err := section.GetKey("bandwidth_limit"); err == nil {
		cfg.BandwidthLimit, err = types.NewBandwidthQuantity(bandwidth.String())
		if err != nil {
			return err
		}
	}

	// plugin_xxx
	cfg.PluginParams = GetMapByPrefix(section.KeysHash(), "plugin_")
	return nil
}

type DomainConf struct {
	CustomDomains []string `ini:"custom_domains" json:"custom_domains"`
	SubDomain     string   `ini:"subdomain" json:"subdomain"`
}

type RoleServerCommonConf struct {
	Role       string   `ini:"role" json:"role"`
	Sk         string   `ini:"sk" json:"sk"`
	AllowUsers []string `ini:"allow_users" json:"allow_users"`
}

// HTTP
type HTTPProxyConf struct {
	BaseProxyConf `ini:",extends"`
	DomainConf    `ini:",extends"`

	Locations         []string          `ini:"locations" json:"locations"`
	HTTPUser          string            `ini:"http_user" json:"http_user"`
	HTTPPwd           string            `ini:"http_pwd" json:"http_pwd"`
	HostHeaderRewrite string            `ini:"host_header_rewrite" json:"host_header_rewrite"`
	Headers           map[string]string `ini:"-" json:"headers"`
	RouteByHTTPUser   string            `ini:"route_by_http_user" json:"route_by_http_user"`
}

func (cfg *HTTPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）
	cfg.Headers = GetMapWithoutPrefix(section.KeysHash(), "header_")
	return nil
}

// HTTPS
type HTTPSProxyConf struct {
	BaseProxyConf `ini:",extends"`
	DomainConf    `ini:",extends"`
}

func (cfg *HTTPSProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）
	return nil
}

// TCP
type TCPProxyConf struct {
	BaseProxyConf `ini:",extends"`
	RemotePort    int `ini:"remote_port" json:"remote_port"`
}

func (cfg *TCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）

	return nil
}

// UDP
type UDPProxyConf struct {
	BaseProxyConf `ini:",extends"`

	RemotePort int `ini:"remote_port" json:"remote_port"`
}

func (cfg *UDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）

	return nil
}

// TCPMux
type TCPMuxProxyConf struct {
	BaseProxyConf   `ini:",extends"`
	DomainConf      `ini:",extends"`
	HTTPUser        string `ini:"http_user" json:"http_user,omitempty"`
	HTTPPwd         string `ini:"http_pwd" json:"http_pwd,omitempty"`
	RouteByHTTPUser string `ini:"route_by_http_user" json:"route_by_http_user"`

	Multiplexer string `ini:"multiplexer"`
}

func (cfg *TCPMuxProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）

	return nil
}

// STCP
type STCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *STCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）
	if cfg.Role == "" {
		cfg.Role = "server"
	}
	return nil
}

// XTCP
type XTCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *XTCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）
	if cfg.Role == "" {
		cfg.Role = "server"
	}
	return nil
}

// SUDP
type SUDPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *SUDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// 添加自定义逻辑解组（如果存在）
	return nil
}

func preUnmarshalFromIni(cfg ProxyConf, prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.GetBaseConfig().decorate(prefix, name, section)
	if err != nil {
		return err
	}

	return nil
}
