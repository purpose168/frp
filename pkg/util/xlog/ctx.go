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
	"context"
)

// key 用于在 context 中存储日志记录器的键类型
type key int

const (
	xlogKey key = 0 // 日志记录器在 context 中的键
)

// NewContext 创建一个新的 context，并将日志记录器存储在其中
func NewContext(ctx context.Context, xl *Logger) context.Context {
	return context.WithValue(ctx, xlogKey, xl)
}

// FromContext 从 context 中获取日志记录器
// 返回日志记录器和一个布尔值，表示是否成功获取
func FromContext(ctx context.Context) (xl *Logger, ok bool) {
	xl, ok = ctx.Value(xlogKey).(*Logger)
	return
}

// FromContextSafe 从 context 中安全地获取日志记录器
// 如果 context 中没有日志记录器，则创建一个新的日志记录器并返回
func FromContextSafe(ctx context.Context) *Logger {
	xl, ok := ctx.Value(xlogKey).(*Logger)
	if !ok {
		xl = New()
	}
	return xl
}
