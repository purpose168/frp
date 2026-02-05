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
	"os"
	"path/filepath"
	"slices"

	"github.com/samber/lo"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/policy/featuregate"
	"github.com/purpose168/frp/pkg/policy/security"
)

// ValidateClientCommonConfig 验证客户端通用配置的有效性
// 参数 c 为客户端通用配置对象
// 返回验证过程中产生的警告和错误信息
func (v *ConfigValidator) ValidateClientCommonConfig(c *v1.ClientCommonConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)

	// 定义验证器函数列表，用于验证客户端配置的各个方面
	validators := []func() (Warning, error){
		func() (Warning, error) { return validateFeatureGates(c) },
		func() (Warning, error) { return v.validateAuthConfig(&c.Auth) },
		func() (Warning, error) { return nil, validateLogConfig(&c.Log) },
		func() (Warning, error) { return nil, validateWebServerConfig(&c.WebServer) },
		func() (Warning, error) { return validateTransportConfig(&c.Transport) },
		func() (Warning, error) { return validateIncludeFiles(c.IncludeConfigFiles) },
	}

	// 依次执行所有验证器，收集警告和错误信息
	for _, validator := range validators {
		w, err := validator()
		warnings = AppendError(warnings, w)
		errs = AppendError(errs, err)
	}
	return warnings, errs
}

// validateFeatureGates 验证特性门控（feature gates）配置的有效性
// 参数 c 为客户端通用配置对象
// 返回验证过程中产生的警告和错误信息
func validateFeatureGates(c *v1.ClientCommonConfig) (Warning, error) {
	// 检查虚拟网络地址是否配置
	if c.VirtualNet.Address != "" {
		// 验证虚拟网络特性是否已启用
		if !featuregate.Enabled(featuregate.VirtualNet) {
			return nil, fmt.Errorf("VirtualNet 功能未启用；请通过设置相应的特性门控标志来启用它")
		}
	}
	return nil, nil
}

// validateAuthConfig 验证认证配置的有效性
// 参数 c 为客户端认证配置对象
// 返回验证过程中产生的警告和错误信息
func (v *ConfigValidator) validateAuthConfig(c *v1.AuthClientConfig) (Warning, error) {
	var errs error
	// 验证认证方法是否在支持的范围内
	if !slices.Contains(SupportedAuthMethods, c.Method) {
		errs = AppendError(errs, fmt.Errorf("无效的认证方法，可选值为 %v", SupportedAuthMethods))
	}
	// 验证额外的认证作用域是否在支持的范围内
	if !lo.Every(SupportedAuthAdditionalScopes, c.AdditionalScopes) {
		errs = AppendError(errs, fmt.Errorf("无效的认证额外作用域，可选值为 %v", SupportedAuthAdditionalScopes))
	}

	// 验证 token 和 tokenSource 的互斥性
	if c.Token != "" && c.TokenSource != nil {
		errs = AppendError(errs, fmt.Errorf("不能同时指定 auth.token 和 auth.tokenSource"))
	}

	// 如果指定了 tokenSource，则验证其有效性
	if c.TokenSource != nil {
		// 检查是否为 exec 类型的 tokenSource
		if c.TokenSource.Type == "exec" {
			if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
				errs = AppendError(errs, err)
			}
		}
		// 验证 tokenSource 配置
		if err := c.TokenSource.Validate(); err != nil {
			errs = AppendError(errs, fmt.Errorf("invalid auth.tokenSource: %v", err))
		}
	}

	// 验证 OIDC 配置
	if err := v.validateOIDCConfig(&c.OIDC); err != nil {
		errs = AppendError(errs, err)
	}
	return nil, errs
}

// validateOIDCConfig 验证 OIDC（OpenID Connect）认证配置的有效性
// 参数 c 为客户端 OIDC 认证配置对象
// 返回验证过程中产生的错误信息
func (v *ConfigValidator) validateOIDCConfig(c *v1.AuthOIDCClientConfig) error {
	// 如果未配置 tokenSource，则无需验证
	if c.TokenSource == nil {
		return nil
	}
	var errs error
	// 验证 oidc.tokenSource 与 oidc 其他字段的互斥性
	if c.ClientID != "" || c.ClientSecret != "" || c.Audience != "" ||
		c.Scope != "" || c.TokenEndpointURL != "" || len(c.AdditionalEndpointParams) > 0 ||
		c.TrustedCaFile != "" || c.InsecureSkipVerify || c.ProxyURL != "" {
		errs = AppendError(errs, fmt.Errorf("不能同时指定 auth.oidc.tokenSource 和 auth.oidc 的任何其他字段"))
	}
	// 检查是否为 exec 类型的 tokenSource
	if c.TokenSource.Type == "exec" {
		if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
			errs = AppendError(errs, err)
		}
	}
	// 验证 tokenSource 配置
	if err := c.TokenSource.Validate(); err != nil {
		errs = AppendError(errs, fmt.Errorf("invalid auth.oidc.tokenSource: %v", err))
	}
	return errs
}

