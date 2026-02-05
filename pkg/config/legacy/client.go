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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/ini.v1"

	legacyauth "github.com/fatedier/frp/pkg/auth/legacy"
	"github.com/fatedier/frp/pkg/util/util"
)

// ClientCommonConf 是从 ini 解析的配置。
// 它包含客户端服务的信息。建议使用 GetDefaultClientConf 而不是直接创建此对象，
// 这样所有未指定的字段都具有合理的默认值。
type ClientCommonConf struct {
	legacyauth.ClientConfig `ini:",extends"`

	// ServerAddr 指定要连接的服务器地址。默认情况下，此值为 "0.0.0.0"。
	ServerAddr string `ini:"server_addr" json:"server_addr"`
	// ServerPort 指定要连接服务器的端口。默认情况下，此值为 7000。
	ServerPort int `ini:"server_port" json:"server_port"`
	// NatHoleSTUNServer 指定用于帮助穿透 NAT 孔的 STUN 服务器。
	NatHoleSTUNServer string `ini:"nat_hole_stun_server" json:"nat_hole_stun_server"`
	// DialServerTimeout 指定连接服务器时等待连接完成的最大时间。
	DialServerTimeout int64 `ini:"dial_server_timeout" json:"dial_server_timeout"`
	// DialServerKeepAlive 指定 frpc 和 frps 之间活动网络连接的保活探测间隔。
	// 如果为负数，则禁用保活探测。
	DialServerKeepAlive int64 `ini:"dial_server_keepalive" json:"dial_server_keepalive"`
	// ConnectServerLocalIP 指定客户端连接服务器时绑定的地址。
	// 默认情况下，此值为空。
	// 此值仅用于 TCP/Websocket 协议。不支持 KCP 协议。
	ConnectServerLocalIP string `ini:"connect_server_local_ip" json:"connect_server_local_ip"`
	// HTTPProxy 指定用于连接服务器的代理地址。如果此值为 ""，则直接连接服务器。
	// 默认情况下，此值从 "http_proxy" 环境变量中读取。
	HTTPProxy string `ini:"http_proxy" json:"http_proxy"`
	// LogFile 指定日志写入的文件。只有当 LogWay 设置为适当值时才会使用此值。
	// 默认情况下，此值为 "console"。
	LogFile string `ini:"log_file" json:"log_file"`
	// LogWay 指定日志管理方式。有效值为 "console" 或 "file"。
	// 如果使用 "console"，日志将打印到 stdout。如果使用 "file"，日志将打印到 LogFile。
	// 默认情况下，此值为 "console"。
	LogWay string `ini:"log_way" json:"log_way"`
	// LogLevel 指定最小日志级别。有效值为 "trace"、"debug"、"info"、"warn" 和 "error"。
	// 默认情况下，此值为 "info"。
	LogLevel string `ini:"log_level" json:"log_level"`
	// LogMaxDays 指定删除前存储日志信息的最大天数。
	// 仅当 LogWay == "file" 时使用。默认情况下，此值为 0。
	LogMaxDays int64 `ini:"log_max_days" json:"log_max_days"`
	// DisableLogColor 当设置为 true 时，在 LogWay == "console" 时禁用日志颜色。
	// 默认情况下，此值为 false。
	DisableLogColor bool `ini:"disable_log_color" json:"disable_log_color"`
	// AdminAddr 指定管理服务器绑定的地址。默认情况下，此值为 "127.0.0.1"。
	AdminAddr string `ini:"admin_addr" json:"admin_addr"`
	// AdminPort 指定管理服务器监听的端口。如果此值为 0，则不会启动管理服务器。
	// 默认情况下，此值为 0。
	AdminPort int `ini:"admin_port" json:"admin_port"`
	// AdminUser 指定管理服务器用于登录的用户名。
	AdminUser string `ini:"admin_user" json:"admin_user"`
	// AdminPwd 指定管理服务器用于登录的密码。
	AdminPwd string `ini:"admin_pwd" json:"admin_pwd"`
	// AssetsDir 指定管理服务器从中加载资源的本地目录。
	// 如果此值为 ""，则使用 statik 从捆绑的可执行文件中加载资源。
	// 默认情况下，此值为 ""。
	AssetsDir string `ini:"assets_dir" json:"assets_dir"`
	// PoolCount 指定客户端将提前与服务器建立的连接数。默认情况下，此值为 0。
	PoolCount int `ini:"pool_count" json:"pool_count"`
	// TCPMux 切换 TCP 流多路复用。这允许来自客户端的多个请求共享单个 TCP 连接。
	// 如果此值为 true，则服务器也必须启用 TCP 多路复用。
	// 默认情况下，此值为 true。
	TCPMux bool `ini:"tcp_mux" json:"tcp_mux"`
	// TCPMuxKeepaliveInterval 指定 TCP 流多路复用的保活间隔。
	// 如果 TCPMux 为 true，则不需要应用层心跳，因为它只能依赖 TCPMUX 中的心跳。
	TCPMuxKeepaliveInterval int64 `ini:"tcp_mux_keepalive_interval" json:"tcp_mux_keepalive_interval"`
	// User 指定代理名称的前缀，以将其与其他客户端区分开来。
	// 如果此值不为 ""，代理名称将自动更改为 "{user}.{proxy_name}"。
	// 默认情况下，此值为 ""。
	User string `ini:"user" json:"user"`
	// DNSServer 指定 FRPC 使用的 DNS 服务器地址。如果此值为 ""，则使用默认 DNS。
	// 默认情况下，此值为 ""。
	DNSServer string `ini:"dns_server" json:"dns_server"`
	// LoginFailExit 控制客户端在登录失败后是否应退出。
	// 如果为 false，客户端将重试直到登录尝试成功。
	// 默认情况下，此值为 true。
	LoginFailExit bool `ini:"login_fail_exit" json:"login_fail_exit"`
	// Start 指定按名称启用的一组代理。如果此集合为空，则启用所有提供的代理。
	// 默认情况下，此值为空集合。
	Start []string `ini:"start" json:"start"`
	// Start map[string]struct{} `json:"start"`
	// Protocol 指定与服务器交互时使用的协议。
	// 有效值为 "tcp"、"kcp"、"quic"、"websocket" 和 "wss"。
	// 默认情况下，此值为 "tcp"。
	Protocol string `ini:"protocol" json:"protocol"`
	// QUIC 协议选项
	QUICKeepalivePeriod    int `ini:"quic_keepalive_period" json:"quic_keepalive_period"`
	QUICMaxIdleTimeout     int `ini:"quic_max_idle_timeout" json:"quic_max_idle_timeout"`
	QUICMaxIncomingStreams int `ini:"quic_max_incoming_streams" json:"quic_max_incoming_streams"`
	// TLSEnable 指定与服务器通信时是否应使用 TLS。
	// 如果 "tls_cert_file" 和 "tls_key_file" 有效，客户端将加载提供的 TLS 配置。
	// 自 v0.50.0 起，默认值已更改为 true，默认启用 TLS。
	TLSEnable bool `ini:"tls_enable" json:"tls_enable"`
	// TLSCertPath 指定客户端将加载的证书文件路径。
	// 仅当 "tls_enable" 为 true 且 "tls_key_file" 有效时才起作用。
	TLSCertFile string `ini:"tls_cert_file" json:"tls_cert_file"`
	// TLSKeyPath 指定客户端将加载的密钥文件路径。
	// 仅当 "tls_enable" 为 true 且 "tls_cert_file" 有效时才起作用。
	TLSKeyFile string `ini:"tls_key_file" json:"tls_key_file"`
	// TLSTrustedCaFile 指定将加载的受信任 CA 文件路径。
	// 仅当 "tls_enable" 有效且已指定服务器的 TLS 配置时才起作用。
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file" json:"tls_trusted_ca_file"`
	// TLSServerName 指定 TLS 证书的自定义服务器名称。
	// 默认情况下，服务器名称与 ServerAddr 相同。
	TLSServerName string `ini:"tls_server_name" json:"tls_server_name"`
	// 如果 disable_custom_tls_first_byte 设置为 false，当启用 TLS 时，frpc 将使用第一个自定义字节与 frps 建立连接。
	// 自 v0.50.0 起，默认值已更改为 true，默认禁用第一个自定义字节。
	DisableCustomTLSFirstByte bool `ini:"disable_custom_tls_first_byte" json:"disable_custom_tls_first_byte"`
	// HeartBeatInterval 指定向服务器发送心跳的间隔（以秒为单位）。
	// 不建议更改此值。默认情况下，此值为 30。设置为负值以禁用它。
	HeartbeatInterval int64 `ini:"heartbeat_interval" json:"heartbeat_interval"`
	// HeartBeatTimeout 指定在连接终止之前允许的最大心跳响应延迟（以秒为单位）。
	// 不建议更改此值。默认情况下，此值为 90。设置为负值以禁用它。
	HeartbeatTimeout int64 `ini:"heartbeat_timeout" json:"heartbeat_timeout"`
	// 客户端元信息
	Metas map[string]string `ini:"-" json:"metas"`
	// UDPPacketSize 指定 UDP 数据包大小
	// 默认情况下，此值为 1500
	UDPPacketSize int64 `ini:"udp_packet_size" json:"udp_packet_size"`
	// IncludeConfigFiles 包含代理的其他配置文件。
	IncludeConfigFiles []string `ini:"includes" json:"includes"`
	// 在管理监听器中启用 golang pprof 处理程序。
	// 必须先设置管理端口。
	PprofEnable bool `ini:"pprof_enable" json:"pprof_enable"`
}

