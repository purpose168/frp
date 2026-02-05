// Copyright 2019 fatedier, fatedier@gmail.com
//
// Licensed under to Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with License.
// You may obtain a copy of License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under to License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under to License.

package mem

import (
	"sync"
	"time"

	"github.com/purpose168/frp/pkg/util/log"
	"github.com/purpose168/frp/pkg/util/metric"
	server "github.com/purpose168/frp/server/metrics"
)

var (
	sm = newServerMetrics()

	ServerMetrics  server.ServerMetrics
	StatsCollector Collector
)

// init 初始化函数，设置服务器指标和统计收集器
func init() {
	ServerMetrics = sm
	StatsCollector = sm
	sm.run()
}

// serverMetrics 服务器指标结构体
type serverMetrics struct {
	info *ServerStatistics
	mu   sync.Mutex
}

// newServerMetrics 创建新的服务器指标
func newServerMetrics() *serverMetrics {
	return &serverMetrics{
		info: &ServerStatistics{
			TotalTrafficIn:  metric.NewDateCounter(ReserveDays),
			TotalTrafficOut: metric.NewDateCounter(ReserveDays),
			CurConns:        metric.NewCounter(),

			ClientCounts:    metric.NewCounter(),
			ProxyTypeCounts: make(map[string]metric.Counter),

			ProxyStatistics: make(map[string]*ProxyStatistics),
		},
	}
}

// run 运行服务器指标清理任务
func (m *serverMetrics) run() {
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			start := time.Now()
			count, total := m.clearUselessInfo(time.Duration(7*24) * time.Hour)
			log.Debugf("清除无用的代理统计数据计数 %d/%d，耗时 %v", count, total, time.Since(start))
		}
	}()
}

// clearUselessInfo 清理无用的代理统计信息
func (m *serverMetrics) clearUselessInfo(continuousOfflineDuration time.Duration) (int, int) {
	count := 0
	total := 0
	// 检查是否有任何已关闭超过 continuousOfflineDuration 的代理并删除它们
	m.mu.Lock()
	defer m.mu.Unlock()
	total = len(m.info.ProxyStatistics)
	for name, data := range m.info.ProxyStatistics {
		if !data.LastCloseTime.IsZero() &&
			data.LastStartTime.Before(data.LastCloseTime) &&
			time.Since(data.LastCloseTime) > continuousOfflineDuration {
			delete(m.info.ProxyStatistics, name)
			count++
			log.Tracef("清除代理 [%s] 的统计数据，最后关闭时间：[%s]", name, data.LastCloseTime.String())
		}
	}
	return count, total
}

// ClearOfflineProxies 清理离线代理
func (m *serverMetrics) ClearOfflineProxies() (int, int) {
	return m.clearUselessInfo(0)
}

// NewClient 创建新客户端
func (m *serverMetrics) NewClient() {
	m.info.ClientCounts.Inc(1)
}

// CloseClient 关闭客户端
func (m *serverMetrics) CloseClient() {
	m.info.ClientCounts.Dec(1)
}

// NewProxy 创建新代理
func (m *serverMetrics) NewProxy(name string, proxyType string, user string, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counter, ok := m.info.ProxyTypeCounts[proxyType]
	if !ok {
		counter = metric.NewCounter()
	}
	counter.Inc(1)
	m.info.ProxyTypeCounts[proxyType] = counter

	proxyStats, ok := m.info.ProxyStatistics[name]
	if !ok || proxyStats.ProxyType != proxyType {
		proxyStats = &ProxyStatistics{
			Name:       name,
			ProxyType:  proxyType,
			CurConns:   metric.NewCounter(),
			TrafficIn:  metric.NewDateCounter(ReserveDays),
			TrafficOut: metric.NewDateCounter(ReserveDays),
		}
		m.info.ProxyStatistics[name] = proxyStats
	}
	proxyStats.User = user
	proxyStats.ClientID = clientID
	proxyStats.LastStartTime = time.Now()
}

// CloseProxy 关闭代理
func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if counter, ok := m.info.ProxyTypeCounts[proxyType]; ok {
		counter.Dec(1)
	}
	if proxyStats, ok := m.info.ProxyStatistics[name]; ok {
		proxyStats.LastCloseTime = time.Now()
	}
}

// OpenConnection 打开连接
func (m *serverMetrics) OpenConnection(name string, _ string) {
	m.info.CurConns.Inc(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Inc(1)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

// CloseConnection 关闭连接
func (m *serverMetrics) CloseConnection(name string, _ string) {
	m.info.CurConns.Dec(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Dec(1)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

// AddTrafficIn 添加入站流量
func (m *serverMetrics) AddTrafficIn(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficIn.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficIn.Inc(trafficBytes)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

// AddTrafficOut 添加出站流量
func (m *serverMetrics) AddTrafficOut(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficOut.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficOut.Inc(trafficBytes)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

// GetServer 获取服务器统计数据
func (m *serverMetrics) GetServer() *ServerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &ServerStats{
		TotalTrafficIn:  m.info.TotalTrafficIn.TodayCount(),
		TotalTrafficOut: m.info.TotalTrafficOut.TodayCount(),
		CurConns:        int64(m.info.CurConns.Count()),
		ClientCounts:    int64(m.info.ClientCounts.Count()),
		ProxyTypeCounts: make(map[string]int64),
	}
	for k, v := range m.info.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = int64(v.Count())
	}
	return s
}

// GetProxiesByType 按类型获取代理统计
func (m *serverMetrics) GetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, proxyStats := range m.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		ps := &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			User:            proxyStats.User,
			ClientID:        proxyStats.ClientID,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        int64(proxyStats.CurConns.Count()),
		}
		if !proxyStats.LastStartTime.IsZero() {
			ps.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			ps.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		res = append(res, ps)
	}
	return res
}

// GetProxiesByTypeAndName 按类型和名称获取代理统计
func (m *serverMetrics) GetProxiesByTypeAndName(proxyType string, proxyName string) (res *ProxyStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, proxyStats := range m.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		if name != proxyName {
			continue
		}

		res = &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			User:            proxyStats.User,
			ClientID:        proxyStats.ClientID,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        int64(proxyStats.CurConns.Count()),
		}
		if !proxyStats.LastStartTime.IsZero() {
			res.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			res.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		break
	}
	return
}

// GetProxyByName 按名称获取代理统计
func (m *serverMetrics) GetProxyByName(proxyName string) (res *ProxyStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[proxyName]
	if ok {
		res = &ProxyStats{
			Name:            proxyName,
			Type:            proxyStats.ProxyType,
			User:            proxyStats.User,
			ClientID:        proxyStats.ClientID,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        int64(proxyStats.CurConns.Count()),
		}
		if !proxyStats.LastStartTime.IsZero() {
			res.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			res.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
	}
	return
}

// GetProxyTraffic 获取代理流量信息
func (m *serverMetrics) GetProxyTraffic(name string) (res *ProxyTrafficInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		res = &ProxyTrafficInfo{
			Name: name,
		}
		res.TrafficIn = proxyStats.TrafficIn.GetLastDaysCount(ReserveDays)
		res.TrafficOut = proxyStats.TrafficOut.GetLastDaysCount(ReserveDays)
	}
	return
}
