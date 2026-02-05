// Copyright 2023 The frp Authors
//
// Licensed under to Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with License.
// You may obtain a copy of License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See License for the specific language governing permissions and
// limitations under License.

package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/purpose168/frp/pkg/config/v1"
)

const tomlServerContent = `
bindAddr = "127.0.0.1"
kcpBindPort = 7000
quicBindPort = 7001
tcpmuxHTTPConnectPort = 7005
custom404Page = "/abc.html"
transport.tcpKeepalive = 10
`

const yamlServerContent = `
bindAddr: 127.0.0.1
kcpBindPort: 7000
quicBindPort: 7001
tcpmuxHTTPConnectPort: 7005
custom404Page: /abc.html
transport:
  tcpKeepalive: 10
`

const jsonServerContent = `
{
  "bindAddr": "127.0.0.1",
  "kcpBindPort": 7000,
  "quicBindPort": 7001,
  "tcpmuxHTTPConnectPort": 7005,
  "custom404Page": "/abc.html",
  "transport": {
    "tcpKeepalive": 10
  }
}
`

// TestLoadServerConfig 测试加载服务器配置
func TestLoadServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"toml", tomlServerContent},
		{"yaml", yamlServerContent},
		{"json", jsonServerContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			svrCfg := v1.ServerConfig{}
			err := LoadConfigure([]byte(test.content), &svrCfg, true)
			require.NoError(err)
			require.EqualValues("127.0.0.1", svrCfg.BindAddr)
			require.EqualValues(7000, svrCfg.KCPBindPort)
			require.EqualValues(7001, svrCfg.QUICBindPort)
			require.EqualValues(7005, svrCfg.TCPMuxHTTPConnectPort)
			require.EqualValues("/abc.html", svrCfg.Custom404Page)
			require.EqualValues(10, svrCfg.Transport.TCPKeepAlive)
		})
	}
}

// TestLoadServerConfigStrictMode 测试在严格模式下加载无效配置时失败
func TestLoadServerConfigStrictMode(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"toml", tomlServerContent},
		{"yaml", yamlServerContent},
		{"json", jsonServerContent},
	}

	for _, strict := range []bool{false, true} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s-strict-%t", test.name, strict), func(t *testing.T) {
				require := require.New(t)
				// 通过一个无意的拼写错误破坏内容
				brokenContent := strings.Replace(test.content, "bindAddr", "bindAdur", 1)
				svrCfg := v1.ServerConfig{}
				err := LoadConfigure([]byte(brokenContent), &svrCfg, strict)
				if strict {
					require.ErrorContains(err, "bindAdur")
				} else {
					require.NoError(err)
					// 由于拼写错误，BindAddr 没有被解析
					require.EqualValues("", svrCfg.BindAddr)
				}
			})
		}
	}
}