// validateTransportConfig 验证传输层配置的有效性
// 参数 c 为客户端传输层配置对象
// 返回验证过程中产生的警告和错误信息
func validateTransportConfig(c *v1.ClientTransportConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)

	// 验证心跳超时和心跳间隔的合理性
	if c.HeartbeatTimeout > 0 && c.HeartbeatInterval > 0 {
		if c.HeartbeatTimeout < c.HeartbeatInterval {
			errs = AppendError(errs, fmt.Errorf("invalid transport.heartbeatTimeout, heartbeat timeout should not less than heartbeat interval"))
		}
	}

	// 当 TLS 未启用时，检查 TLS 相关配置是否被错误设置
	if !lo.FromPtr(c.TLS.Enable) {
		// 定义检查 TLS 配置的函数
		checkTLSConfig := func(name string, value string) Warning {
			if value != "" {
				return fmt.Errorf("当 transport.tls.enable 为 false 时，%s 无效", name)
			}
			return nil
		}

		// 检查各个 TLS 配置项
		warnings = AppendError(warnings, checkTLSConfig("transport.tls.certFile", c.TLS.CertFile))
		warnings = AppendError(warnings, checkTLSConfig("transport.tls.keyFile", c.TLS.KeyFile))
		warnings = AppendError(warnings, checkTLSConfig("transport.tls.trustedCaFile", c.TLS.TrustedCaFile))
	}

	// 验证传输协议是否在支持的范围内
	if !slices.Contains(SupportedTransportProtocols, c.Protocol) {
		errs = AppendError(errs, fmt.Errorf("无效的 transport.protocol，可选值为 %v", SupportedTransportProtocols))
	}
	return warnings, errs
}

// validateIncludeFiles 验证包含的配置文件路径的有效性
// 参数 files 为配置文件路径列表
// 返回验证过程中产生的错误信息
func validateIncludeFiles(files []string) (Warning, error) {
	var errs error
	// 遍历所有包含的文件路径
	for _, f := range files {
		// 获取文件的绝对路径
		absDir, err := filepath.Abs(filepath.Dir(f))
		if err != nil {
			errs = AppendError(errs, fmt.Errorf("include: 解析 %s 的目录失败: %v", f, err))
			continue
		}
		// 检查目录是否存在
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			errs = AppendError(errs, fmt.Errorf("include: %s 的目录不存在", f))
		}
	}
	return nil, errs
}

// ValidateAllClientConfig 验证所有客户端配置的有效性
// 参数 c 为客户端通用配置对象
// 参数 proxyCfgs 为代理配置器列表
// 参数 visitorCfgs 为访问者配置器列表
// 参数 unsafeFeatures 为不安全特性配置对象
// 返回验证过程中产生的警告和错误信息
func ValidateAllClientConfig(
	c *v1.ClientCommonConfig,
	proxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	unsafeFeatures *security.UnsafeFeatures,
) (Warning, error) {
	// 创建配置验证器
	validator := NewConfigValidator(unsafeFeatures)
	var warnings Warning
	// 验证客户端通用配置
	if c != nil {
		warning, err := validator.ValidateClientCommonConfig(c)
		warnings = AppendError(warnings, warning)
		if err != nil {
			return warnings, err
		}
	}

	// 验证所有代理配置
	for _, c := range proxyCfgs {
		if err := ValidateProxyConfigurerForClient(c); err != nil {
			return warnings, fmt.Errorf("proxy %s: %v", c.GetBaseConfig().Name, err)
		}
	}

	// 验证所有访问者配置
	for _, c := range visitorCfgs {
		if err := ValidateVisitorConfigurer(c); err != nil {
			return warnings, fmt.Errorf("visitor %s: %v", c.GetBaseConfig().Name, err)
		}
	}
	return warnings, nil
}
