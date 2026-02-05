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
	"strings"

	"gopkg.in/ini.v1"

	legacyauth "github.com/purpose168/frp/pkg/auth/legacy"
)

// HTTPPluginOptions 指定支持 HTTP 协议的服务器插件的配置选项。
type HTTPPluginOptions struct {
	Name      string   `ini:"name"`
	Addr      string   `ini:"addr"`
	Path      string   `ini:"path"`
	Ops       []string `ini:"ops"`
	TLSVerify bool     `ini:"tlsVerify"`
}

// ServerCommonConf 包含服务器服务的信息。建议使用 GetDefaultServerConf 而不是直接创建此对象，
// 这样所有未指定的字段都具有合理的默认值。
type ServerCommonConf struct {
	legacyauth.ServerConfig `ini:",extends"`

	// BindAddr 指定服务器绑定的地址。默认情况下，此值为 "0.0.0.0"。
	BindAddr string `ini:"bind_addr" json:"bind_addr"`
	// BindPort 指定服务器监听的端口。默认情况下，此值为 7000。
	BindPort int `ini:"bind_port" json:"bind_port"`
	// KCPBindPort 指定服务器监听的 KCP 端口。如果此值为 0，则服务器不会监听 KCP 连接。
	// 默认情况下，此值为 0。
	KCPBindPort int `ini:"kcp_bind_port" json:"kcp_bind_port"`
	// QUICBindPort 指定服务器监听的 QUIC 端口。将此值设置为 0 将禁用此功能。
	// 默认情况下，此值为 0。
	QUICBindPort int `ini:"quic_bind_port" json:"quic_bind_port"`
	// QUIC 协议选项
	QUICKeepalivePeriod    int `ini:"quic_keepalive_period" json:"quic_keepalive_period"`
	QUICMaxIdleTimeout     int `ini:"quic_max_idle_timeout" json:"quic_max_idle_timeout"`
	QUICMaxIncomingStreams int `ini:"quic_max_incoming_streams" json:"quic_max_incoming_streams"`
	// ProxyBindAddr 指定代理绑定的地址。此值可能与 BindAddr 相同。
	ProxyBindAddr string `ini:"proxy_bind_addr" json:"proxy_bind_addr"`
	// VhostHTTPPort 指定服务器监听 HTTP Vhost 请求的端口。如果此值为 0，则服务器不会监听 HTTP 请求。
	// 默认情况下，此值为 0。
	VhostHTTPPort int `ini:"vhost_http_port" json:"vhost_http_port"`
	// VhostHTTPSPort 指定服务器监听 HTTPS Vhost 请求的端口。如果此值为 0，则服务器不会监听 HTTPS 请求。
	// 默认情况下，此值为 0。
	VhostHTTPSPort int `ini:"vhost_https_port" json:"vhost_https_port"`
	// TCPMuxHTTPConnectPort 指定服务器监听 TCP HTTP CONNECT 请求的端口。
	// 如果此值为 0，则服务器不会在单个端口上多路复用 TCP 请求。
	// 如果不是 -1，则将在此值上监听 HTTP CONNECT 请求。默认情况下，此值为 0。
	TCPMuxHTTPConnectPort int `ini:"tcpmux_httpconnect_port" json:"tcpmux_httpconnect_port"`
	// 如果 TCPMuxPassthrough 为 true，frps 不会对流量进行任何更新。
	TCPMuxPassthrough bool `ini:"tcpmux_passthrough" json:"tcpmux_passthrough"`
	// VhostHTTPTimeout 指定 Vhost HTTP 服务器的响应头超时（以秒为单位）。默认情况下，此值为 60。
	VhostHTTPTimeout int64 `ini:"vhost_http_timeout" json:"vhost_http_timeout"`
	// DashboardAddr 指定仪表板绑定的地址。默认情况下，此值为 "0.0.0.0"。
	DashboardAddr string `ini:"dashboard_addr" json:"dashboard_addr"`
	// DashboardPort 指定仪表板监听的端口。如果此值为 0，则不会启动仪表板。默认情况下，此值为 0。
	DashboardPort int `ini:"dashboard_port" json:"dashboard_port"`
	// DashboardTLSCertFile 指定服务器将加载的证书文件路径。
	// 如果 "dashboard_tls_cert_file"、"dashboard_tls_key_file" 有效，则服务器将使用此提供的 TLS 配置。
	DashboardTLSCertFile string `ini:"dashboard_tls_cert_file" json:"dashboard_tls_cert_file"`
	// DashboardTLSKeyFile 指定服务器将加载的密钥文件路径。
	// 如果 "dashboard_tls_cert_file"、"dashboard_tls_key_file" 有效，则服务器将使用此提供的 TLS 配置。
	DashboardTLSKeyFile string `ini:"dashboard_tls_key_file" json:"dashboard_tls_key_file"`
	// DashboardTLSMode 指定仪表板在 HTTP 或 HTTPS 模式之间的模式。默认情况下，此值为 false，即 HTTP 模式。
	DashboardTLSMode bool `ini:"dashboard_tls_mode" json:"dashboard_tls_mode"`
	// DashboardUser 指定仪表板用于登录的用户名。
	DashboardUser string `ini:"dashboard_user" json:"dashboard_user"`
	// DashboardPwd 指定仪表板用于登录的密码。
	DashboardPwd string `ini:"dashboard_pwd" json:"dashboard_pwd"`
	// EnablePrometheus 将在 {dashboard_addr}:{dashboard_port} 的 /metrics API 上导出 Prometheus 指标。
	EnablePrometheus bool `ini:"enable_prometheus" json:"enable_prometheus"`
	// AssetsDir 指定仪表板从中加载资源的本地目录。如果此值为 ""，则将使用 statik 从捆绑的可执行文件中加载资源。
	// 默认情况下，此值为 ""。
	AssetsDir string `ini:"assets_dir" json:"assets_dir"`
	// LogFile 指定日志写入的文件。只有当 LogWay 设置为适当值时才会使用此值。默认情况下，此值为 "console"。
	LogFile string `ini:"log_file" json:"log_file"`
	// LogWay 指定日志管理方式。有效值为 "console" 或 "file"。
	// 如果使用 "console"，日志将打印到 stdout。如果使用 "file"，日志将打印到 LogFile。默认情况下，此值为 "console"。
	LogWay string `ini:"log_way" json:"log_way"`
	// LogLevel 指定最小日志级别。有效值为 "trace"、"debug"、"info"、"warn" 和 "error"。默认情况下，此值为 "info"。
	LogLevel string `ini:"log_level" json:"log_level"`
	// LogMaxDays 指定删除前存储日志信息的最大天数。仅当 LogWay == "file" 时使用。默认情况下，此值为 0。
	LogMaxDays int64 `ini:"log_max_days" json:"log_max_days"`
	// DisableLogColor 当设置为 true 时，在 LogWay == "console" 时禁用日志颜色。默认情况下，此值为 false。
	DisableLogColor bool `ini:"disable_log_color" json:"disable_log_color"`
	// DetailedErrorsToClient 定义是否向 frpc 发送特定错误（包含调试信息）。默认情况下，此值为 true。
	DetailedErrorsToClient bool `ini:"detailed_errors_to_client" json:"detailed_errors_to_client"`

	// SubDomainHost 指定在使用 Vhost 代理时附加到客户端请求的子域名的域名。
	// 例如，如果此值设置为 "frps.com" 且客户端请求子域名 "test"，则生成的 URL 将为 "test.frps.com"。
	// 默认情况下，此值为 ""。
	SubDomainHost string `ini:"subdomain_host" json:"subdomain_host"`
	// TCPMux 切换 TCP 流多路复用。这允许来自客户端的多个请求共享单个 TCP 连接。
	// 默认情况下，此值为 true。
	TCPMux bool `ini:"tcp_mux" json:"tcp_mux"`
	// TCPMuxKeepaliveInterval 指定 TCP 流多路复用器的保活间隔。
	// 如果 TCPMux 为 true，则不需要应用层心跳，因为它只能依赖 TCPMux 中的心跳。
	TCPMuxKeepaliveInterval int64 `ini:"tcp_mux_keepalive_interval" json:"tcp_mux_keepalive_interval"`
	// TCPKeepAlive 指定 frpc 和 frps 之间的活动网络连接的保活探测之间的间隔。
	// 如果为负数，则禁用保活探测。
	TCPKeepAlive int64 `ini:"tcp_keepalive" json:"tcp_keepalive"`
	// Custom404Page 指定要显示的自定义 404 页面的路径。如果此值为 ""，则将显示默认页面。
	// 默认情况下，此值为 ""。
	Custom404Page string `ini:"custom_404_page" json:"custom_404_page"`

	// AllowPorts 指定客户端可以代理到的端口集合。如果此值的长度为 0，则允许所有端口。
	// 默认情况下，此值为空集合。
	AllowPorts map[int]struct{} `ini:"-" json:"-"`
	// 原始字符串。
	AllowPortsStr string `ini:"-" json:"-"`
	// MaxPoolCount 指定每个代理的最大池大小。默认情况下，此值为 5。
	MaxPoolCount int64 `ini:"max_pool_count" json:"max_pool_count"`
	// MaxPortsPerClient 指定单个客户端可以代理到的最大端口数。如果此值为 0，则不应用限制。
	// 默认情况下，此值为 0。
	MaxPortsPerClient int64 `ini:"max_ports_per_client" json:"max_ports_per_client"`
	// TLSOnly 指定是否仅接受 TLS 加密连接。默认情况下，此值为 false。
	TLSOnly bool `ini:"tls_only" json:"tls_only"`
	// TLSCertFile 指定服务器将加载的证书文件路径。如果 "tls_cert_file"、"tls_key_file" 有效，
	// 则服务器将使用此提供的 TLS 配置。否则，服务器将使用自身生成的 TLS 配置。
	TLSCertFile string `ini:"tls_cert_file" json:"tls_cert_file"`
	// TLSKeyFile 指定服务器将加载的密钥文件路径。如果 "tls_cert_file"、"tls_key_file" 有效，
	// 则服务器将使用此提供的 TLS 配置。否则，服务器将使用自身生成的 TLS 配置。
	TLSKeyFile string `ini:"tls_key_file" json:"tls_key_file"`
	// TLSTrustedCaFile 指定服务器将加载的客户端证书文件的路径。它仅在 "tls_only" 为 true 时有效。
	// 如果 "tls_trusted_ca_file" 有效，则服务器将验证每个客户端的证书。
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file" json:"tls_trusted_ca_file"`
	// HeartBeatTimeout 指定在终止连接之前等待心跳的最长时间。不建议更改此值。
	// 默认情况下，此值为 90。设置为负值以禁用它。
	HeartbeatTimeout int64 `ini:"heartbeat_timeout" json:"heartbeat_timeout"`
	// UserConnTimeout 指定等待工作连接的最长时间。默认情况下，此值为 10。
	UserConnTimeout int64 `ini:"user_conn_timeout" json:"user_conn_timeout"`
	// HTTPPlugins 指定支持 HTTP 协议的服务器插件。
	HTTPPlugins map[string]HTTPPluginOptions `ini:"-" json:"http_plugins"`
	// UDPPacketSize 指定 UDP 数据包大小。默认情况下，此值为 1500。
	UDPPacketSize int64 `ini:"udp_packet_size" json:"udp_packet_size"`
	// 在仪表板监听器中启用 golang pprof 处理程序。必须先设置仪表板端口。
	PprofEnable bool `ini:"pprof_enable" json:"pprof_enable"`
	// NatHoleAnalysisDataReserveHours 指定保留 NAT 穿透分析数据的小时数。
	NatHoleAnalysisDataReserveHours int64 `ini:"nat_hole_analysis_data_reserve_hours" json:"nat_hole_analysis_data_reserve_hours"`
}

