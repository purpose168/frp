// Copyright 2020 guylewin, guy@lewin.co.il
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

package tcpmux

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	libnet "github.com/fatedier/golib/net"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/vhost"
)

// HTTPConnectTCPMuxer HTTP CONNECT 方法 TCP 多路复用器
// 用于处理通过 HTTP CONNECT 方法建立的 TCP 连接
type HTTPConnectTCPMuxer struct {
	*vhost.Muxer

	// 如果 passthrough 设置为 true，CONNECT 请求将被转发到后端服务
	// 否则，它将向客户端返回 OK 响应，并将剩余内容转发到后端服务
	passthrough bool
}

// NewHTTPConnectTCPMuxer 创建一个新的 HTTP CONNECT TCP 多路复用器
// listener: 网络监听器，用于接收传入的连接
// passthrough: 是否透传 CONNECT 请求到后端服务
// timeout: 超时时间
func NewHTTPConnectTCPMuxer(listener net.Listener, passthrough bool, timeout time.Duration) (*HTTPConnectTCPMuxer, error) {
	// 创建多路复用器实例
	ret := &HTTPConnectTCPMuxer{passthrough: passthrough}
	// 创建底层多路复用器，使用 getHostFromHTTPConnect 作为主机名提取函数
	mux, err := vhost.NewMuxer(listener, ret.getHostFromHTTPConnect, timeout)
	// 设置认证函数
	mux.SetCheckAuthFunc(ret.auth).
		// 设置成功钩子函数，用于发送 CONNECT 响应
		SetSuccessHookFunc(ret.sendConnectResponse).
		// 设置失败钩子函数
		SetFailHookFunc(vhostFailed)
	ret.Muxer = mux
	return ret, err
}

// readHTTPConnectRequest 从读取器中读取 HTTP CONNECT 请求
// 返回主机名、HTTP 用户名、HTTP 密码和可能的错误
func (muxer *HTTPConnectTCPMuxer) readHTTPConnectRequest(rd io.Reader) (host, httpUser, httpPwd string, err error) {
	// 创建带缓冲的读取器
	bufioReader := bufio.NewReader(rd)

	// 读取 HTTP 请求
	req, err := http.ReadRequest(bufioReader)
	if err != nil {
		return
	}

	// 检查请求方法是否为 CONNECT
	if req.Method != "CONNECT" {
		err = fmt.Errorf("连接到 tcp 虚拟主机必须使用 CONNECT 方法")
		return
	}

	// 获取规范化的主机名
	host, _ = httppkg.CanonicalHost(req.Host)
	// 获取代理认证头
	proxyAuth := req.Header.Get("Proxy-Authorization")
	if proxyAuth != "" {
		// 解析基本认证信息
		httpUser, httpPwd, _ = httppkg.ParseBasicAuth(proxyAuth)
	}
	return
}

// sendConnectResponse 发送 CONNECT 响应
// 如果 passthrough 为 false，则发送 200 OK 响应给客户端
func (muxer *HTTPConnectTCPMuxer) sendConnectResponse(c net.Conn, _ map[string]string) error {
	// 如果设置为透传模式，则不发送响应
	if muxer.passthrough {
		return nil
	}
	// 创建 OK 响应
	res := httppkg.OkResponse()
	if res.Body != nil {
		defer res.Body.Close()
	}
	// 将响应写入连接
	return res.Write(c)
}

// auth 验证用户凭据
// 检查提供的用户名和密码是否与请求中的凭据匹配
func (muxer *HTTPConnectTCPMuxer) auth(c net.Conn, username, password string, reqInfo map[string]string) (bool, error) {
	// 从请求信息中获取用户名和密码
	reqUsername := reqInfo["HTTPUser"]
	reqPassword := reqInfo["HTTPPwd"]
	// 验证凭据是否匹配
	if username == reqUsername && password == reqPassword {
		return true, nil
	}

	// 创建未授权响应
	resp := httppkg.ProxyUnauthorizedResponse()
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	// 将未授权响应写入连接
	_ = resp.Write(c)
	return false, nil
}

// vhostFailed 处理虚拟主机失败的情况
// 发送 404 Not Found 响应并关闭连接
func vhostFailed(c net.Conn) {
	// 创建未找到响应
	res := vhost.NotFoundResponse()
	if res.Body != nil {
		defer res.Body.Close()
	}
	// 将响应写入连接
	_ = res.Write(c)
	// 关闭连接
	_ = c.Close()
}

// getHostFromHTTPConnect 从 HTTP CONNECT 请求中提取主机信息
// 返回连接、请求信息映射和可能的错误
func (muxer *HTTPConnectTCPMuxer) getHostFromHTTPConnect(c net.Conn) (net.Conn, map[string]string, error) {
	// 创建请求信息映射
	reqInfoMap := make(map[string]string, 0)
	// 创建共享连接和读取器
	sc, rd := libnet.NewSharedConn(c)

	// 读取 HTTP CONNECT 请求
	host, httpUser, httpPwd, err := muxer.readHTTPConnectRequest(rd)
	if err != nil {
		return nil, reqInfoMap, err
	}

	// 填充请求信息映射
	reqInfoMap["Host"] = host
	reqInfoMap["Scheme"] = "tcp"
	reqInfoMap["HTTPUser"] = httpUser
	reqInfoMap["HTTPPwd"] = httpPwd

	// 确定输出连接
	outConn := c
	// 如果设置为透传模式，使用共享连接
	if muxer.passthrough {
		outConn = sc
	}
	return outConn, reqInfoMap, nil
}