// 支持的源包括：字符串（文件路径）、[]byte、Reader 接口。
func UnmarshalClientConfFromIni(source any) (ClientCommonConf, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ClientCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		return ClientCommonConf{}, fmt.Errorf("无效的配置文件，未找到 [common] 部分")
	}

	common := GetDefaultClientConf()
	err = s.MapTo(&common)
	if err != nil {
		return ClientCommonConf{}, err
	}

	common.Metas = GetMapWithoutPrefix(s.KeysHash(), "meta_")
	common.OidcAdditionalEndpointParams = GetMapWithoutPrefix(s.KeysHash(), "oidc_additional_")

	return common, nil
}

// 如果 len(startProxy) 为 0，则启动所有代理
// 否则只启动 startProxy 映射中的代理
func LoadAllProxyConfsFromIni(
	prefix string,
	source any,
	start []string,
) (map[string]ProxyConf, map[string]VisitorConf, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return nil, nil, err
	}

	proxyConfs := make(map[string]ProxyConf)
	visitorConfs := make(map[string]VisitorConf)

	if prefix != "" {
		prefix += "."
	}

	startProxy := make(map[string]struct{})
	for _, s := range start {
		startProxy[s] = struct{}{}
	}

	startAll := len(startProxy) == 0

	// Build template sections from range section And append to ini.File.
	rangeSections := make([]*ini.Section, 0)
	for _, section := range f.Sections() {

		if !strings.HasPrefix(section.Name(), "range:") {
			continue
		}

		rangeSections = append(rangeSections, section)
	}

	for _, section := range rangeSections {
		err = renderRangeProxyTemplates(f, section)
		if err != nil {
			return nil, nil, fmt.Errorf("渲染代理 %s 的模板失败: %v", section.Name(), err)
		}
	}

	for _, section := range f.Sections() {
		name := section.Name()

		if name == ini.DefaultSection || name == "common" || strings.HasPrefix(name, "range:") {
			continue
		}

		_, shouldStart := startProxy[name]
		if !startAll && !shouldStart {
			continue
		}

		roleType := section.Key("role").String()
		if roleType == "" {
			roleType = "server"
		}

		switch roleType {
		case "server":
			newConf, newErr := NewProxyConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, fmt.Errorf("解析代理 %s 失败，错误: %v", name, newErr)
			}
			proxyConfs[prefix+name] = newConf
		case "visitor":
			newConf, newErr := NewVisitorConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, fmt.Errorf("解析访问者 %s 失败，错误: %v", name, newErr)
			}
			visitorConfs[prefix+name] = newConf
		default:
			return nil, nil, fmt.Errorf("代理 %s 的角色应为 'server' 或 'visitor'", name)
		}
	}
	return proxyConfs, visitorConfs, nil
}