// GetDefaultServerConf 返回具有合理默认值的服务器配置。
// 注意：此处的一些默认值将被设置为空，并将通过 'Complete' 函数将它们转换为新配置，
// 以将它们设置为新配置的默认值。
func GetDefaultServerConf() ServerCommonConf {
	return ServerCommonConf{
		ServerConfig:           legacyauth.GetDefaultServerConf(),
		DashboardAddr:          "0.0.0.0",
		LogFile:                "console",
		LogWay:                 "console",
		DetailedErrorsToClient: true,
		TCPMux:                 true,
		AllowPorts:             make(map[int]struct{}),
		HTTPPlugins:            make(map[string]HTTPPluginOptions),
	}
}

// UnmarshalServerConfFromIni 从 INI 格式的源数据解析服务器配置。
// 返回解析后的 ServerCommonConf 结构体和可能的错误。
func UnmarshalServerConfFromIni(source any) (ServerCommonConf, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ServerCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		return ServerCommonConf{}, err
	}

	common := GetDefaultServerConf()
	err = s.MapTo(&common)
	if err != nil {
		return ServerCommonConf{}, err
	}

	// allow_ports
	allowPortStr := s.Key("allow_ports").String()
	if allowPortStr != "" {
		common.AllowPortsStr = allowPortStr
	}

	// plugin.xxx
	pluginOpts := make(map[string]HTTPPluginOptions)
	for _, section := range f.Sections() {
		name := section.Name()
		if !strings.HasPrefix(name, "plugin.") {
			continue
		}

		opt, err := loadHTTPPluginOpt(section)
		if err != nil {
			return ServerCommonConf{}, err
		}

		pluginOpts[opt.Name] = *opt
	}
	common.HTTPPlugins = pluginOpts

	return common, nil
}

// loadHTTPPluginOpt 从 INI 配置节加载 HTTP 插件选项。
// 返回解析后的 HTTPPluginOptions 指针和可能的错误。
func loadHTTPPluginOpt(section *ini.Section) (*HTTPPluginOptions, error) {
	name := strings.TrimSpace(strings.TrimPrefix(section.Name(), "plugin."))

	opt := &HTTPPluginOptions{}
	err := section.MapTo(opt)
	if err != nil {
		return nil, err
	}

	opt.Name = name

	return opt, nil
}
