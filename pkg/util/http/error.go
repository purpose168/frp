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

package http

import "fmt"

// Error 是 HTTP 错误
type Error struct {
	// Code 是 HTTP 状态码
	Code int
	// Err 是错误信息
	Err error
}

// Error 返回错误字符串
func (e *Error) Error() string {
	return e.Err.Error()
}

// NewError 创建一个新的 HTTP 错误
// 参数 code 是 HTTP 状态码
// 参数 msg 是错误消息
// 返回错误实例
func NewError(code int, msg string) *Error {
	return &Error{
		Code: code,
		Err:  fmt.Errorf("%s", msg),
	}
}