func renderRangeProxyTemplates(f *ini.File, section *ini.Section) error {
	// 验证
	localPortStr := section.Key("local_port").String()
	remotePortStr := section.Key("remote_port").String()
	if localPortStr == "" || remotePortStr == "" {
		return fmt.Errorf("local_port 或 remote_port 为空")
	}

	localPorts, err := util.ParseRangeNumbers(localPortStr)
	if err != nil {
		return err
	}

	remotePorts, err := util.ParseRangeNumbers(remotePortStr)
	if err != nil {
		return err
	}

	if len(localPorts) != len(remotePorts) {
		return fmt.Errorf("本地端口数量应与远程端口数量相同")
	}

	if len(localPorts) == 0 {
		return fmt.Errorf("local_port 和 remote_port 是必需的")
	}

	// 模板
	prefix := strings.TrimSpace(strings.TrimPrefix(section.Name(), "range:"))

	for i := range localPorts {
		tmpname := fmt.Sprintf("%s_%d", prefix, i)

		tmpsection, err := f.NewSection(tmpname)
		if err != nil {
			return err
		}

		copySection(section, tmpsection)
		if _, err := tmpsection.NewKey("local_port", fmt.Sprintf("%d", localPorts[i])); err != nil {
			return fmt.Errorf("在部分中创建 local_port 键时出错: %v", err)
		}
		if _, err := tmpsection.NewKey("remote_port", fmt.Sprintf("%d", remotePorts[i])); err != nil {
			return fmt.Errorf("在部分中创建 remote_port 键时出错: %v", err)
		}
	}

	return nil
}

