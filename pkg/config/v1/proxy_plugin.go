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

const (
	// PluginHTTP2HTTPS HTTP 到 HTTPS 插件
	PluginHTTP2HTTPS = "http2https"
	// PluginHTTPProxy HTTP 代理插件
	PluginHTTPProxy = "http_proxy"
	// PluginHTTPS2HTTP HTTPS 到 HTTP 插件
	PluginHTTPS2HTTP = "https2http"
	// PluginHTTPS2HTTPS HTTPS 到 HTTPS 插件
	PluginHTTPS2HTTPS = "https2https"
	// PluginHTTP2HTTP HTTP 到 HTTP 插件
	PluginHTTP2HTTP = "http2http"
	// PluginSocks5 Socks5 插件
	PluginSocks5 = "socks5"
	// PluginStaticFile 静态文件插件
	PluginStaticFile = "static_file"
	// PluginUnixDomainSocket Unix 域套接字插件
	PluginUnixDomainSocket = "unix_domain_socket"
	// PluginTLS2Raw TLS 到原始数据插件
	PluginTLS2Raw = "tls2raw"
	// PluginVirtualNet 虚拟网络插件
	PluginVirtualNet = "virtual_net"
)

// clientPluginOptionsTypeMap 客户端插件选项类型映射
var clientPluginOptionsTypeMap = map[string]reflect.Type{
	PluginHTTP2HTTPS:       reflect.TypeOf(HTTP2HTTPSPluginOptions{}),
	PluginHTTPProxy:        reflect.TypeOf(HTTPProxyPluginOptions{}),
	PluginHTTPS2HTTP:       reflect.TypeOf(HTTPS2HTTPPluginOptions{}),
	PluginHTTPS2HTTPS:      reflect.TypeOf(HTTPS2HTTPSPluginOptions{}),
	PluginHTTP2HTTP:        reflect.TypeOf(HTTP2HTTPPluginOptions{}),
	PluginSocks5:           reflect.TypeOf(Socks5PluginOptions{}),
	PluginStaticFile:       reflect.TypeOf(StaticFilePluginOptions{}),
	PluginUnixDomainSocket: reflect.TypeOf(UnixDomainSocketPluginOptions{}),
	PluginTLS2Raw:          reflect.TypeOf(TLS2RawPluginOptions{}),
	PluginVirtualNet:       reflect.TypeOf(VirtualNetPluginOptions{}),
}

// ClientPluginOptions 客户端插件选项接口
type ClientPluginOptions interface {
	Complete()
}

// TypedClientPluginOptions 类型化客户端插件选项结构体
type TypedClientPluginOptions struct {
	// Type 插件类型
	Type string `json:"type"`
	ClientPluginOptions
}

// UnmarshalJSON 自定义 JSON 反序列化方法
func (c *TypedClientPluginOptions) UnmarshalJSON(b []byte) error {
	// 处理 null 值
	if len(b) == 4 && string(b) == "null" {
		return nil
	}

	// 解析插件类型
	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	if c.Type == "" {
		return errors.New("插件类型为空")
	}

	// 根据类型获取对应的选项结构
	v, ok := clientPluginOptionsTypeMap[typeStruct.Type]
	if !ok {
		return fmt.Errorf("未知的插件类型: %s", typeStruct.Type)
	}
	options := reflect.New(v).Interface().(ClientPluginOptions)

	// 创建 JSON 解码器
	decoder := json.NewDecoder(bytes.NewBuffer(b))
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	// 解码插件选项
	if err := decoder.Decode(options); err != nil {
		return fmt.Errorf("反序列化 ClientPluginOptions 错误: %v", err)
	}
	c.ClientPluginOptions = options
	return nil
}

// MarshalJSON 自定义 JSON 序列化方法
func (c *TypedClientPluginOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ClientPluginOptions)
}

// HTTP2HTTPSPluginOptions HTTP 到 HTTPS 插件选项结构体
type HTTP2HTTPSPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalAddr 本地地址
	LocalAddr string `json:"localAddr,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"hostHeaderRewrite,omitempty"`
	// RequestHeaders 请求头操作
	RequestHeaders HeaderOperations `json:"requestHeaders,omitempty"`
}

// Complete 填充 HTTP 到 HTTPS 插件选项的默认值
func (o *HTTP2HTTPSPluginOptions) Complete() {}

// HTTPProxyPluginOptions HTTP 代理插件选项结构体
type HTTPProxyPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// HTTPUser HTTP 用户名
	HTTPUser string `json:"httpUser,omitempty"`
	// HTTPPassword HTTP 密码
	HTTPPassword string `json:"httpPassword,omitempty"`
}

// Complete 填充 HTTP 代理插件选项的默认值
func (o *HTTPProxyPluginOptions) Complete() {}

