// Copyright 2017 fatedier, fatedier@gmail.com
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

package mem

import (
	"time"

	"github.com/fatedier/frp/pkg/util/metric"
)

const (
	// ReserveDays 保留天数，用于存储历史流量数据
	ReserveDays = 7
)

// ServerStats 服务器统计信息，用于对外展示服务器级别的统计数据
type ServerStats struct {
	// TotalTrafficIn 服务器总入站流量（字节）
	TotalTrafficIn int64
	// TotalTrafficOut 服务器总出站流量（字节）
	TotalTrafficOut int64
	// CurConns 当前连接数
	CurConns int64
	// ClientCounts 客户端数量
	ClientCounts int64
	// ProxyTypeCounts 各类型代理的数量统计，key 为代理类型，value 为数量
	ProxyTypeCounts map[string]int64
}

// ProxyStats 代理统计信息，用于对外展示单个代理的统计数据
type ProxyStats struct {
	// Name 代理名称
	Name string
	// Type 代理类型
	Type string
	// User 用户名
	User string
	// ClientID 客户端ID
	ClientID string
	// TodayTrafficIn 今日入站流量（字节）
	TodayTrafficIn int64
	// TodayTrafficOut 今日出站流量（字节）
	TodayTrafficOut int64
	// LastStartTime 最后启动时间
	LastStartTime string
	// LastCloseTime 最后关闭时间
	LastCloseTime string
	// CurConns 当前连接数
	CurConns int64
}

// ProxyTrafficInfo 代理流量信息，用于展示代理的历史流量数据
type ProxyTrafficInfo struct {
	// Name 代理名称
	Name string
	// TrafficIn 入站流量历史数据，按日期存储
	TrafficIn []int64
	// TrafficOut 出站流量历史数据，按日期存储
	TrafficOut []int64
}

// ProxyStatistics 代理统计信息，用于内部存储代理的详细统计数据
type ProxyStatistics struct {
	// Name 代理名称
	Name string
	// ProxyType 代理类型
	ProxyType string
	// User 用户名
	User string
	// ClientID 客户端ID
	ClientID string
	// TrafficIn 入站流量计数器，按日期存储
	TrafficIn metric.DateCounter
	// TrafficOut 出站流量计数器，按日期存储
	TrafficOut metric.DateCounter
	// CurConns 当前连接数计数器
	CurConns metric.Counter
	// LastStartTime 最后启动时间
	LastStartTime time.Time
	// LastCloseTime 最后关闭时间
	LastCloseTime time.Time
}

// ServerStatistics 服务器统计信息，用于内部存储服务器的详细统计数据
type ServerStatistics struct {
	// TotalTrafficIn 总入站流量计数器，按日期存储
	TotalTrafficIn metric.DateCounter
	// TotalTrafficOut 总出站流量计数器，按日期存储
	TotalTrafficOut metric.DateCounter
	// CurConns 当前连接数计数器
	CurConns metric.Counter

	// ClientCounts 客户端数量计数器
	ClientCounts metric.Counter

	// ProxyTypeCounts 各类型代理的数量统计，key 为代理类型，value 为计数器
	ProxyTypeCounts map[string]metric.Counter

	// ProxyStatistics 不同代理的统计信息，key 为代理名称
	ProxyStatistics map[string]*ProxyStatistics
}

// Collector 指标收集器接口，定义了收集和获取服务器及代理指标的方法
type Collector interface {
	// GetServer 获取服务器统计信息
	GetServer() *ServerStats
	// GetProxiesByType 根据代理类型获取代理统计信息列表
	GetProxiesByType(proxyType string) []*ProxyStats
	// GetProxiesByTypeAndName 根据代理类型和名称获取代理统计信息
	GetProxiesByTypeAndName(proxyType string, proxyName string) *ProxyStats
	// GetProxyByName 根据代理名称获取代理统计信息
	GetProxyByName(proxyName string) *ProxyStats
	// GetProxyTraffic 获取代理流量信息
	GetProxyTraffic(name string) *ProxyTrafficInfo
	// ClearOfflineProxies 清理离线代理，返回清理的代理数量和清理的连接数量
	ClearOfflineProxies() (int, int)
}
