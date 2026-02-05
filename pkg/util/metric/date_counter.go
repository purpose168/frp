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

package metric

import (
	"sync"
	"time"
)

// DateCounter 是日期计数器接口
type DateCounter interface {
	// TodayCount 返回今天的计数值
	TodayCount() int64
	// GetLastDaysCount 返回最近几天的计数值
	GetLastDaysCount(lastdays int64) []int64
	// Inc 增加计数值
	Inc(int64)
	// Dec 减少计数值
	Dec(int64)
	// Snapshot 创建计数器的快照
	Snapshot() DateCounter
	// Clear 清零计数值
	Clear()
}

// NewDateCounter 创建新的日期计数器
// 参数 reserveDays 是保留天数
// 返回日期计数器实例
func NewDateCounter(reserveDays int64) DateCounter {
	if reserveDays <= 0 {
		reserveDays = 1
	}
	return newStandardDateCounter(reserveDays)
}

// StandardDateCounter 是标准日期计数器实现
type StandardDateCounter struct {
	// reserveDays 是保留天数
	reserveDays int64
	// counts 是每天的计数值
	counts []int64

	// lastUpdateDate 是最后更新日期
	lastUpdateDate time.Time
	// mu 是互斥锁
	mu sync.Mutex
}

// newStandardDateCounter 创建新的标准日期计数器
// 参数 reserveDays 是保留天数
// 返回标准日期计数器实例
func newStandardDateCounter(reserveDays int64) *StandardDateCounter {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s := &StandardDateCounter{
		reserveDays:    reserveDays,
		counts:         make([]int64, reserveDays),
		lastUpdateDate: now,
	}
	return s
}

// TodayCount 返回今天的计数值
func (c *StandardDateCounter) TodayCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rotate(time.Now())
	return c.counts[0]
}

// GetLastDaysCount 返回最近几天的计数值
// 参数 lastdays 是要获取的天数
// 返回每天的计数值数组
func (c *StandardDateCounter) GetLastDaysCount(lastdays int64) []int64 {
	if lastdays > c.reserveDays {
		lastdays = c.reserveDays
	}
	counts := make([]int64, lastdays)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	for i := 0; i < int(lastdays); i++ {
		counts[i] = c.counts[i]
	}
	return counts
}

// Inc 增加今天的计数值
// 参数 count 是要增加的值
func (c *StandardDateCounter) Inc(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	c.counts[0] += count
}

// Dec 减少今天的计数值
// 参数 count 是要减少的值
func (c *StandardDateCounter) Dec(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	c.counts[0] -= count
}

// Snapshot 创建计数器的快照
// 返回日期计数器快照
func (c *StandardDateCounter) Snapshot() DateCounter {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := newStandardDateCounter(c.reserveDays)
	for i := 0; i < int(c.reserveDays); i++ {
		tmp.counts[i] = c.counts[i]
	}
	return tmp
}

// Clear 清零所有计数值
func (c *StandardDateCounter) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < int(c.reserveDays); i++ {
		c.counts[i] = 0
	}
}

// rotate 旋转计数器，将旧数据移到后面
// 调用此函数前必须持有锁
func (c *StandardDateCounter) rotate(now time.Time) {
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(now.Sub(c.lastUpdateDate).Hours() / 24)

	defer func() {
		c.lastUpdateDate = now
	}()

	if days <= 0 {
		return
	} else if days >= int(c.reserveDays) {
		c.counts = make([]int64, c.reserveDays)
		return
	}
	newCounts := make([]int64, c.reserveDays)

	for i := days; i < int(c.reserveDays); i++ {
		newCounts[i] = c.counts[i-days]
	}
	c.counts = newCounts
}
