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

package validation

import (
	"errors"
	"fmt"
	"slices"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// ValidateVisitorConfigurer 验证访问者配置器的有效性
// 参数 c 为访问者配置器对象
// 返回验证过程中产生的错误信息
func ValidateVisitorConfigurer(c v1.VisitorConfigurer) error {
	base := c.GetBaseConfig()
	// 验证基础配置
	if err := validateVisitorBaseConfig(base); err != nil {
		return err
	}

	// 根据访问者类型进行相应的验证
	switch v := c.(type) {
	case *v1.STCPVisitorConfig:
	case *v1.SUDPVisitorConfig:
	case *v1.XTCPVisitorConfig:
		return validateXTCPVisitorConfig(v)
	default:
		return errors.New("未知的访问者配置类型")
	}
	return nil
}

// validateVisitorBaseConfig 验证访问者基础配置的有效性
// 参数 c 为访问者基础配置对象
// 返回验证过程中产生的错误信息
func validateVisitorBaseConfig(c *v1.VisitorBaseConfig) error {
	// 验证名称是否配置
	if c.Name == "" {
		return errors.New("名称是必需的")
	}

	// 验证服务端名称是否配置
	if c.ServerName == "" {
		return errors.New("服务端名称是必需的")
	}

	// 验证绑定端口是否配置
	if c.BindPort == 0 {
		return errors.New("绑定端口是必需的")
	}
	return nil
}

// validateXTCPVisitorConfig 验证 XTCP 访问者配置的有效性
// 参数 c 为 XTCP 访问者配置对象
// 返回验证过程中产生的错误信息
func validateXTCPVisitorConfig(c *v1.XTCPVisitorConfig) error {
	// 验证协议是否为 kcp 或 quic
	if !slices.Contains([]string{"kcp", "quic"}, c.Protocol) {
		return fmt.Errorf("协议应为 kcp 或 quic")
	}
	return nil
}
