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

package udp

import (
	"encoding/base64"
	"net"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	"github.com/fatedier/golib/pool"

	"github.com/fatedier/frp/pkg/msg"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// NewUDPPacket 创建一个新的 UDP 数据包
// 参数 buf 是数据包内容
// 参数 laddr 是本地地址
// 参数 raddr 是远程地址
func NewUDPPacket(buf []byte, laddr, raddr *net.UDPAddr) *msg.UDPPacket {
	return &msg.UDPPacket{
		Content:    base64.StdEncoding.EncodeToString(buf),
		LocalAddr:  laddr,
		RemoteAddr: raddr,
	}
}

// GetContent 从 UDP 数据包中获取原始内容
// 参数 m 是 UDP 数据包消息
// 返回解码后的字节数组和可能的错误
func GetContent(m *msg.UDPPacket) (buf []byte, err error) {
	buf, err = base64.StdEncoding.DecodeString(m.Content)
	return
}

// ForwardUserConn 在用户 UDP 连接和消息通道之间转发数据
// 参数 udpConn 是用户 UDP 连接
// 参数 readCh 是读取 UDP 数据包的通道
// 参数 sendCh 是发送 UDP 数据包的通道
// 参数 bufSize 是缓冲区大小
func ForwardUserConn(udpConn *net.UDPConn, readCh <-chan *msg.UDPPacket, sendCh chan<- *msg.UDPPacket, bufSize int) {
	// 从 readCh 读取数据并写入 udpConn
	go func() {
		for udpMsg := range readCh {
			buf, err := GetContent(udpMsg)
			if err != nil {
				continue
			}
			_, _ = udpConn.WriteToUDP(buf, udpMsg.RemoteAddr)
		}
	}()

	// 从 udpConn 读取数据并发送到 sendCh
	buf := pool.GetBuf(bufSize)
	defer pool.PutBuf(buf)
	for {
		n, remoteAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		// buf[:n] 将被编码为字符串，因此字节可以被重用
		udpMsg := NewUDPPacket(buf[:n], nil, remoteAddr)

		select {
		case sendCh <- udpMsg:
		default:
		}
	}
}

// Forwarder 是一个 UDP 转发器，在目标地址和消息通道之间转发数据
// 参数 dstAddr 是目标 UDP 地址
// 参数 readCh 是读取 UDP 数据包的通道
// 参数 sendCh 是发送消息的通道
// 参数 bufSize 是缓冲区大小
// 参数 proxyProtocolVersion 是代理协议版本（可选）
func Forwarder(dstAddr *net.UDPAddr, readCh <-chan *msg.UDPPacket, sendCh chan<- msg.Message, bufSize int, proxyProtocolVersion string) {
	var mu sync.RWMutex
	udpConnMap := make(map[string]*net.UDPConn)

	// 从 dstAddr 读取数据并写入 sendCh
	writerFn := func(raddr *net.UDPAddr, udpConn *net.UDPConn) {
		addr := raddr.String()
		defer func() {
			mu.Lock()
			delete(udpConnMap, addr)
			mu.Unlock()
			udpConn.Close()
		}()

		buf := pool.GetBuf(bufSize)
		for {
			_ = udpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, _, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			udpMsg := NewUDPPacket(buf[:n], nil, raddr)
			if err = errors.PanicToError(func() {
				select {
				case sendCh <- udpMsg:
				default:
				}
			}); err != nil {
				return
			}
		}
	}

	// 从 readCh 读取数据
	go func() {
		for udpMsg := range readCh {
			buf, err := GetContent(udpMsg)
			if err != nil {
				continue
			}

			mu.Lock()
			udpConn, ok := udpConnMap[udpMsg.RemoteAddr.String()]
			if !ok {
				udpConn, err = net.DialUDP("udp", nil, dstAddr)
				if err != nil {
					mu.Unlock()
					continue
				}
				udpConnMap[udpMsg.RemoteAddr.String()] = udpConn
			}
			mu.Unlock()

			// 如果配置了代理协议，则添加代理协议头部（仅针对新连接的第一个数据包）
			if !ok && proxyProtocolVersion != "" && udpMsg.RemoteAddr != nil {
				ppBuf, err := netpkg.BuildProxyProtocolHeader(udpMsg.RemoteAddr, dstAddr, proxyProtocolVersion)
				if err == nil {
					// 将代理协议头部添加到 UDP 负载前面
					finalBuf := make([]byte, len(ppBuf)+len(buf))
					copy(finalBuf, ppBuf)
					copy(finalBuf[len(ppBuf):], buf)
					buf = finalBuf
				}
			}

			_, err = udpConn.Write(buf)
			if err != nil {
				udpConn.Close()
			}

			if !ok {
				go writerFn(udpMsg.RemoteAddr, udpConn)
			}
		}
	}()
}
