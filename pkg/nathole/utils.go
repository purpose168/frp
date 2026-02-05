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
	"bytes"
	"fmt"
	"net"
	"strconv"

	"github.com/fatedier/golib/crypto"
	"github.com/pion/stun/v2"

	"github.com/fatedier/frp/pkg/msg"
)

// EncodeMessage 编码消息
func EncodeMessage(m msg.Message, key []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := msg.WriteMsg(buffer, m); err != nil {
		return nil, err
	}

	buf, err := crypto.Encode(buffer.Bytes(), key)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// DecodeMessageInto 解码消息到指定对象
func DecodeMessageInto(data, key []byte, m msg.Message) error {
	buf, err := crypto.Decode(data, key)
	if err != nil {
		return err
	}

	return msg.ReadMsgInto(bytes.NewReader(buf), m)
}

// ChangedAddress 变更地址
type ChangedAddress struct {
	// IP IP 地址
	IP net.IP
	// Port 端口
	Port int
}

// GetFrom 从 STUN 消息中获取变更地址
func (s *ChangedAddress) GetFrom(m *stun.Message) error {
	a := (*stun.MappedAddress)(s)
	return a.GetFromAs(m, stun.AttrChangedAddress)
}

// String 返回字符串表示
func (s *ChangedAddress) String() string {
	return net.JoinHostPort(s.IP.String(), strconv.Itoa(s.Port))
}

// ListAllLocalIPs 列出所有本地 IP 地址
func ListAllLocalIPs() ([]net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

// ListLocalIPsForNatHole 列出用于 NAT 穿透的本地 IP 地址
func ListLocalIPsForNatHole(maxItems int) ([]string, error) {
	if maxItems <= 0 {
		return nil, fmt.Errorf("maxItems 必须大于 0")
	}

	ips, err := ListAllLocalIPs()
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, maxItems)
	for _, ip := range ips {
		if len(filtered) >= maxItems {
			break
		}

		// 忽略 IPv6 地址
		if ip.To4() == nil {
			continue
		}
		// 忽略本地回环 IP
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}

		filtered = append(filtered, ip.String())
	}
	return filtered, nil
}
