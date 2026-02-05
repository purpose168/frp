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
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fatedier/golib/pool"
	"golang.org/x/net/ipv4"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/transport"
	"github.com/purpose168/frp/pkg/util/xlog"
)

var (
	// mode 0：简单检测模式，通常用于双方都是 EasyNAT 或 HardNAT & EasyNAT（公网网络）
	// a. 接收方发送低 TTL 的检测消息
	// b. 发送方向接收方发送正常检测消息
	// c. 接收方接收检测消息并向发送方回送消息
	//
	// mode 1：用于 HardNAT & EasyNAT，向多个猜测端口发送检测消息
	// 通常适用于端口变化规律的场景
	// 大部分步骤与 mode 0 相同，但 EasyNAT 固定为接收方，将向发送方的多个猜测端口
	// 发送低 TTL 的检测消息
	//
	// mode 2：用于 HardNAT & EasyNAT，端口变化不规律
	// a. HardNAT 机器监听多个端口并向 EasyNAT 机器发送低 TTL 的检测消息
	// b. EasyNAT 机器向 HardNAT 机器的随机端口发送检测消息
	//
	// mode 3：用于 HardNAT & HardNAT，双方的端口变化都是规律的
	// 大部分步骤与 mode 1 相同，但发送方也需要向接收方的多个猜测
	// 端口发送检测消息
	//
	// mode 4：用于 HardNAT & HardNAT，其中一个端口变化是规律的
	// 规律端口变化通常在发送方
	// a. 接收方监听多个端口并向发送方的猜测范围端口发送低 TTL 的检测消息
	// b. 发送方向接收方的随机端口发送检测消息
	SupportedModes = []int{DetectMode0, DetectMode1, DetectMode2, DetectMode3, DetectMode4}
	SupportedRoles = []string{DetectRoleSender, DetectRoleReceiver}

	DetectMode0        = 0
	DetectMode1        = 1
	DetectMode2        = 2
	DetectMode3        = 3
	DetectMode4        = 4
	DetectRoleSender   = "sender"
	DetectRoleReceiver = "receiver"
)

// PrepareOptions 定义 NAT 穿透准备选项
type PrepareOptions struct {
	// DisableAssistedAddrs 禁用本地网络接口用于 NAT 穿透期间的辅助连接
	DisableAssistedAddrs bool
}

// PrepareResult 准备结果
type PrepareResult struct {
	// Addrs 地址列表
	Addrs []string
	// AssistedAddrs 辅助地址列表
	AssistedAddrs []string
	// ListenConn 监听连接
	ListenConn *net.UDPConn
	// NatType NAT 类型
	NatType string
	// Behavior 行为
	Behavior string
}

// PreCheck 用于检查代理是否准备好进行穿透
// 在调用 Prepare 之前调用此函数以避免不必要的准备工作
func PreCheck(
	ctx context.Context, transporter transport.MessageTransporter,
	proxyName string, timeout time.Duration,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var natHoleRespMsg *msg.NatHoleResp
	transactionID := NewTransactionID()
	m, err := transporter.Do(timeoutCtx, &msg.NatHoleVisitor{
		TransactionID: transactionID,
		ProxyName:     proxyName,
		PreCheck:      true,
	}, transactionID, msg.TypeNameNatHoleResp)
	if err != nil {
		return fmt.Errorf("获取 natHoleRespMsg 错误: %v", err)
	}
	mm, ok := m.(*msg.NatHoleResp)
	if !ok {
		return fmt.Errorf("获取 natHoleRespMsg 错误: 无效的消息类型")
	}
	natHoleRespMsg = mm

	if natHoleRespMsg.Error != "" {
		return fmt.Errorf("%s", natHoleRespMsg.Error)
	}
	return nil
}

