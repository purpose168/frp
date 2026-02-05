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
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	v1 "github.com/purpose168/frp/pkg/config/v1"
)

// validateProxyBaseConfigForClient 验证客户端代理基础配置的有效性
// 参数 c 为代理基础配置对象
// 返回验证过程中产生的错误信息
func validateProxyBaseConfigForClient(c *v1.ProxyBaseConfig) error {
	// 验证代理名称是否为空
	if c.Name == "" {
		return errors.New("名称不应为空")
	}

	// 验证注解的有效性
	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	// 验证代理协议版本是否支持
	if !slices.Contains([]string{"", "v1", "v2"}, c.Transport.ProxyProtocolVersion) {
		return fmt.Errorf("不支持的代理协议版本: %s", c.Transport.ProxyProtocolVersion)
	}
	// 验证带宽限制模式是否为 client 或 server
	if !slices.Contains([]string{"client", "server"}, c.Transport.BandwidthLimitMode) {
		return fmt.Errorf("带宽限制模式应为 client 或 server")
	}

	// 如果未配置插件，则验证本地端口
	if c.Plugin.Type == "" {
		if err := ValidatePort(c.LocalPort, "localPort"); err != nil {
			return fmt.Errorf("localPort: %v", err)
		}
	}

	// 验证健康检查类型是否支持
	if !slices.Contains([]string{"", "tcp", "http"}, c.HealthCheck.Type) {
		return fmt.Errorf("不支持的健康检查类型: %s", c.HealthCheck.Type)
	}
	// 如果配置了健康检查，验证相关参数
	if c.HealthCheck.Type != "" {
		if c.HealthCheck.Type == "http" &&
			c.HealthCheck.Path == "" {
			return fmt.Errorf("健康检查路径不应为空")
		}
	}

	// 如果配置了插件，验证插件选项
	if c.Plugin.Type != "" {
		if err := ValidateClientPluginOptions(c.Plugin.ClientPluginOptions); err != nil {
			return fmt.Errorf("插件 %s: %v", c.Plugin.Type, err)
		}
	}
	return nil
}

// validateProxyBaseConfigForServer 验证服务端代理基础配置的有效性
// 参数 c 为代理基础配置对象
// 返回验证过程中产生的错误信息
func validateProxyBaseConfigForServer(c *v1.ProxyBaseConfig) error {
	// 验证注解的有效性
	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	return nil
}

// validateDomainConfigForClient 验证客户端域名配置的有效性
// 参数 c 为域名配置对象
// 返回验证过程中产生的错误信息
func validateDomainConfigForClient(c *v1.DomainConfig) error {
	// 验证子域名和自定义域名是否都为空
	if c.SubDomain == "" && len(c.CustomDomains) == 0 {
		return errors.New("子域名和自定义域名不应都为空")
	}
	return nil
}

// validateDomainConfigForServer 验证服务端域名配置的有效性
// 参数 c 为域名配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateDomainConfigForServer(c *v1.DomainConfig, s *v1.ServerConfig) error {
	// 验证自定义域名是否属于子域名主机
	for _, domain := range c.CustomDomains {
		if s.SubDomainHost != "" && len(strings.Split(s.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, s.SubDomainHost) {
				return fmt.Errorf("自定义域名 [%s] 不应属于子域名主机 [%s]", domain, s.SubDomainHost)
			}
		}
	}

	// 验证子域名配置
	if c.SubDomain != "" {
		// 检查服务端是否启用子域名功能
		if s.SubDomainHost == "" {
			return errors.New("不支持子域名，因为服务端未启用此功能")
		}

		// 检查子域名中是否包含非法字符
		if strings.Contains(c.SubDomain, ".") || strings.Contains(c.SubDomain, "*") {
			return errors.New("子域名中不支持 '.' 和 '*'")
		}
	}
	return nil
}

// ValidateProxyConfigurerForClient 验证客户端代理配置器的有效性
// 参数 c 为代理配置器对象
// 返回验证过程中产生的错误信息
func ValidateProxyConfigurerForClient(c v1.ProxyConfigurer) error {
	base := c.GetBaseConfig()
	// 验证基础配置
	if err := validateProxyBaseConfigForClient(base); err != nil {
		return err
	}

	// 根据代理类型进行相应的验证
	switch v := c.(type) {
	case *v1.TCPProxyConfig:
		return validateTCPProxyConfigForClient(v)
	case *v1.UDPProxyConfig:
		return validateUDPProxyConfigForClient(v)
	case *v1.TCPMuxProxyConfig:
		return validateTCPMuxProxyConfigForClient(v)
	case *v1.HTTPProxyConfig:
		return validateHTTPProxyConfigForClient(v)
	case *v1.HTTPSProxyConfig:
		return validateHTTPSProxyConfigForClient(v)
	case *v1.STCPProxyConfig:
		return validateSTCPProxyConfigForClient(v)
	case *v1.XTCPProxyConfig:
		return validateXTCPProxyConfigForClient(v)
	case *v1.SUDPProxyConfig:
		return validateSUDPProxyConfigForClient(v)
	}
	return errors.New("未知的代理配置类型")
}

