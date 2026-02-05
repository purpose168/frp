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

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
)

// ValidateServerConfig 验证服务端配置的有效性
// 参数 c 为服务端配置对象
// 返回验证过程中产生的警告和错误信息
func (v *ConfigValidator) ValidateServerConfig(c *v1.ServerConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)
	// 验证认证方法是否在支持的范围内
	if !slices.Contains(SupportedAuthMethods, c.Auth.Method) {
		errs = AppendError(errs, fmt.Errorf("无效的认证方法，可选值为 %v", SupportedAuthMethods))
	}
	// 验证额外的认证作用域是否在支持的范围内
	if !lo.Every(SupportedAuthAdditionalScopes, c.Auth.AdditionalScopes) {
		errs = AppendError(errs, fmt.Errorf("无效的认证额外作用域，可选值为 %v", SupportedAuthAdditionalScopes))
	}

	// 验证 token 和 tokenSource 的互斥性
	if c.Auth.Token != "" && c.Auth.TokenSource != nil {
		errs = AppendError(errs, fmt.Errorf("不能同时指定 auth.token 和 auth.tokenSource"))
	}

	// 如果指定了 tokenSource，则验证其有效性
	if c.Auth.TokenSource != nil {
		// 检查是否为 exec 类型的 tokenSource
		if c.Auth.TokenSource.Type == "exec" {
			if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
				errs = AppendError(errs, err)
			}
		}
		// 验证 tokenSource 配置
		if err := c.Auth.TokenSource.Validate(); err != nil {
			errs = AppendError(errs, fmt.Errorf("无效的 auth.tokenSource: %v", err))
		}
	}

	// 验证日志配置
	if err := validateLogConfig(&c.Log); err != nil {
		errs = AppendError(errs, err)
	}

	// 验证 Web 服务器配置
	if err := validateWebServerConfig(&c.WebServer); err != nil {
		errs = AppendError(errs, err)
	}

	// 验证各个端口号的有效性
	errs = AppendError(errs, ValidatePort(c.BindPort, "bindPort"))
	errs = AppendError(errs, ValidatePort(c.KCPBindPort, "kcpBindPort"))
	errs = AppendError(errs, ValidatePort(c.QUICBindPort, "quicBindPort"))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPPort, "vhostHTTPPort"))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPSPort, "vhostHTTPSPort"))
	errs = AppendError(errs, ValidatePort(c.TCPMuxHTTPConnectPort, "tcpMuxHTTPConnectPort"))

	// 验证 HTTP 插件操作的有效性
	for _, p := range c.HTTPPlugins {
		if !lo.Every(SupportedHTTPPluginOps, p.Ops) {
			errs = AppendError(errs, fmt.Errorf("无效的 http 插件操作，可选值为 %v", SupportedHTTPPluginOps))
		}
	}
	return warnings, errs
}