// TestRenderWithTemplate 测试使用模板渲染
func TestRenderWithTemplate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"toml", tomlServerContent, tomlServerContent},
		{"yaml", yamlServerContent, yamlServerContent},
		{"json", jsonServerContent, jsonServerContent},
		{"template numeric", `key = {{ 123 }}`, "key = 123"},
		{"template string", `key = {{ "xyz" }}`, "key = xyz"},
		{"template quote", `key = {{ printf "%q" "with space" }}`, `key = "with space"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			got, err := RenderWithTemplate([]byte(test.content), nil)
			require.NoError(err)
			require.EqualValues(test.want, string(got))
		})
	}
}

// TestCustomStructStrictMode 测试自定义结构的严格模式
func TestCustomStructStrictMode(t *testing.T) {
	require := require.New(t)

	proxyStr := `
serverPort = 7000

[[proxies]]
name = "test"
type = "tcp"
remotePort = 6000
`
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure([]byte(proxyStr), &clientCfg, true)
	require.NoError(err)

	proxyStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(proxyStr), &clientCfg, true)
	require.Error(err)

	visitorStr := `
serverPort = 7000

[[visitors]]
name = "test"
type = "stcp"
bindPort = 6000
serverName = "server"
`
	err = LoadConfigure([]byte(visitorStr), &clientCfg, true)
	require.NoError(err)

	visitorStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(visitorStr), &clientCfg, true)
	require.Error(err)

	pluginStr := `
serverPort = 7000

[[proxies]]
name = "test"
type = "tcp"
remotePort = 6000
[proxies.plugin]
type = "unix_domain_socket"
unixPath = "/tmp/uds.sock"
`
	err = LoadConfigure([]byte(pluginStr), &clientCfg, true)
	require.NoError(err)
	pluginStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(pluginStr), &clientCfg, true)
	require.Error(err)
}

// TestYAMLMergeInStrictMode 测试即使在严格模式下，YAML 合并功能也能正常工作
// 通过正确处理以点号开头的字段
func TestYAMLMergeInStrictMode(t *testing.T) {
	require := require.New(t)

	yamlContent := `
serverAddr: "127.0.0.1"
serverPort: 7000

.common: &common
  type: stcp
  secretKey: "test-secret"
  localIP: 127.0.0.1
  transport:
    useEncryption: true
    useCompression: true

proxies:
- name: ssh
  localPort: 22
  <<: *common
- name: web
  localPort: 80
  <<: *common
`

	clientCfg := v1.ClientConfig{}
	// 这应该在严格模式下工作
	err := LoadConfigure([]byte(yamlContent), &clientCfg, true)
	require.NoError(err)

	// 验证合并工作正常
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Equal(7000, clientCfg.ServerPort)
	require.Len(clientCfg.Proxies, 2)

	// 检查第一个代理
	sshProxy := clientCfg.Proxies[0].ProxyConfigurer
	require.Equal("ssh", sshProxy.GetBaseConfig().Name)
	require.Equal("stcp", sshProxy.GetBaseConfig().Type)

	// 检查第二个代理
	webProxy := clientCfg.Proxies[1].ProxyConfigurer
	require.Equal("web", webProxy.GetBaseConfig().Name)
	require.Equal("stcp", webProxy.GetBaseConfig().Type)
}

// TestOptimizedYAMLProcessing 测试 YAML 处理的优化逻辑
func TestOptimizedYAMLProcessing(t *testing.T) {
	require := require.New(t)

	yamlWithDotFields := []byte(`
serverAddr: "127.0.0.1"
.common: &common
  type: stcp
proxies:
- name: test
  <<: *common
`)

	yamlWithoutDotFields := []byte(`
serverAddr: "127.0.0.1"
proxies:
- name: test
  type: tcp
  localPort: 22
`)

	// 测试没有点字段的 YAML 在严格模式下工作
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure(yamlWithoutDotFields, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Len(clientCfg.Proxies, 1)
	require.Equal("test", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Name)

	// 测试有点字段的 YAML 仍然在严格模式下工作
	err = LoadConfigure(yamlWithDotFields, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Len(clientCfg.Proxies, 1)
	require.Equal("test", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Name)
	require.Equal("stcp", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Type)
}

// TestYAMLEdgeCases 测试 YAML 解析的边界情况，包括非映射类型
func TestYAMLEdgeCases(t *testing.T) {
	require := require.New(t)

	// 测试根级别的数组（对于 frp 配置应该失败）
	arrayYAML := []byte(`
- item1
- item2
`)
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure(arrayYAML, &clientCfg, true)
	require.Error(err) // 应该失败，因为 ClientConfig 期望一个对象

	// 测试根级别的标量（对于 frp 配置应该失败）
	scalarYAML := []byte(`"just a string"`)
	err = LoadConfigure(scalarYAML, &clientCfg, true)
	require.Error(err) // 应该失败，因为 ClientConfig 期望一个对象

	// 测试空对象（应该工作）
	emptyYAML := []byte(`{}`)
	err = LoadConfigure(emptyYAML, &clientCfg, true)
	require.NoError(err)

	// 测试没有点号的嵌套结构（应该工作）
	nestedYAML := []byte(`
serverAddr: "127.0.0.1"
serverPort: 7000
`)
	err = LoadConfigure(nestedYAML, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Equal(7000, clientCfg.ServerPort)
}
