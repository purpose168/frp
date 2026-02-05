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

import (
	"net/http"

	"github.com/purpose168/frp/pkg/util/log"
)

// responseWriter 包装 http.ResponseWriter 以记录响应状态码
type responseWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader 写入响应状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

// NewRequestLogger 创建请求日志中间件
// 参数 next 是下一个处理器
// 返回 HTTP 处理器
func NewRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("http 请求: [%s]", r.URL.Path)
		rw := &responseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Infof("http 响应 [%s]: 状态码 [%d]", r.URL.Path, rw.code)
	})
}
