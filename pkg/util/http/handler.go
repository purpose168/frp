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
	"net/http"

	"github.com/fatedier/frp/pkg/util/log"
)

// GeneralResponse 是通用响应结构
type GeneralResponse struct {
	// Code 是响应码
	Code int
	// Msg 是响应消息
	Msg string
}

// APIHandler 是一个处理函数，返回响应对象或错误。
type APIHandler func(ctx *Context) (any, error)

// MakeHTTPHandlerFunc 将普通的 APIHandler 转换为 http.HandlerFunc。
func MakeHTTPHandlerFunc(handler APIHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		res, err := handler(ctx)
		if err != nil {
			log.Warnf("http 响应 [%s]：错误：%v", r.URL.Path, err)
			code := http.StatusInternalServerError
			if e, ok := err.(*Error); ok {
				code = e.Code
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_ = json.NewEncoder(w).Encode(GeneralResponse{Code: code, Msg: err.Error()})
			return
		}

		if res == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch v := res.(type) {
		case []byte:
			_, _ = w.Write(v)
		case string:
			_, _ = w.Write([]byte(v))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(v)
		}
	}
}
