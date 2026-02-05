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

package vhost

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
)

// NotFoundPagePath 自定义 404 页面路径
var NotFoundPagePath = ""

// NotFound 默认的 404 页面内容
const (
	NotFound = `<!DOCTYPE html>
<html>
<head>
<title>未找到页面</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>未找到请求的页面</h1>
<p>很抱歉，您请求的页面当前不可用。<br/>
请稍后再试。</p>
<p>该服务器由 <a href="https://github.com/fatedier/frp">frp</a> 提供支持。</p>
<p><em>真诚的，frp。</em></p>
</body>
</html>
`
)

// getNotFoundPageContent 获取 404 页面内容
func getNotFoundPageContent() []byte {
	var (
		buf []byte
		err error
	)
	// 如果配置了自定义 404 页面路径，则读取自定义页面
	if NotFoundPagePath != "" {
		buf, err = os.ReadFile(NotFoundPagePath)
		if err != nil {
			log.Warnf("读取自定义 404 页面错误: %v", err)
			buf = []byte(NotFound)
		}
	} else {
		// 否则使用默认的 404 页面
		buf = []byte(NotFound)
	}
	return buf
}

// NotFoundResponse 创建 404 响应
func NotFoundResponse() *http.Response {
	// 创建响应头
	header := make(http.Header)
	header.Set("server", "frp/"+version.Full())
	header.Set("Content-Type", "text/html")

	// 获取 404 页面内容
	content := getNotFoundPageContent()
	// 创建响应
	res := &http.Response{
		Status:        "Not Found",
		StatusCode:    404,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
	}
	return res
}
