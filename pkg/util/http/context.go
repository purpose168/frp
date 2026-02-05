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
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// Context 是 HTTP 请求上下文
type Context struct {
	// Req 是 HTTP 请求
	Req *http.Request
	// Resp 是 HTTP 响应写入器
	Resp http.ResponseWriter
	// vars 是路径变量映射
	vars map[string]string
}

// NewContext 创建一个新的 HTTP 请求上下文
// 参数 w 是 HTTP 响应写入器
// 参数 r 是 HTTP 请求
// 返回上下文实例
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Req:  r,
		Resp: w,
		vars: mux.Vars(r),
	}
}

// Param 获取路径参数
// 参数 key 是参数键
// 返回参数值
func (c *Context) Param(key string) string {
	return c.vars[key]
}

// Query 获取查询参数
// 参数 key 是查询参数键
// 返回查询参数值
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// BindJSON 将请求体绑定到指定的对象
// 参数 obj 是要绑定的对象
// 返回可能的错误
func (c *Context) BindJSON(obj any) error {
	body, err := io.ReadAll(c.Req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, obj)
}

// Body 获取请求体
// 返回请求体字节数组和可能的错误
func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.Req.Body)
}