// Prepare 用于在穿透之前做一些准备工作
func Prepare(stunServers []string, opts PrepareOptions) (*PrepareResult, error) {
	// 发现 NAT 类型
	addrs, localAddr, err := Discover(stunServers, "")
	if err != nil {
		return nil, fmt.Errorf("发现错误: %v", err)
	}
	if len(addrs) < 2 {
		return nil, fmt.Errorf("发现错误: 地址数量不足")
	}

	localIPs, _ := ListLocalIPsForNatHole(10)
	natFeature, err := ClassifyNATFeature(addrs, localIPs)
	if err != nil {
		return nil, fmt.Errorf("分类 NAT 特征错误: %v", err)
	}

	laddr, err := net.ResolveUDPAddr("udp4", localAddr.String())
	if err != nil {
		return nil, fmt.Errorf("解析本地 UDP 地址错误: %v", err)
	}
	listenConn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, fmt.Errorf("监听本地 UDP 地址错误: %v", err)
	}

	// 应用 NAT 穿透选项
	var assistedAddrs []string
	if !opts.DisableAssistedAddrs {
		assistedAddrs = make([]string, 0, len(localIPs))
		for _, ip := range localIPs {
			assistedAddrs = append(assistedAddrs, net.JoinHostPort(ip, strconv.Itoa(laddr.Port)))
		}
	}
	return &PrepareResult{
		Addrs:         addrs,
		AssistedAddrs: assistedAddrs,
		ListenConn:    listenConn,
		NatType:       natFeature.NatType,
		Behavior:      natFeature.Behavior,
	}, nil
}

// ExchangeInfo 用于在客户端和访问者之间交换信息
// 1. 通过 msgTransporter 向服务器发送输入消息
// 2. 服务器将从客户端和访问者收集信息并分析。然后向它们发送 NatHoleResp 消息以告诉它们下一步如何操作
// 3. 从服务器接收 NatHoleResp 消息
func ExchangeInfo(
	ctx context.Context, transporter transport.MessageTransporter,
	laneKey string, m msg.Message, timeout time.Duration,
) (*msg.NatHoleResp, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var natHoleRespMsg *msg.NatHoleResp
	m, err := transporter.Do(timeoutCtx, m, laneKey, msg.TypeNameNatHoleResp)
	if err != nil {
		return nil, fmt.Errorf("获取 natHoleRespMsg 错误: %v", err)
	}
	mm, ok := m.(*msg.NatHoleResp)
	if !ok {
		return nil, fmt.Errorf("获取 natHoleRespMsg 错误: 无效的消息类型")
	}
	natHoleRespMsg = mm

	if natHoleRespMsg.Error != "" {
		return nil, fmt.Errorf("natHoleRespMsg 获取错误信息: %s", natHoleRespMsg.Error)
	}
	if len(natHoleRespMsg.CandidateAddrs) == 0 {
		return nil, fmt.Errorf("natHoleRespMsg 获取空候选地址")
	}
	return natHoleRespMsg, nil
}

