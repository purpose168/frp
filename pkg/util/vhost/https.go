// Copyright 2016 fatedier, fatedier@gmail.com
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
	"crypto/tls"
	"io"
	"net"
	"time"

	libnet "github.com/fatedier/golib/net"
)

// HTTPSMuxer HTTPS 多路复用器
type HTTPSMuxer struct {
	*Muxer
}

// NewHTTPSMuxer 创建一个新的 HTTPS 多路复用器
func NewHTTPSMuxer(listener net.Listener, timeout time.Duration) (*HTTPSMuxer, error) {
	// 创建多路复用器，使用 GetHTTPSHostname 函数提取主机名
	mux, err := NewMuxer(listener, GetHTTPSHostname, timeout)
	// 设置失败钩子函数
	mux.SetFailHookFunc(vhostFailed)
	if err != nil {
		return nil, err
	}
	return &HTTPSMuxer{mux}, err
}

// GetHTTPSHostname 从 TLS 连接中获取 HTTPS 主机名
func GetHTTPSHostname(c net.Conn) (_ net.Conn, _ map[string]string, err error) {
	// 创建请求信息映射
	reqInfoMap := make(map[string]string, 0)
	// 创建共享连接和读取器
	sc, rd := libnet.NewSharedConn(c)

	// 读取客户端 Hello 消息
	clientHello, err := readClientHello(rd)
	if err != nil {
		return nil, reqInfoMap, err
	}

	// 设置主机名和协议
	reqInfoMap["Host"] = clientHello.ServerName
	reqInfoMap["Scheme"] = "https"
	return sc, reqInfoMap, nil
}

// readClientHello 从读取器中读取客户端 Hello 消息
func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo

	// 注意：握手总是失败，因为 readOnlyConn 不是真正的连接
	// 只要成功读取客户端 Hello，失败应该只在调用 GetConfigForClient 之后发生
	// 所以我们只关心 hello 从未被设置时的错误
	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = &tls.ClientHelloInfo{}
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()

	if hello == nil {
		return nil, err
	}
	return hello, nil
}

// vhostFailed 处理虚拟主机失败的情况
func vhostFailed(c net.Conn) {
	// 发送 alertUnrecognizedName 警报
	_ = tls.Server(c, &tls.Config{}).Handshake()
	c.Close()
}

// readOnlyConn 只读连接
type readOnlyConn struct {
	reader io.Reader
}

// 只读连接的方法实现
func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(_ []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(_ time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(_ time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(_ time.Time) error { return nil }
