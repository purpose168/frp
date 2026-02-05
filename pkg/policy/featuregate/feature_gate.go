// Copyright 2025 The frp Authors
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

package featuregate

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Feature 表示特性门控的名称
type Feature string

// FeatureStage 表示特性的成熟度级别
type FeatureStage string

const (
	// Alpha 表示该特性是实验性的，默认禁用
	Alpha FeatureStage = "ALPHA"
	// Beta 表示该特性更加稳定但仍可能发生变化，默认禁用
	Beta FeatureStage = "BETA"
	// GA 表示该特性已正式发布，默认启用
	GA FeatureStage = ""
)

// FeatureSpec 描述一个特性及其属性
type FeatureSpec struct {
	// Default 是该特性的默认启用状态
	Default bool
	// LockToDefault 指示该特性不能从其默认状态更改
	LockToDefault bool
	// Stage 指示该特性的成熟度级别
	Stage FeatureStage
}

// 在此处定义所有可用的特性
var (
	VirtualNet = Feature("VirtualNet")
)

// defaultFeatures 定义默认特性及其规范
var defaultFeatures = map[Feature]FeatureSpec{
	// 实际特性
	VirtualNet: {Default: false, Stage: Alpha},
}

// FeatureGate 指示给定特性是否已启用
type FeatureGate interface {
	// Enabled 如果键已启用则返回 true
	Enabled(key Feature) bool
	// KnownFeatures 返回描述已知特性的字符串切片
	KnownFeatures() []string
}

// MutableFeatureGate 允许动态特性门控配置
type MutableFeatureGate interface {
	FeatureGate

	// SetFromMap 从 map[string]bool 设置特性门控值
	SetFromMap(m map[string]bool) error
	// Add 将特性添加到特性门控
	Add(features map[Feature]FeatureSpec) error
	// String 返回表示特性门控配置的字符串
	String() string
}

// featureGate 实现 FeatureGate 和 MutableFeatureGate 接口
type featureGate struct {
	// lock 保护对 known、enabled 的写入以及对 closed 的读/写
	lock sync.Mutex
	// known 保存 map[Feature]FeatureSpec
	known atomic.Value
	// enabled 保存 map[Feature]bool
	enabled atomic.Value
	// closed 一旦特性门控被视为不可变，则设置为 true
	closed bool
}

// NewFeatureGate 创建一个具有默认特性的新特性门控
func NewFeatureGate() MutableFeatureGate {
	known := map[Feature]FeatureSpec{}
	for k, v := range defaultFeatures {
		known[k] = v
	}

	f := &featureGate{}
	f.known.Store(known)
	f.enabled.Store(map[Feature]bool{})
	return f
}

// SetFromMap 从 map[string]bool 设置特性门控值
func (f *featureGate) SetFromMap(m map[string]bool) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	// 复制现有状态
	known := map[Feature]FeatureSpec{}
	for k, v := range f.known.Load().(map[Feature]FeatureSpec) {
		known[k] = v
	}
	enabled := map[Feature]bool{}
	for k, v := range f.enabled.Load().(map[Feature]bool) {
		enabled[k] = v
	}

	// 应用新设置
	for k, v := range m {
		k := Feature(k)
		featureSpec, ok := known[k]
		if !ok {
			return fmt.Errorf("无法识别的特性门控：%s", k)
		}
		if featureSpec.LockToDefault && featureSpec.Default != v {
			return fmt.Errorf("无法将特性门控 %v 设置为 %v，该特性已锁定为 %v", k, v, featureSpec.Default)
		}
		enabled[k] = v
	}

	// 持久化更改
	f.known.Store(known)
	f.enabled.Store(enabled)
	return nil
}

// Add 将特性添加到特性门控
func (f *featureGate) Add(features map[Feature]FeatureSpec) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.closed {
		return fmt.Errorf("特性门控关闭后无法添加特性门控")
	}

	// 复制现有状态
	known := map[Feature]FeatureSpec{}
	for k, v := range f.known.Load().(map[Feature]FeatureSpec) {
		known[k] = v
	}

	// 添加新特性
	for name, spec := range features {
		if existingSpec, found := known[name]; found {
			if existingSpec == spec {
				continue
			}
			return fmt.Errorf("具有不同规范的特性门控 %q 已存在：%v", name, existingSpec)
		}
		known[name] = spec
	}

	// 持久化更改
	f.known.Store(known)

	return nil
}

// String 返回包含所有已启用特性门控的字符串，格式为 "key1=value1,key2=value2,..."
func (f *featureGate) String() string {
	pairs := []string{}
	for k, v := range f.enabled.Load().(map[Feature]bool) {
		pairs = append(pairs, fmt.Sprintf("%s=%t", k, v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// Enabled 如果键已启用则返回 true
func (f *featureGate) Enabled(key Feature) bool {
	if v, ok := f.enabled.Load().(map[Feature]bool)[key]; ok {
		return v
	}
	if v, ok := f.known.Load().(map[Feature]FeatureSpec)[key]; ok {
		return v.Default
	}
	return false
}

// KnownFeatures 返回描述 FeatureGate 已知特性的字符串切片
// GA 特性从列表中隐藏
func (f *featureGate) KnownFeatures() []string {
	knownFeatures := f.known.Load().(map[Feature]FeatureSpec)
	known := make([]string, 0, len(knownFeatures))
	for k, v := range knownFeatures {
		if v.Stage == GA {
			continue
		}
		known = append(known, fmt.Sprintf("%s=true|false (%s - default=%t)", k, v.Stage, v.Default))
	}
	sort.Strings(known)
	return known
}

// 默认特性门控实例
var DefaultFeatureGates = NewFeatureGate()

// Enabled 检查默认特性门控中是否启用了某个特性
func Enabled(name Feature) bool {
	return DefaultFeatureGates.Enabled(name)
}

// SetFromMap 在默认特性门控中从 map 设置特性门控值
func SetFromMap(featureMap map[string]bool) error {
	return DefaultFeatureGates.SetFromMap(featureMap)
}
