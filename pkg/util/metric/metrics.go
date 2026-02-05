// Copyright 2020 fatedier, fatedier@gmail.com
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

package metric

// GaugeMetric 表示一个可以任意上下波动的数值
type GaugeMetric interface {
	// Inc 增加数值
	Inc()
	// Dec 减少数值
	Dec()
	// Set 设置数值
	Set(float64)
}

// CounterMetric 表示一个只能增加的数值
type CounterMetric interface {
	// Inc 增加数值
	Inc()
}

// HistogramMetric 统计单个观测值
type HistogramMetric interface {
	// Observe 观测一个值
	Observe(float64)
}
