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

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// ValidateClientPluginOptions 验证客户端插件选项的有效性
// 参数 c 为客户端插件选项对象
// 返回验证过程中产生的错误信息
func ValidateClientPluginOptions(c v1.ClientPluginOptions) error {
	// 根据插件类型进行相应的验证
	switch v := c.(type) {
	case *v1.HTTP2HTTPSPluginOptions:
		return validateHTTP2HTTPSPluginOptions(v)
	case *v1.HTTPS2HTTPPluginOptions:
		return validateHTTPS2HTTPPluginOptions(v)
	case *v1.HTTPS2HTTPSPluginOptions:
		return validateHTTPS2HTTPSPluginOptions(v)
	case *v1.StaticFilePluginOptions:
		return validateStaticFilePluginOptions(v)
	case *v1.UnixDomainSocketPluginOptions:
		return validateUnixDomainSocketPluginOptions(v)
	case *v1.TLS2RawPluginOptions:
		return validateTLS2RawPluginOptions(v)
	}
	return nil
}

// validateHTTP2HTTPSPluginOptions 验证 HTTP 到 HTTPS 插件选项的有效性
// 参数 c 为 HTTP2HTTPS 插件选项对象
// 返回验证过程中产生的错误信息
func validateHTTP2HTTPSPluginOptions(c *v1.HTTP2HTTPSPluginOptions) error {
	// 验证本地地址是否配置
	if c.LocalAddr == "" {
		return errors.New("localAddr 是必需的")
	}
	return nil
}

// validateHTTPS2HTTPPluginOptions 验证 HTTPS 到 HTTP 插件选项的有效性
// 参数 c 为 HTTPS2HTTP 插件选项对象
// 返回验证过程中产生的错误信息
func validateHTTPS2HTTPPluginOptions(c *v1.HTTPS2HTTPPluginOptions) error {
	// 验证本地地址是否配置
	if c.LocalAddr == "" {
		return errors.New("localAddr 是必需的")
	}
	return nil
}

// validateHTTPS2HTTPSPluginOptions 验证 HTTPS 到 HTTPS 插件选项的有效性
// 参数 c 为 HTTPS2HTTPS 插件选项对象
// 返回验证过程中产生的错误信息
func validateHTTPS2HTTPSPluginOptions(c *v1.HTTPS2HTTPSPluginOptions) error {
	// 验证本地地址是否配置
	if c.LocalAddr == "" {
		return errors.New("localAddr 是必需的")
	}
	return nil
}

// validateStaticFilePluginOptions 验证静态文件插件选项的有效性
// 参数 c 为静态文件插件选项对象
// 返回验证过程中产生的错误信息
func validateStaticFilePluginOptions(c *v1.StaticFilePluginOptions) error {
	// 验证本地路径是否配置
	if c.LocalPath == "" {
		return errors.New("localPath 是必需的")
	}
	return nil
}

// validateUnixDomainSocketPluginOptions 验证 Unix 域套接字插件选项的有效性
// 参数 c 为 Unix 域套接字插件选项对象
// 返回验证过程中产生的错误信息
func validateUnixDomainSocketPluginOptions(c *v1.UnixDomainSocketPluginOptions) error {
	// 验证 Unix 路径是否配置
	if c.UnixPath == "" {
		return errors.New("unixPath 是必需的")
	}
	return nil
}

// validateTLS2RawPluginOptions 验证 TLS 到原始数据插件选项的有效性
// 参数 c 为 TLS2Raw 插件选项对象
// 返回验证过程中产生的错误信息
func validateTLS2RawPluginOptions(c *v1.TLS2RawPluginOptions) error {
	// 验证本地地址是否配置
	if c.LocalAddr == "" {
		return errors.New("localAddr 是必需的")
	}
	return nil
}
