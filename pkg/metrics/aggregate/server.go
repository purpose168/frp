// Copyright 2020 fatedier, fatedier@gmail.com
//
// Licensed under to Apache License, Version 2.0 (the "License");
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

package aggregate

import (
	"github.com/fatedier/frp/pkg/metrics/mem"
	"github.com/fatedier/frp/pkg/metrics/prometheus"
	"github.com/fatedier/frp/server/metrics"
)

// EnableMem 启用以标记指标到内存监控系统
func EnableMem() {
	sm.Add(mem.ServerMetrics)
}

// EnablePrometheus 启用以标记指标到 Prometheus
func EnablePrometheus() {
	sm.Add(prometheus.ServerMetrics)
}

var sm = &serverMetrics{}

// init 初始化函数，注册服务器指标
func init() {
	metrics.Register(sm)
}

// serverMetrics 服务器指标结构体
type serverMetrics struct {
	ms []metrics.ServerMetrics
}

// Add 添加服务器指标
func (m *serverMetrics) Add(sm metrics.ServerMetrics) {
	m.ms = append(m.ms, sm)
}

// NewClient 创建新客户端
func (m *serverMetrics) NewClient() {
	for _, v := range m.ms {
		v.NewClient()
	}
}

// CloseClient 关闭客户端
func (m *serverMetrics) CloseClient() {
	for _, v := range m.ms {
		v.CloseClient()
	}
}

// NewProxy 创建新代理
func (m *serverMetrics) NewProxy(name string, proxyType string, user string, clientID string) {
	for _, v := range m.ms {
		v.NewProxy(name, proxyType, user, clientID)
	}
}

// CloseProxy 关闭代理
func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	for _, v := range m.ms {
		v.CloseProxy(name, proxyType)
	}
}

// OpenConnection 打开连接
func (m *serverMetrics) OpenConnection(name string, proxyType string) {
	for _, v := range m.ms {
		v.OpenConnection(name, proxyType)
	}
}

// CloseConnection 关闭连接
func (m *serverMetrics) CloseConnection(name string, proxyType string) {
	for _, v := range m.ms {
		v.CloseConnection(name, proxyType)
	}
}

// AddTrafficIn 添加入站流量
func (m *serverMetrics) AddTrafficIn(name string, proxyType string, trafficBytes int64) {
	for _, v := range m.ms {
		v.AddTrafficIn(name, proxyType, trafficBytes)
	}
}

// AddTrafficOut 添加出站流量
func (m *serverMetrics) AddTrafficOut(name string, proxyType string, trafficBytes int64) {
	for _, v := range m.ms {
		v.AddTrafficOut(name, proxyType, trafficBytes)
	}
}