// MakeHole 用于在客户端和访问者之间进行 NAT 打洞
func MakeHole(ctx context.Context, listenConn *net.UDPConn, m *msg.NatHoleResp, key []byte) (*net.UDPConn, *net.UDPAddr, error) {
	xl := xlog.FromContextSafe(ctx)
	transactionID := NewTransactionID()
	sendToRangePortsFunc := func(conn *net.UDPConn, addr string) error {
		return sendSidMessage(ctx, conn, m.Sid, transactionID, addr, key, m.DetectBehavior.TTL)
	}

	listenConns := []*net.UDPConn{listenConn}
	var detectAddrs []string
	if m.DetectBehavior.Role == DetectRoleSender {
		// 发送方
		if m.DetectBehavior.SendDelayMs > 0 {
			time.Sleep(time.Duration(m.DetectBehavior.SendDelayMs) * time.Millisecond)
		}
		detectAddrs = m.AssistedAddrs
		detectAddrs = append(detectAddrs, m.CandidateAddrs...)
	} else {
		// 接收方
		if len(m.DetectBehavior.CandidatePorts) == 0 {
			detectAddrs = m.CandidateAddrs
		}

		if m.DetectBehavior.ListenRandomPorts > 0 {
			for i := 0; i < m.DetectBehavior.ListenRandomPorts; i++ {
				tmpConn, err := net.ListenUDP("udp4", nil)
				if err != nil {
					xl.Warnf("监听随机 UDP 地址错误: %v", err)
					continue
				}
				listenConns = append(listenConns, tmpConn)
			}
		}
	}

	detectAddrs = slices.Compact(detectAddrs)
	for _, detectAddr := range detectAddrs {
		for _, conn := range listenConns {
			if err := sendSidMessage(ctx, conn, m.Sid, transactionID, detectAddr, key, m.DetectBehavior.TTL); err != nil {
				xl.Tracef("从 %s 向 %s 发送 sid 消息错误: %v", conn.LocalAddr(), detectAddr, err)
			}
		}
	}
	if len(m.DetectBehavior.CandidatePorts) > 0 {
		for _, conn := range listenConns {
			sendSidMessageToRangePorts(ctx, conn, m.CandidateAddrs, m.DetectBehavior.CandidatePorts, sendToRangePortsFunc)
		}
	}
	if m.DetectBehavior.SendRandomPorts > 0 {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for i := range listenConns {
			go sendSidMessageToRandomPorts(ctx, listenConns[i], m.CandidateAddrs, m.DetectBehavior.SendRandomPorts, sendToRangePortsFunc)
		}
	}

	timeout := 5 * time.Second
	if m.DetectBehavior.ReadTimeoutMs > 0 {
		timeout = time.Duration(m.DetectBehavior.ReadTimeoutMs) * time.Millisecond
	}

	if len(listenConns) == 1 {
		raddr, err := waitDetectMessage(ctx, listenConns[0], m.Sid, key, timeout, m.DetectBehavior.Role)
		if err != nil {
			return nil, nil, fmt.Errorf("等待检测消息错误: %v", err)
		}
		return listenConns[0], raddr, nil
	}

	type result struct {
		lConn *net.UDPConn
		raddr *net.UDPAddr
	}
	resultCh := make(chan result)
	for _, conn := range listenConns {
		go func(lConn *net.UDPConn) {
			addr, err := waitDetectMessage(ctx, lConn, m.Sid, key, timeout, m.DetectBehavior.Role)
			if err != nil {
				lConn.Close()
				return
			}
			select {
			case resultCh <- result{lConn: lConn, raddr: addr}:
			default:
				lConn.Close()
			}
		}(conn)
	}

	select {
	case result := <-resultCh:
		return result.lConn, result.raddr, nil
	case <-time.After(timeout):
		return nil, nil, fmt.Errorf("等待检测消息超时")
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("等待检测消息已取消")
	}
}

// waitDetectMessage 等待检测消息
func waitDetectMessage(
	ctx context.Context, conn *net.UDPConn, sid string, key []byte,
	timeout time.Duration, role string,
) (*net.UDPAddr, error) {
	xl := xlog.FromContextSafe(ctx)
	for {
		buf := pool.GetBuf(1024)
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		n, raddr, err := conn.ReadFromUDP(buf)
		_ = conn.SetReadDeadline(time.Time{})
		if err != nil {
			return nil, err
		}
		xl.Debugf("获取 UDP 消息本地 %s，来自 %s", conn.LocalAddr(), raddr)
		var m msg.NatHoleSid
		if err := DecodeMessageInto(buf[:n], key, &m); err != nil {
			xl.Warnf("解码 sid 消息错误: %v", err)
			continue
		}
		pool.PutBuf(buf)

		if m.Sid != sid {
			xl.Warnf("获取到错误 sid 的 sid 消息: %s，期望: %s", m.Sid, sid)
			continue
		}

		if !m.Response {
			// 如果我们是发送方，只等待响应消息
			if role == DetectRoleSender {
				continue
			}

			m.Response = true
			buf2, err := EncodeMessage(&m, key)
			if err != nil {
				xl.Warnf("编码 sid 消息错误: %v", err)
				continue
			}
			_, _ = conn.WriteToUDP(buf2, raddr)
		}
		return raddr, nil
	}
}

