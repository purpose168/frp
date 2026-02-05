// Copyright 2023 The frp Authors
//
// Licensed under to Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with License.
// You may obtain a copy of License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nathole

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/stun/v2"
)

var responseTimeout = 3 * time.Second

// Message 消息
type Message struct {
	// Body 消息体
	Body []byte
	// Addr 地址
	Addr string
}

// Discover 发现外部地址
// 如果 localAddr 为空，则监听随机端口
func Discover(stunServers []string, localAddr string) ([]string, net.Addr, error) {
	// 创建 discoverConn 并从 messageChan 获取响应
	discoverConn, err := listen(localAddr)
	if err != nil {
		return nil, nil, err
	}
	defer discoverConn.Close()

	go discoverConn.readLoop()

	addresses := make([]string, 0, len(stunServers))
	for _, addr := range stunServers {
		// 从 STUN 服务器获取外部地址
		externalAddrs, err := discoverConn.discoverFromStunServer(addr)
		if err != nil {
			return nil, nil, err
		}
		addresses = append(addresses, externalAddrs...)
	}
	return addresses, discoverConn.localAddr, nil
}

// stunResponse STUN 响应
type stunResponse struct {
	// externalAddr 外部地址
	externalAddr string
	// otherAddr 其他地址
	otherAddr string
}

// discoverConn 发现连接
type discoverConn struct {
	// conn UDP 连接
	conn *net.UDPConn

	// localAddr 本地地址
	localAddr net.Addr
	// messageChan 消息通道
	messageChan chan *Message
}

// listen 监听
func listen(localAddr string) (*discoverConn, error) {
	var local *net.UDPAddr
	if localAddr != "" {
		addr, err := net.ResolveUDPAddr("udp4", localAddr)
		if err != nil {
			return nil, err
		}
		local = addr
	}
	conn, err := net.ListenUDP("udp4", local)
	if err != nil {
		return nil, err
	}

	return &discoverConn{
		conn:        conn,
		localAddr:   conn.LocalAddr(),
		messageChan: make(chan *Message, 10),
	}, nil
}

// Close 关闭连接
func (c *discoverConn) Close() error {
	if c.messageChan != nil {
		close(c.messageChan)
		c.messageChan = nil
	}
	return c.conn.Close()
}

// readLoop 读取循环
func (c *discoverConn) readLoop() {
	for {
		buf := make([]byte, 1024)
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		buf = buf[:n]

		c.messageChan <- &Message{
			Body: buf,
			Addr: addr.String(),
		}
	}
}

// doSTUNRequest 执行 STUN 请求
func (c *discoverConn) doSTUNRequest(addr string) (*stunResponse, error) {
	serverAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	request, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, err
	}

	if err = request.NewTransactionID(); err != nil {
		return nil, err
	}
	if _, err := c.conn.WriteTo(request.Raw, serverAddr); err != nil {
		return nil, err
	}

	var m stun.Message
	select {
	case msg := <-c.messageChan:
		m.Raw = msg.Body
		if err := m.Decode(); err != nil {
			return nil, err
		}
	case <-time.After(responseTimeout):
		return nil, fmt.Errorf("等待 STUN 服务器响应超时")
	}
	xorAddrGetter := &stun.XORMappedAddress{}
	mappedAddrGetter := &stun.MappedAddress{}
	changedAddrGetter := ChangedAddress{}
	otherAddrGetter := &stun.OtherAddress{}

	resp := &stunResponse{}
	if err := mappedAddrGetter.GetFrom(&m); err == nil {
		resp.externalAddr = mappedAddrGetter.String()
	}
	if err := xorAddrGetter.GetFrom(&m); err == nil {
		resp.externalAddr = xorAddrGetter.String()
	}
	if err := changedAddrGetter.GetFrom(&m); err == nil {
		resp.otherAddr = changedAddrGetter.String()
	}
	if err := otherAddrGetter.GetFrom(&m); err == nil {
		resp.otherAddr = otherAddrGetter.String()
	}
	return resp, nil
}

// discoverFromStunServer 从 STUN 服务器发现
func (c *discoverConn) discoverFromStunServer(addr string) ([]string, error) {
	resp, err := c.doSTUNRequest(addr)
	if err != nil {
		return nil, err
	}
	if resp.externalAddr == "" {
		return nil, fmt.Errorf("未找到外部地址")
	}

	externalAddrs := make([]string, 0, 2)
	externalAddrs = append(externalAddrs, resp.externalAddr)

	if resp.otherAddr == "" {
		return externalAddrs, nil
	}

	// 从变更地址查找外部地址
	resp, err = c.doSTUNRequest(resp.otherAddr)
	if err != nil {
		return nil, err
	}
	if resp.externalAddr != "" {
		externalAddrs = append(externalAddrs, resp.externalAddr)
	}
	return externalAddrs, nil
}
