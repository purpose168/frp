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

package limit

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// Reader 是限流读取器
type Reader struct {
	// r 是底层读取器
	r io.Reader
	// limiter 是速率限制器
	limiter *rate.Limiter
}

// NewReader 创建新的限流读取器
// 参数 r 是底层读取器
// 参数 limiter 是速率限制器
// 返回限流读取器实例
func NewReader(r io.Reader, limiter *rate.Limiter) *Reader {
	return &Reader{
		r:       r,
		limiter: limiter,
	}
}

// Read 从底层读取器读取数据，并应用速率限制
// 参数 p 是读取缓冲区
// 返回读取的字节数和可能的错误
func (r *Reader) Read(p []byte) (n int, err error) {
	b := r.limiter.Burst()
	if b < len(p) {
		p = p[:b]
	}
	n, err = r.r.Read(p)
	if err != nil {
		return
	}

	err = r.limiter.WaitN(context.Background(), n)
	if err != nil {
		return
	}
	return
}