// HTTPS2HTTPPluginOptions HTTPS 到 HTTP 插件选项结构体
type HTTPS2HTTPPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalAddr 本地地址
	LocalAddr string `json:"localAddr,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"hostHeaderRewrite,omitempty"`
	// RequestHeaders 请求头操作
	RequestHeaders HeaderOperations `json:"requestHeaders,omitempty"`
	// EnableHTTP2 是否启用 HTTP/2
	EnableHTTP2 *bool `json:"enableHTTP2,omitempty"`
	// CrtPath 证书文件路径
	CrtPath string `json:"crtPath,omitempty"`
	// KeyPath 密钥文件路径
	KeyPath string `json:"keyPath,omitempty"`
}

// Complete 填充 HTTPS 到 HTTP 插件选项的默认值
func (o *HTTPS2HTTPPluginOptions) Complete() {
	// 设置默认启用 HTTP/2
	o.EnableHTTP2 = util.EmptyOr(o.EnableHTTP2, lo.ToPtr(true))
}

// HTTPS2HTTPSPluginOptions HTTPS 到 HTTPS 插件选项结构体
type HTTPS2HTTPSPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalAddr 本地地址
	LocalAddr string `json:"localAddr,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"hostHeaderRewrite,omitempty"`
	// RequestHeaders 请求头操作
	RequestHeaders HeaderOperations `json:"requestHeaders,omitempty"`
	// EnableHTTP2 是否启用 HTTP/2
	EnableHTTP2 *bool `json:"enableHTTP2,omitempty"`
	// CrtPath 证书文件路径
	CrtPath string `json:"crtPath,omitempty"`
	// KeyPath 密钥文件路径
	KeyPath string `json:"keyPath,omitempty"`
}

// Complete 填充 HTTPS 到 HTTPS 插件选项的默认值
func (o *HTTPS2HTTPSPluginOptions) Complete() {
	// 设置默认启用 HTTP/2
	o.EnableHTTP2 = util.EmptyOr(o.EnableHTTP2, lo.ToPtr(true))
}

// HTTP2HTTPPluginOptions HTTP 到 HTTP 插件选项结构体
type HTTP2HTTPPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalAddr 本地地址
	LocalAddr string `json:"localAddr,omitempty"`
	// HostHeaderRewrite 主机头重写
	HostHeaderRewrite string `json:"hostHeaderRewrite,omitempty"`
	// RequestHeaders 请求头操作
	RequestHeaders HeaderOperations `json:"requestHeaders,omitempty"`
}

// Complete 填充 HTTP 到 HTTP 插件选项的默认值
func (o *HTTP2HTTPPluginOptions) Complete() {}

// Socks5PluginOptions Socks5 插件选项结构体
type Socks5PluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// Username 用户名
	Username string `json:"username,omitempty"`
	// Password 密码
	Password string `json:"password,omitempty"`
}

// Complete 填充 Socks5 插件选项的默认值
func (o *Socks5PluginOptions) Complete() {}

// StaticFilePluginOptions 静态文件插件选项结构体
type StaticFilePluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalPath 本地路径
	LocalPath string `json:"localPath,omitempty"`
	// StripPrefix 要去除的前缀
	StripPrefix string `json:"stripPrefix,omitempty"`
	// HTTPUser HTTP 用户名
	HTTPUser string `json:"httpUser,omitempty"`
	// HTTPPassword HTTP 密码
	HTTPPassword string `json:"httpPassword,omitempty"`
}

// Complete 填充静态文件插件选项的默认值
func (o *StaticFilePluginOptions) Complete() {}

// UnixDomainSocketPluginOptions Unix 域套接字插件选项结构体
type UnixDomainSocketPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// UnixPath Unix 路径
	UnixPath string `json:"unixPath,omitempty"`
}

// Complete 填充 Unix 域套接字插件选项的默认值
func (o *UnixDomainSocketPluginOptions) Complete() {}

// TLS2RawPluginOptions TLS 到原始数据插件选项结构体
type TLS2RawPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
	// LocalAddr 本地地址
	LocalAddr string `json:"localAddr,omitempty"`
	// CrtPath 证书文件路径
	CrtPath string `json:"crtPath,omitempty"`
	// KeyPath 密钥文件路径
	KeyPath string `json:"keyPath,omitempty"`
}

// Complete 填充 TLS 到原始数据插件选项的默认值
func (o *TLS2RawPluginOptions) Complete() {}

// VirtualNetPluginOptions 虚拟网络插件选项结构体
type VirtualNetPluginOptions struct {
	// Type 插件类型
	Type string `json:"type,omitempty"`
}

// Complete 填充虚拟网络插件选项的默认值
func (o *VirtualNetPluginOptions) Complete() {}
