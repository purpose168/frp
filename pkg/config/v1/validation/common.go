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
	"fmt"
	"slices"

	v1 "github.com/purpose168/frp/pkg/config/v1"
)

// validateWebServerConfig 验证 Web 服务器配置的有效性
// 参数 c 为 Web 服务器配置对象
// 返回验证过程中产生的错误信息
func validateWebServerConfig(c *v1.WebServerConfig) error {
	// 检查 TLS 配置是否存在
	if c.TLS != nil {
		// 验证 TLS 证书文件是否配置
		if c.TLS.CertFile == "" {
			return fmt.Errorf("启用 TLS 时必须指定 tls.certFile")
		}
		// 验证 TLS 密钥文件是否配置
		if c.TLS.KeyFile == "" {
			return fmt.Errorf("启用 TLS 时必须指定 tls.keyFile")
		}
	}

	// 验证端口号的有效性
	return ValidatePort(c.Port, "webServer.port")
}

// ValidatePort 检查网络端口号是否在有效范围内
// 参数 port 为端口号
// 参数 fieldPath 为配置字段路径，用于错误提示
// 返回验证过程中产生的错误信息
func ValidatePort(port int, fieldPath string) error {
	// 检查端口号是否在 0 到 65535 之间
	if 0 <= port && port <= 65535 {
		return nil
	}
	return fmt.Errorf("%s: 端口号 %d 必须在 0..65535 范围内", fieldPath, port)
}

// validateLogConfig 验证日志配置的有效性
// 参数 c 为日志配置对象
// 返回验证过程中产生的错误信息
func validateLogConfig(c *v1.LogConfig) error {
	// 验证日志级别是否在支持的范围内
	if !slices.Contains(SupportedLogLevels, c.Level) {
		return fmt.Errorf("无效的日志级别，可选值为 %v", SupportedLogLevels)
	}
	return nil
}