// sendSidMessage 发送 sid 消息
func sendSidMessage(
	ctx context.Context, conn *net.UDPConn,
	sid string, transactionID string, addr string, key []byte, ttl int,
) error {
	xl := xlog.FromContextSafe(ctx)
	ttlStr := ""
	if ttl > 0 {
		ttlStr = fmt.Sprintf("，TTL %d", ttl)
	}
	xl.Tracef("从 %s 向 %s 发送 sid 消息%s", conn.LocalAddr(), addr, ttlStr)
	raddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	if transactionID == "" {
		transactionID = NewTransactionID()
	}
	m := &msg.NatHoleSid{
		TransactionID: transactionID,
		Sid:           sid,
		Response:      false,
		Nonce:         strings.Repeat("0", rand.IntN(20)),
	}
	buf, err := EncodeMessage(m, key)
	if err != nil {
		return err
	}
	if ttl > 0 {
		uConn := ipv4.NewConn(conn)
		original, err := uConn.TTL()
		if err != nil {
			xl.Tracef("获取 TTL 错误 %v", err)
			return err
		}
		xl.Tracef("原始 TTL %d", original)

		err = uConn.SetTTL(ttl)
		if err != nil {
			xl.Tracef("设置 TTL 错误 %v", err)
		} else {
			defer func() {
				_ = uConn.SetTTL(original)
			}()
		}
	}

	if _, err := conn.WriteToUDP(buf, raddr); err != nil {
		return err
	}
	return nil
}

// sendSidMessageToRangePorts 向端口范围发送 sid 消息
func sendSidMessageToRangePorts(
	ctx context.Context, conn *net.UDPConn, addrs []string, ports []msg.PortsRange,
	sendFunc func(*net.UDPConn, string) error,
) {
	xl := xlog.FromContextSafe(ctx)
	for _, ip := range slices.Compact(parseIPs(addrs)) {
		for _, portsRange := range ports {
			for i := portsRange.From; i <= portsRange.To; i++ {
				detectAddr := net.JoinHostPort(ip, strconv.Itoa(i))
				if err := sendFunc(conn, detectAddr); err != nil {
					xl.Tracef("从 %s 向 %s 发送 sid 消息错误: %v", conn.LocalAddr(), detectAddr, err)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}
	}
}

// sendSidMessageToRandomPorts 向随机端口发送 sid 消息
func sendSidMessageToRandomPorts(
	ctx context.Context, conn *net.UDPConn, addrs []string, count int,
	sendFunc func(*net.UDPConn, string) error,
) {
	xl := xlog.FromContextSafe(ctx)
	used := sets.New[int]()
	getUnusedPort := func() int {
		for i := 0; i < 10; i++ {
			port := rand.IntN(65535-1024) + 1024
			if !used.Has(port) {
				used.Insert(port)
				return port
			}
		}
		return 0
	}

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		port := getUnusedPort()
		if port == 0 {
			continue
		}

		for _, ip := range slices.Compact(parseIPs(addrs)) {
			detectAddr := net.JoinHostPort(ip, strconv.Itoa(port))
			if err := sendFunc(conn, detectAddr); err != nil {
				xl.Tracef("从 %s 向 %s 发送 sid 消息错误: %v", conn.LocalAddr(), detectAddr, err)
			}
			time.Sleep(time.Millisecond * 15)
		}
	}
}

// parseIPs 解析 IP 地址
func parseIPs(addrs []string) []string {
	var ips []string
	for _, addr := range addrs {
		if ip, _, err := net.SplitHostPort(addr); err == nil {
			ips = append(ips, ip)
		}
	}
	return ips
}
