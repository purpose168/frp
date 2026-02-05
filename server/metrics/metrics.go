package metrics

import (
	"sync"
)

// ServerMetrics 服务器指标收集接口
// 定义了收集服务器各种运行指标的方法
type ServerMetrics interface {
	// NewClient 当新客户端连接时调用
	NewClient()
	// CloseClient 当客户端断开连接时调用
	CloseClient()
	// NewProxy 当新代理创建时调用
	// 参数name是代理名称
	// 参数proxyType是代理类型
	// 参数user是用户名
	// 参数clientID是客户端ID
	NewProxy(name string, proxyType string, user string, clientID string)
	// CloseProxy 当代理关闭时调用
	// 参数name是代理名称
	// 参数proxyType是代理类型
	CloseProxy(name string, proxyType string)
	// OpenConnection 当新连接建立时调用
	// 参数name是代理名称
	// 参数proxyType是代理类型
	OpenConnection(name string, proxyType string)
	// CloseConnection 当连接关闭时调用
	// 参数name是代理名称
	// 参数proxyType是代理类型
	CloseConnection(name string, proxyType string)
	// AddTrafficIn 增加入站流量统计
	// 参数name是代理名称
	// 参数proxyType是代理类型
	// 参数trafficBytes是流量字节数
	AddTrafficIn(name string, proxyType string, trafficBytes int64)
	// AddTrafficOut 增加出站流量统计
	// 参数name是代理名称
	// 参数proxyType是代理类型
	// 参数trafficBytes是流量字节数
	AddTrafficOut(name string, proxyType string, trafficBytes int64)
}

// Server 全局服务器指标收集实例
var Server ServerMetrics = noopServerMetrics{}

// registerMetrics 确保指标收集器只注册一次
var registerMetrics sync.Once

// Register 注册服务器指标收集器
// 该函数只能被调用一次
// 参数m是要注册的指标收集器
func Register(m ServerMetrics) {
	registerMetrics.Do(func() {
		Server = m
	})
}

// noopServerMetrics 空实现的服务器指标收集器
// 用于默认情况下不收集任何指标
type noopServerMetrics struct{}

func (noopServerMetrics) NewClient()                              {}
func (noopServerMetrics) CloseClient()                            {}
func (noopServerMetrics) NewProxy(string, string, string, string) {}
func (noopServerMetrics) CloseProxy(string, string)               {}
func (noopServerMetrics) OpenConnection(string, string)           {}
func (noopServerMetrics) CloseConnection(string, string)          {}
func (noopServerMetrics) AddTrafficIn(string, string, int64)      {}
func (noopServerMetrics) AddTrafficOut(string, string, int64)     {}
