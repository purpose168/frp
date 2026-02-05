// Copyright 2019 fatedier, fatedier@gmail.com
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

package xlog

import (
	"cmp"
	"slices"

	"github.com/fatedier/frp/pkg/util/log"
)

// LogPrefix 表示日志前缀信息
type LogPrefix struct {
	// Name 是前缀的名称，不会在日志中显示，但用于标识前缀。
	Name string
	// Value 是前缀的值，会在日志中显示。
	Value string
	// 优先级越高的前缀会先显示，默认为 10。
	Priority int
}

// Logger 日志记录器，对于前缀操作不是线程安全的
type Logger struct {
	prefixes     []LogPrefix // 前缀列表
	prefixString string      // 渲染后的前缀字符串
}

// New 创建一个新的日志记录器
// 返回新创建的 Logger 实例
func New() *Logger {
	return &Logger{
		prefixes: make([]LogPrefix, 0),
	}
}

// ResetPrefixes 重置日志记录器的前缀
// 返回旧的前缀列表
func (l *Logger) ResetPrefixes() (old []LogPrefix) {
	old = l.prefixes
	l.prefixes = make([]LogPrefix, 0)
	l.prefixString = ""
	return
}

// AppendPrefix 追加一个简单的前缀
// prefix: 前缀字符串
// 返回日志记录器实例，支持链式调用
func (l *Logger) AppendPrefix(prefix string) *Logger {
	return l.AddPrefix(LogPrefix{
		Name:     prefix,
		Value:    prefix,
		Priority: 10,
	})
}

// AddPrefix 添加一个带名称、值和优先级的前缀
// prefix: 前缀信息
// 返回日志记录器实例，支持链式调用
func (l *Logger) AddPrefix(prefix LogPrefix) *Logger {
	found := false
	// 如果优先级小于等于 0，设置为默认值 10
	if prefix.Priority <= 0 {
		prefix.Priority = 10
	}
	// 检查是否已存在同名前缀，如果存在则更新值和优先级
	for _, p := range l.prefixes {
		if p.Name == prefix.Name {
			found = true
			p.Value = prefix.Value
			p.Priority = prefix.Priority
		}
	}
	// 如果不存在同名前缀，添加到前缀列表
	if !found {
		l.prefixes = append(l.prefixes, prefix)
	}
	// 重新渲染前缀字符串
	l.renderPrefixString()
	return l
}

// renderPrefixString 渲染前缀字符串
// 根据前缀的优先级排序，然后拼接成前缀字符串
func (l *Logger) renderPrefixString() {
	// 按照优先级排序前缀
	slices.SortStableFunc(l.prefixes, func(a, b LogPrefix) int {
		return cmp.Compare(a.Priority, b.Priority)
	})
	// 重置前缀字符串
	l.prefixString = ""
	// 拼接所有前缀
	for _, v := range l.prefixes {
		l.prefixString += "[" + v.Value + "] "
	}
}

// Spawn 创建一个新的日志记录器，继承当前记录器的所有前缀
// 返回新创建的 Logger 实例
func (l *Logger) Spawn() *Logger {
	nl := New()
	nl.prefixes = append(nl.prefixes, l.prefixes...)
	nl.renderPrefixString()
	return nl
}

// Errorf 记录 Error 级别的日志
// format: 格式化字符串
// v: 格式化参数
func (l *Logger) Errorf(format string, v ...any) {
	log.Logger.Errorf(l.prefixString+format, v...)
}

// Warnf 记录 Warn 级别的日志
// format: 格式化字符串
// v: 格式化参数
func (l *Logger) Warnf(format string, v ...any) {
	log.Logger.Warnf(l.prefixString+format, v...)
}

// Infof 记录 Info 级别的日志
// format: 格式化字符串
// v: 格式化参数
func (l *Logger) Infof(format string, v ...any) {
	log.Logger.Infof(l.prefixString+format, v...)
}

// Debugf 记录 Debug 级别的日志
// format: 格式化字符串
// v: 格式化参数
func (l *Logger) Debugf(format string, v ...any) {
	log.Logger.Debugf(l.prefixString+format, v...)
}

// Tracef 记录 Trace 级别的日志
// format: 格式化字符串
// v: 格式化参数
func (l *Logger) Tracef(format string, v ...any) {
	log.Logger.Tracef(l.prefixString+format, v...)
}
