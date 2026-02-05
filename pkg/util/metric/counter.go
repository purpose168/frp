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
	"sync/atomic"
)

// Counter 是计数器接口
type Counter interface {
	// Count 返回当前计数值
	Count() int32
	// Inc 增加计数值
	Inc(int32)
	// Dec 减少计数值
	Dec(int32)
	// Snapshot 创建计数器的快照
	Snapshot() Counter
	// Clear 清零计数值
	Clear()
}

// NewCounter 创建新的计数器
// 返回计数器实例
func NewCounter() Counter {
	return &StandardCounter{
		count: 0,
	}
}

// StandardCounter 是标准计数器实现
type StandardCounter struct {
	// count 是计数值
	count int32
}

// Count 返回当前计数值
func (c *StandardCounter) Count() int32 {
	return atomic.LoadInt32(&c.count)
}

// Inc 增加计数值
// 参数 count 是要增加的值
func (c *StandardCounter) Inc(count int32) {
	atomic.AddInt32(&c.count, count)
}

// Dec 减少计数值
// 参数 count 是要减少的值
func (c *StandardCounter) Dec(count int32) {
	atomic.AddInt32(&c.count, -count)
}

// Snapshot 创建计数器的快照
// 返回计数器快照
func (c *StandardCounter) Snapshot() Counter {
	tmp := &StandardCounter{
		count: atomic.LoadInt32(&c.count),
	}
	return tmp
}

// Clear 清零计数值
func (c *StandardCounter) Clear() {
	atomic.StoreInt32(&c.count, 0)
}
