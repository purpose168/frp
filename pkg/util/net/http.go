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

package net

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/util/util"
)

// HTTPAuthMiddleware 是 HTTP 认证中间件
type HTTPAuthMiddleware struct {
	// user 是用户名
	user string
	// passwd 是密码
	passwd string
	// authFailDelay 是认证失败延迟时间
	authFailDelay time.Duration
}

// NewHTTPAuthMiddleware 创建 HTTP 认证中间件
// 参数 user 是用户名
// 参数 passwd 是密码
// 返回 HTTP 认证中间件实例
func NewHTTPAuthMiddleware(user, passwd string) *HTTPAuthMiddleware {
	return &HTTPAuthMiddleware{
		user:   user,
		passwd: passwd,
	}
}

// SetAuthFailDelay 设置认证失败延迟时间
// 参数 delay 是延迟时间
// 返回中间件实例
func (authMid *HTTPAuthMiddleware) SetAuthFailDelay(delay time.Duration) *HTTPAuthMiddleware {
	authMid.authFailDelay = delay
	return authMid
}

// Middleware 返回 HTTP 处理器
// 参数 next 是下一个处理器
// 返回 HTTP 处理器
func (authMid *HTTPAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqUser, reqPasswd, hasAuth := r.BasicAuth()
		if (authMid.user == "" && authMid.passwd == "") ||
			(hasAuth && util.ConstantTimeEqString(reqUser, authMid.user) &&
				util.ConstantTimeEqString(reqPasswd, authMid.passwd)) {
			next.ServeHTTP(w, r)
		} else {
			if authMid.authFailDelay > 0 {
				time.Sleep(authMid.authFailDelay)
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}

// HTTPGzipWrapper 是 HTTP Gzip 包装器
type HTTPGzipWrapper struct {
	// h 是 HTTP 处理器
	h http.Handler
}

// ServeHTTP 处理 HTTP 请求，支持 Gzip 压缩
// 参数 w 是响应写入器
// 参数 r 是 HTTP 请求
func (gw *HTTPGzipWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gw.h.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	gw.h.ServeHTTP(gzr, r)
}

// MakeHTTPGzipHandler 创建 HTTP Gzip 处理器
// 参数 h 是 HTTP 处理器
// 返回 Gzip 包装的 HTTP 处理器
func MakeHTTPGzipHandler(h http.Handler) http.Handler {
	return &HTTPGzipWrapper{
		h: h,
	}
}

// gzipResponseWriter 是 Gzip 响应写入器
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Write 写入数据
// 参数 b 是要写入的数据
// 返回写入的字节数和可能的错误
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