func copySection(source, target *ini.Section) {
	for key, value := range source.KeysHash() {
		_, _ = target.NewKey(key, value)
	}
}

// GetDefaultClientConf 返回具有默认值的客户端配置。
// 注意：此处的一些默认值将被设置为空，并将通过 'Complete' 函数将它们转换为新配置，
// 以将它们设置为新配置的默认值。
func GetDefaultClientConf() ClientCommonConf {
	return ClientCommonConf{
		ClientConfig:              legacyauth.GetDefaultClientConf(),
		TCPMux:                    true,
		LoginFailExit:             true,
		Protocol:                  "tcp",
		Start:                     make([]string, 0),
		TLSEnable:                 true,
		DisableCustomTLSFirstByte: true,
		Metas:                     make(map[string]string),
		IncludeConfigFiles:        make([]string, 0),
	}
}

func (cfg *ClientCommonConf) Validate() error {
	if cfg.HeartbeatTimeout > 0 && cfg.HeartbeatInterval > 0 {
		if cfg.HeartbeatTimeout < cfg.HeartbeatInterval {
			return fmt.Errorf("无效的 heartbeat_timeout，heartbeat_timeout 小于 heartbeat_interval")
		}
	}

	if !cfg.TLSEnable {
		if cfg.TLSCertFile != "" {
			fmt.Println("警告！当 tls_enable 为 false 时，tls_cert_file 无效")
		}

		if cfg.TLSKeyFile != "" {
			fmt.Println("警告！当 tls_enable 为 false 时，tls_key_file 无效")
		}

		if cfg.TLSTrustedCaFile != "" {
			fmt.Println("警告！当 tls_enable 为 false 时，tls_trusted_ca_file 无效")
		}
	}

	if !slices.Contains([]string{"tcp", "kcp", "quic", "websocket", "wss"}, cfg.Protocol) {
		return fmt.Errorf("无效的协议")
	}

	for _, f := range cfg.IncludeConfigFiles {
		absDir, err := filepath.Abs(filepath.Dir(f))
		if err != nil {
			return fmt.Errorf("include: 解析 %s 的目录失败: %v", f, err)
		}
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return fmt.Errorf("include: %s 的目录不存在", f)
		}
	}
	return nil
}
