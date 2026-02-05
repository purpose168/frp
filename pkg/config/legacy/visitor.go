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
)

type VisitorType string

const (
	VisitorTypeSTCP VisitorType = "stcp"
	VisitorTypeXTCP VisitorType = "xtcp"
	VisitorTypeSUDP VisitorType = "sudp"
)

// 访问者
var (
	visitorConfTypeMap = map[VisitorType]reflect.Type{
		VisitorTypeSTCP: reflect.TypeOf(STCPVisitorConf{}),
		VisitorTypeXTCP: reflect.TypeOf(XTCPVisitorConf{}),
		VisitorTypeSUDP: reflect.TypeOf(SUDPVisitorConf{}),
	}
)

type VisitorConf interface {
	// GetBaseConfig 返回访问者的基础配置。
	GetBaseConfig() *BaseVisitorConf
	// UnmarshalFromIni 从 ini 解组配置。
	UnmarshalFromIni(prefix string, name string, section *ini.Section) error
}

// DefaultVisitorConf 通过 visitorType 创建一个空的 VisitorConf 对象。
// 如果 visitorType 不存在，则返回 nil。
func DefaultVisitorConf(visitorType VisitorType) VisitorConf {
	v, ok := visitorConfTypeMap[visitorType]
	if !ok {
		return nil
	}
	return reflect.New(v).Interface().(VisitorConf)
}

type BaseVisitorConf struct {
	ProxyName      string `ini:"name" json:"name"`
	ProxyType      string `ini:"type" json:"type"`
	UseEncryption  bool   `ini:"use_encryption" json:"use_encryption"`
	UseCompression bool   `ini:"use_compression" json:"use_compression"`
	Role           string `ini:"role" json:"role"`
	Sk             string `ini:"sk" json:"sk"`
	// 如果未设置服务器用户，则默认为当前用户
	ServerUser string `ini:"server_user" json:"server_user"`
	ServerName string `ini:"server_name" json:"server_name"`
	BindAddr   string `ini:"bind_addr" json:"bind_addr"`
	// BindPort 是访问者监听的端口。
	// 它可以小于 0，这意味着不绑定到端口，仅接收从其他访问者重定向的连接。
	// （目前 SUDP 不支持此功能）
	BindPort int `ini:"bind_port" json:"bind_port"`
}

// 基础
func (cfg *BaseVisitorConf) GetBaseConfig() *BaseVisitorConf {
	return cfg
}

func (cfg *BaseVisitorConf) unmarshalFromIni(_ string, name string, _ *ini.Section) error {
	// 基本解组后的自定义装饰：
	cfg.ProxyName = name

	// bind_addr
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}
	return nil
}

func preVisitorUnmarshalFromIni(cfg VisitorConf, prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.GetBaseConfig().unmarshalFromIni(prefix, name, section)
	if err != nil {
		return err
	}
	return nil
}

type SUDPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

func (cfg *SUDPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// 添加自定义逻辑解组（如果存在）

	return
}

type STCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

func (cfg *STCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// 添加自定义逻辑解组（如果存在）

	return
}

type XTCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`

	Protocol          string `ini:"protocol" json:"protocol,omitempty"`
	KeepTunnelOpen    bool   `ini:"keep_tunnel_open" json:"keep_tunnel_open,omitempty"`
	MaxRetriesAnHour  int    `ini:"max_retries_an_hour" json:"max_retries_an_hour,omitempty"`
	MinRetryInterval  int    `ini:"min_retry_interval" json:"min_retry_interval,omitempty"`
	FallbackTo        string `ini:"fallback_to" json:"fallback_to,omitempty"`
	FallbackTimeoutMs int    `ini:"fallback_timeout_ms" json:"fallback_timeout_ms,omitempty"`
}

func (cfg *XTCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// 添加自定义逻辑解组（如果存在）
	if cfg.Protocol == "" {
		cfg.Protocol = "quic"
	}
	if cfg.MaxRetriesAnHour <= 0 {
		cfg.MaxRetriesAnHour = 8
	}
	if cfg.MinRetryInterval <= 0 {
		cfg.MinRetryInterval = 90
	}
	if cfg.FallbackTimeoutMs <= 0 {
		cfg.FallbackTimeoutMs = 1000
	}
	return
}

// 从 ini 加载的访问者
func NewVisitorConfFromIni(prefix string, name string, section *ini.Section) (VisitorConf, error) {
	// section.Key: 如果键不存在，section 将使用默认值设置它。
	visitorType := VisitorType(section.Key("type").String())

	if visitorType == "" {
		return nil, fmt.Errorf("类型不应为空")
	}

	conf := DefaultVisitorConf(visitorType)
	if conf == nil {
		return nil, fmt.Errorf("类型 [%s] 错误", visitorType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, fmt.Errorf("类型 [%s] 错误", visitorType)
	}
	return conf, nil
}
