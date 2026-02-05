package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/purpose168/frp/server/metrics"
)

const (
	// namespace 命名空间
	namespace = "frp"
	// serverSubsystem 服务器子系统
	serverSubsystem = "server"
)

// ServerMetrics 服务器指标实例
var ServerMetrics metrics.ServerMetrics = newServerMetrics()

// serverMetrics 服务器指标结构体
type serverMetrics struct {
	// clientCount 客户端数量
	clientCount prometheus.Gauge
	// proxyCount 代理数量
	proxyCount *prometheus.GaugeVec
	// proxyCountDetailed 详细代理数量
	proxyCountDetailed *prometheus.GaugeVec
	// connectionCount 连接数量
	connectionCount *prometheus.GaugeVec
	// trafficIn 入站流量
	trafficIn *prometheus.CounterVec
	// trafficOut 出站流量
	trafficOut *prometheus.CounterVec
}

// NewClient 创建新客户端
func (m *serverMetrics) NewClient() {
	m.clientCount.Inc()
}

// CloseClient 关闭客户端
func (m *serverMetrics) CloseClient() {
	m.clientCount.Dec()
}

// NewProxy 创建新代理
func (m *serverMetrics) NewProxy(name string, proxyType string, _ string, _ string) {
	m.proxyCount.WithLabelValues(proxyType).Inc()
	m.proxyCountDetailed.WithLabelValues(proxyType, name).Inc()
}

// CloseProxy 关闭代理
func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.proxyCount.WithLabelValues(proxyType).Dec()
	m.proxyCountDetailed.WithLabelValues(proxyType, name).Dec()
}

// OpenConnection 打开连接
func (m *serverMetrics) OpenConnection(name string, proxyType string) {
	m.connectionCount.WithLabelValues(name, proxyType).Inc()
}

// CloseConnection 关闭连接
func (m *serverMetrics) CloseConnection(name string, proxyType string) {
	m.connectionCount.WithLabelValues(name, proxyType).Dec()
}

// AddTrafficIn 添加入站流量
func (m *serverMetrics) AddTrafficIn(name string, proxyType string, trafficBytes int64) {
	m.trafficIn.WithLabelValues(name, proxyType).Add(float64(trafficBytes))
}

// AddTrafficOut 添加出站流量
func (m *serverMetrics) AddTrafficOut(name string, proxyType string, trafficBytes int64) {
	m.trafficOut.WithLabelValues(name, proxyType).Add(float64(trafficBytes))
}

// newServerMetrics 创建新的服务器指标
func newServerMetrics() *serverMetrics {
	m := &serverMetrics{
		clientCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "client_counts",
			Help:      "frps 的当前客户端数量",
		}),
		proxyCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "proxy_counts",
			Help:      "当前代理数量",
		}, []string{"type"}),
		proxyCountDetailed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "proxy_counts_detailed",
			Help:      "按类型和名称分组的当前代理数量",
		}, []string{"type", "name"}),
		connectionCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "connection_counts",
			Help:      "当前连接数量",
		}, []string{"name", "type"}),
		trafficIn: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "traffic_in",
			Help:      "总入站流量",
		}, []string{"name", "type"}),
		trafficOut: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "traffic_out",
			Help:      "总出站流量",
		}, []string{"name", "type"}),
	}
	prometheus.MustRegister(m.clientCount)
	prometheus.MustRegister(m.proxyCount)
	prometheus.MustRegister(m.proxyCountDetailed)
	prometheus.MustRegister(m.connectionCount)
	prometheus.MustRegister(m.trafficIn)
	prometheus.MustRegister(m.trafficOut)
	return m
}
