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

package wait

import (
	"math/rand/v2"
	"time"

	"github.com/purpose168/frp/pkg/util/util"
)

// BackoffFunc 退避函数类型，根据前一次持续时间和条件错误状态计算新的退避时间
type BackoffFunc func(previousDuration time.Duration, previousConditionError bool) time.Duration

// Backoff 实现退避函数接口
func (f BackoffFunc) Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration {
	return f(previousDuration, previousConditionError)
}

// BackoffManager 退避管理器接口
type BackoffManager interface {
	Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration
}

// FastBackoffOptions 快速退避选项
type FastBackoffOptions struct {
	Duration           time.Duration // 基础持续时间
	Factor             float64       // 退避因子，每次重试后持续时间乘以该因子
	Jitter             float64       // 抖动因子，用于在退避时间上添加随机性
	MaxDuration        time.Duration // 最大持续时间
	InitDurationIfFail time.Duration // 首次失败时的初始持续时间

	// 如果 FastRetryCount > 0，则在 FastRetryWindow 时间窗口内，
	// 前 FastRetryCount 次调用将以 FastRetryDelay 的延迟执行重试。
	FastRetryCount  int           // 快速重试次数
	FastRetryDelay  time.Duration // 快速重试延迟
	FastRetryJitter float64       // 快速重试抖动因子
	FastRetryWindow time.Duration // 快速重试时间窗口
}

// fastBackoffImpl 快速退避实现
type fastBackoffImpl struct {
	options FastBackoffOptions

	lastCalledTime      time.Time // 上次调用时间
	consecutiveErrCount int       // 连续错误计数

	fastRetryCutoffTime     time.Time // 快速重试截止时间
	countsInFastRetryWindow int       // 快速重试窗口内的计数
}

// NewFastBackoffManager 创建一个新的快速退避管理器
func NewFastBackoffManager(options FastBackoffOptions) BackoffManager {
	return &fastBackoffImpl{
		options:                 options,
		countsInFastRetryWindow: 1,
	}
}

// Backoff 根据前一次持续时间和条件错误状态计算新的退避时间
func (f *fastBackoffImpl) Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration {
	// 如果是首次调用，返回基础持续时间
	if f.lastCalledTime.IsZero() {
		f.lastCalledTime = time.Now()
		return f.options.Duration
	}
	now := time.Now()
	f.lastCalledTime = now

	// 根据条件错误状态更新连续错误计数
	if previousConditionError {
		f.consecutiveErrCount++
	} else {
		f.consecutiveErrCount = 0
	}

	// 如果启用了快速重试且发生了错误
	if f.options.FastRetryCount > 0 && previousConditionError {
		f.countsInFastRetryWindow++
		// 如果在快速重试次数范围内，使用快速重试延迟
		if f.countsInFastRetryWindow <= f.options.FastRetryCount {
			return Jitter(f.options.FastRetryDelay, f.options.FastRetryJitter)
		}
		// 如果超过了快速重试时间窗口，重置计数
		if now.After(f.fastRetryCutoffTime) {
			// 重置
			f.fastRetryCutoffTime = now.Add(f.options.FastRetryWindow)
			f.countsInFastRetryWindow = 0
		}
	}

	// 如果发生了错误，计算退避时间
	if previousConditionError {
		var duration time.Duration
		// 如果是首次错误，使用初始失败持续时间
		if f.consecutiveErrCount == 1 {
			duration = util.EmptyOr(f.options.InitDurationIfFail, previousDuration)
		} else {
			duration = previousDuration
		}

		// 设置默认持续时间为 1 秒
		duration = util.EmptyOr(duration, time.Second)
		// 应用退避因子
		if f.options.Factor != 0 {
			duration = time.Duration(float64(duration) * f.options.Factor)
		}
		// 应用抖动
		if f.options.Jitter > 0 {
			duration = Jitter(duration, f.options.Jitter)
		}
		// 限制最大持续时间
		if f.options.MaxDuration > 0 && duration > f.options.MaxDuration {
			duration = f.options.MaxDuration
		}
		return duration
	}
	// 如果没有错误，返回基础持续时间
	return f.options.Duration
}

// BackoffUntil 重复执行函数 f，直到返回 true 或 stopCh 被关闭
// backoff 参数控制重试之间的延迟
// sliding 参数控制是否使用滑动窗口退避
func BackoffUntil(f func() (bool, error), backoff BackoffManager, sliding bool, stopCh <-chan struct{}) {
	var delay time.Duration
	previousError := false

	ticker := time.NewTicker(backoff.Backoff(delay, previousError))
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		// 如果不使用滑动窗口，在执行前计算退避时间
		if !sliding {
			delay = backoff.Backoff(delay, previousError)
		}

		// 执行函数
		if done, err := f(); done {
			return
		} else if err != nil {
			previousError = true
		} else {
			previousError = false
		}

		// 如果使用滑动窗口，在执行后计算退避时间
		if sliding {
			delay = backoff.Backoff(delay, previousError)
		}

		ticker.Reset(delay)
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
	}
}

// Jitter 返回一个在 duration 和 duration + maxFactor * duration 之间的时间持续时间
//
// 这允许客户端避免收敛到周期性行为。如果 maxFactor 为 0.0，将选择建议的默认值。
func Jitter(duration time.Duration, maxFactor float64) time.Duration {
	if maxFactor <= 0.0 {
		maxFactor = 1.0
	}
	wait := duration + time.Duration(rand.Float64()*maxFactor*float64(duration))
	return wait
}

// Until 以固定的周期重复执行函数 f，直到 stopCh 被关闭
func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
	ff := func() (bool, error) {
		f()
		return false, nil
	}
	BackoffUntil(ff, BackoffFunc(func(time.Duration, bool) time.Duration {
		return period
	}), true, stopCh)
}
