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

	"github.com/purpose168/frp/pkg/util/util"
)

// VisitorTransport 访客传输配置结构体
type VisitorTransport struct {
	UseEncryption  bool `json:"useEncryption,omitempty"`
	UseCompression bool `json:"useCompression,omitempty"`
}

// VisitorBaseConfig 访客基础配置结构体
type VisitorBaseConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Enabled 控制是否启用此访问客。nil 或 true 表示启用，false 表示禁用
	// 这允许单独控制每个访问客，补充全局 "start" 字段
	Enabled   *bool            `json:"enabled,omitempty"`
	Transport VisitorTransport `json:"transport,omitempty"`
	SecretKey string           `json:"secretKey,omitempty"`
	// 如果未设置服务器用户，则默认为当前用户
	ServerUser string `json:"serverUser,omitempty"`
	ServerName string `json:"serverName,omitempty"`
	BindAddr   string `json:"bindAddr,omitempty"`
	// BindPort 是访问客监听的端口
	// 它可以小于 0，这意味着不绑定到端口，仅接收从其他访问客重定向的连接（SUDP 目前不支持此功能）
	BindPort int `json:"bindPort,omitempty"`

	// Plugin 指定应使用的插件
	Plugin TypedVisitorPluginOptions `json:"plugin,omitempty"`
}

// GetBaseConfig 获取基础配置
func (c *VisitorBaseConfig) GetBaseConfig() *VisitorBaseConfig {
	return c
}

// Complete 完成访问客基础配置
func (c *VisitorBaseConfig) Complete(g *ClientCommonConfig) {
	if c.BindAddr == "" {
		c.BindAddr = "127.0.0.1"
	}

	namePrefix := ""
	if g.User != "" {
		namePrefix = g.User + "."
	}
	c.Name = namePrefix + c.Name

	if c.ServerUser != "" {
		c.ServerName = c.ServerUser + "." + c.ServerName
	} else {
		c.ServerName = namePrefix + c.ServerName
	}
}

// VisitorConfigurer 访客配置器接口
type VisitorConfigurer interface {
	Complete(*ClientCommonConfig)
	GetBaseConfig() *VisitorBaseConfig
}

// VisitorType 访客类型
type VisitorType string

const (
	VisitorTypeSTCP VisitorType = "stcp"
	VisitorTypeXTCP VisitorType = "xtcp"
	VisitorTypeSUDP VisitorType = "sudp"
)

var visitorConfigTypeMap = map[VisitorType]reflect.Type{
	VisitorTypeSTCP: reflect.TypeOf(STCPVisitorConfig{}),
	VisitorTypeXTCP: reflect.TypeOf(XTCPVisitorConfig{}),
	VisitorTypeSUDP: reflect.TypeOf(SUDPVisitorConfig{}),
}

// TypedVisitorConfig 类型化的访问客配置结构体
type TypedVisitorConfig struct {
	Type string `json:"type"`
	VisitorConfigurer
}

// UnmarshalJSON 实现自定义的 JSON 反序列化
func (c *TypedVisitorConfig) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return errors.New("类型是必需的")
	}

	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	configurer := NewVisitorConfigurerByType(VisitorType(typeStruct.Type))
	if configurer == nil {
		return fmt.Errorf("未知的访问客类型: %s", typeStruct.Type)
	}
	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(configurer); err != nil {
		return fmt.Errorf("反序列化 VisitorConfig 错误: %v", err)
	}
	c.VisitorConfigurer = configurer
	return nil
}

// MarshalJSON 实现自定义的 JSON 序列化
func (c *TypedVisitorConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.VisitorConfigurer)
}

// NewVisitorConfigurerByType 根据类型创建访问客配置器
func NewVisitorConfigurerByType(t VisitorType) VisitorConfigurer {
	v, ok := visitorConfigTypeMap[t]
	if !ok {
		return nil
	}
	vc := reflect.New(v).Interface().(VisitorConfigurer)
	vc.GetBaseConfig().Type = string(t)
	return vc
}

var _ VisitorConfigurer = &STCPVisitorConfig{}

// STCPVisitorConfig STCP 访客配置结构体
type STCPVisitorConfig struct {
	VisitorBaseConfig
}

var _ VisitorConfigurer = &SUDPVisitorConfig{}

// SUDPVisitorConfig SUDP 访客配置结构体
type SUDPVisitorConfig struct {
	VisitorBaseConfig
}

var _ VisitorConfigurer = &XTCPVisitorConfig{}

// XTCPVisitorConfig XTCP 访客配置结构体
type XTCPVisitorConfig struct {
	VisitorBaseConfig

	Protocol          string `json:"protocol,omitempty"`
	KeepTunnelOpen    bool   `json:"keepTunnelOpen,omitempty"`
	MaxRetriesAnHour  int    `json:"maxRetriesAnHour,omitempty"`
	MinRetryInterval  int    `json:"minRetryInterval,omitempty"`
	FallbackTo        string `json:"fallbackTo,omitempty"`
	FallbackTimeoutMs int    `json:"fallbackTimeoutMs,omitempty"`

	// NatTraversal NAT 穿透配置
	NatTraversal *NatTraversalConfig `json:"natTraversal,omitempty"`
}

// Complete 完成 XTCP 访客配置
func (c *XTCPVisitorConfig) Complete(g *ClientCommonConfig) {
	c.VisitorBaseConfig.Complete(g)

	c.Protocol = util.EmptyOr(c.Protocol, "quic")
	c.MaxRetriesAnHour = util.EmptyOr(c.MaxRetriesAnHour, 8)
	c.MinRetryInterval = util.EmptyOr(c.MinRetryInterval, 90)
	c.FallbackTimeoutMs = util.EmptyOr(c.FallbackTimeoutMs, 1000)

	if c.FallbackTo != "" {
		c.FallbackTo = lo.Ternary(g.User == "", "", g.User+".") + c.FallbackTo
	}
}