// validateTCPProxyConfigForClient 验证客户端 TCP 代理配置的有效性
// 参数 c 为 TCP 代理配置对象
// 返回验证过程中产生的错误信息
func validateTCPProxyConfigForClient(c *v1.TCPProxyConfig) error {
	return nil
}

// validateUDPProxyConfigForClient 验证客户端 UDP 代理配置的有效性
// 参数 c 为 UDP 代理配置对象
// 返回验证过程中产生的错误信息
func validateUDPProxyConfigForClient(c *v1.UDPProxyConfig) error {
	return nil
}

// validateTCPMuxProxyConfigForClient 验证客户端 TCP 多路复用代理配置的有效性
// 参数 c 为 TCP 多路复用代理配置对象
// 返回验证过程中产生的错误信息
func validateTCPMuxProxyConfigForClient(c *v1.TCPMuxProxyConfig) error {
	// 验证域名配置
	if err := validateDomainConfigForClient(&c.DomainConfig); err != nil {
		return err
	}

	// 验证多路复用器类型
	if !slices.Contains([]string{string(v1.TCPMultiplexerHTTPConnect)}, c.Multiplexer) {
		return fmt.Errorf("不支持的多路复用器: %s", c.Multiplexer)
	}
	return nil
}

// validateHTTPProxyConfigForClient 验证客户端 HTTP 代理配置的有效性
// 参数 c 为 HTTP 代理配置对象
// 返回验证过程中产生的错误信息
func validateHTTPProxyConfigForClient(c *v1.HTTPProxyConfig) error {
	return validateDomainConfigForClient(&c.DomainConfig)
}

// validateHTTPSProxyConfigForClient 验证客户端 HTTPS 代理配置的有效性
// 参数 c 为 HTTPS 代理配置对象
// 返回验证过程中产生的错误信息
func validateHTTPSProxyConfigForClient(c *v1.HTTPSProxyConfig) error {
	return validateDomainConfigForClient(&c.DomainConfig)
}

// validateSTCPProxyConfigForClient 验证客户端 STCP 代理配置的有效性
// 参数 c 为 STCP 代理配置对象
// 返回验证过程中产生的错误信息
func validateSTCPProxyConfigForClient(c *v1.STCPProxyConfig) error {
	return nil
}

// validateXTCPProxyConfigForClient 验证客户端 XTCP 代理配置的有效性
// 参数 c 为 XTCP 代理配置对象
// 返回验证过程中产生的错误信息
func validateXTCPProxyConfigForClient(c *v1.XTCPProxyConfig) error {
	return nil
}

// validateSUDPProxyConfigForClient 验证客户端 SUDP 代理配置的有效性
// 参数 c 为 SUDP 代理配置对象
// 返回验证过程中产生的错误信息
func validateSUDPProxyConfigForClient(c *v1.SUDPProxyConfig) error {
	return nil
}

// ValidateProxyConfigurerForServer 验证服务端代理配置器的有效性
// 参数 c 为代理配置器对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func ValidateProxyConfigurerForServer(c v1.ProxyConfigurer, s *v1.ServerConfig) error {
	base := c.GetBaseConfig()
	// 验证基础配置
	if err := validateProxyBaseConfigForServer(base); err != nil {
		return err
	}

	// 根据代理类型进行相应的验证
	switch v := c.(type) {
	case *v1.TCPProxyConfig:
		return validateTCPProxyConfigForServer(v, s)
	case *v1.UDPProxyConfig:
		return validateUDPProxyConfigForServer(v, s)
	case *v1.TCPMuxProxyConfig:
		return validateTCPMuxProxyConfigForServer(v, s)
	case *v1.HTTPProxyConfig:
		return validateHTTPProxyConfigForServer(v, s)
	case *v1.HTTPSProxyConfig:
		return validateHTTPSProxyConfigForServer(v, s)
	case *v1.STCPProxyConfig:
		return validateSTCPProxyConfigForServer(v, s)
	case *v1.XTCPProxyConfig:
		return validateXTCPProxyConfigForServer(v, s)
	case *v1.SUDPProxyConfig:
		return validateSUDPProxyConfigForServer(v, s)
	default:
		return errors.New("未知的代理配置类型")
	}
}

// validateTCPProxyConfigForServer 验证服务端 TCP 代理配置的有效性
// 参数 c 为 TCP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateTCPProxyConfigForServer(c *v1.TCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// validateUDPProxyConfigForServer 验证服务端 UDP 代理配置的有效性
// 参数 c 为 UDP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateUDPProxyConfigForServer(c *v1.UDPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// validateTCPMuxProxyConfigForServer 验证服务端 TCP 多路复用代理配置的有效性
// 参数 c 为 TCP 多路复用代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateTCPMuxProxyConfigForServer(c *v1.TCPMuxProxyConfig, s *v1.ServerConfig) error {
	// 检查服务端是否启用 TCP 多路复用 HTTP 连接功能
	if c.Multiplexer == string(v1.TCPMultiplexerHTTPConnect) &&
		s.TCPMuxHTTPConnectPort == 0 {
		return fmt.Errorf("不支持使用 httpconnect 多路复用器的 tcpmux，因为服务端未启用此功能")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

// validateHTTPProxyConfigForServer 验证服务端 HTTP 代理配置的有效性
// 参数 c 为 HTTP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateHTTPProxyConfigForServer(c *v1.HTTPProxyConfig, s *v1.ServerConfig) error {
	// 检查服务端是否配置了虚拟主机 HTTP 端口
	if s.VhostHTTPPort == 0 {
		return fmt.Errorf("未设置虚拟主机 HTTP 端口时不支持类型 [http]")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

// validateHTTPSProxyConfigForServer 验证服务端 HTTPS 代理配置的有效性
// 参数 c 为 HTTPS 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateHTTPSProxyConfigForServer(c *v1.HTTPSProxyConfig, s *v1.ServerConfig) error {
	// 检查服务端是否配置了虚拟主机 HTTPS 端口
	if s.VhostHTTPSPort == 0 {
		return fmt.Errorf("未设置虚拟主机 HTTPS 端口时不支持类型 [https]")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

// validateSTCPProxyConfigForServer 验证服务端 STCP 代理配置的有效性
// 参数 c 为 STCP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateSTCPProxyConfigForServer(c *v1.STCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// validateXTCPProxyConfigForServer 验证服务端 XTCP 代理配置的有效性
// 参数 c 为 XTCP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateXTCPProxyConfigForServer(c *v1.XTCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// validateSUDPProxyConfigForServer 验证服务端 SUDP 代理配置的有效性
// 参数 c 为 SUDP 代理配置对象
// 参数 s 为服务端配置对象
// 返回验证过程中产生的错误信息
func validateSUDPProxyConfigForServer(c *v1.SUDPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// ValidateAnnotations 验证注解集合是否正确定义
// 参数 annotations 为注解映射
// 返回验证过程中产生的错误信息
func ValidateAnnotations(annotations map[string]string) error {
	// 如果没有注解，直接返回
	if len(annotations) == 0 {
		return nil
	}

	var errs error
	// 验证每个注解键是否为合格名称
	for k := range annotations {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(k)) {
			errs = AppendError(errs, fmt.Errorf("注解键 %s 无效: %s", k, msg))
		}
	}
	// 验证注解大小
	if err := ValidateAnnotationsSize(annotations); err != nil {
		errs = AppendError(errs, err)
	}
	return errs
}

// TotalAnnotationSizeLimitB 注解总大小限制（字节）
const TotalAnnotationSizeLimitB int = 256 * (1 << 10) // 256 kB

// ValidateAnnotationsSize 验证注解总大小是否超过限制
// 参数 annotations 为注解映射
// 返回验证过程中产生的错误信息
func ValidateAnnotationsSize(annotations map[string]string) error {
	var totalSize int64
	// 计算所有注解键和值的总大小
	for k, v := range annotations {
		totalSize += (int64)(len(k)) + (int64)(len(v))
	}
	// 检查是否超过限制
	if totalSize > (int64)(TotalAnnotationSizeLimitB) {
		return fmt.Errorf("注解大小 %d 超过限制 %d", totalSize, TotalAnnotationSizeLimitB)
	}
	return nil
}
