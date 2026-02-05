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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/util"
)

// NatHoleTimeout NAT 穿透超时时间（秒）
var NatHoleTimeout int64 = 10

// NewTransactionID 创建新的事务 ID
func NewTransactionID() string {
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", time.Now().Unix(), id)
}

// ClientCfg 客户端配置
type ClientCfg struct {
	// name 名称
	name string
	// sk 密钥
	sk string
	// allowUsers 允许的用户列表
	allowUsers []string
	// sidCh 会话 ID 通道
	sidCh chan string
}

// Session 会话
type Session struct {
	// sid 会话 ID
	sid string
	// analysisKey 分析键
	analysisKey string
	// recommandMode 推荐模式
	recommandMode int
	// recommandIndex 推荐索引
	recommandIndex int

	// visitorMsg 访问者消息
	visitorMsg *msg.NatHoleVisitor
	// visitorTransporter 访问者传输器
	visitorTransporter transport.MessageTransporter
	// vResp 访问者响应
	vResp *msg.NatHoleResp
	// vNatFeature 访问者 NAT 特征
	vNatFeature *NatFeature
	// vBehavior 访问者行为
	vBehavior RecommandBehavior

	// clientMsg 客户端消息
	clientMsg *msg.NatHoleClient
	// clientTransporter 客户端传输器
	clientTransporter transport.MessageTransporter
	// cResp 客户端响应
	cResp *msg.NatHoleResp
	// cNatFeature 客户端 NAT 特征
	cNatFeature *NatFeature
	// cBehavior 客户端行为
	cBehavior RecommandBehavior

	// notifyCh 通知通道
	notifyCh chan struct{}
}

// genAnalysisKey 生成分析键
func (s *Session) genAnalysisKey() {
	hash := md5.New()
	vIPs := slices.Compact(parseIPs(s.visitorMsg.MappedAddrs))
	if len(vIPs) > 0 {
		hash.Write([]byte(vIPs[0]))
	}
	hash.Write([]byte(s.vNatFeature.NatType))
	hash.Write([]byte(s.vNatFeature.Behavior))
	hash.Write([]byte(strconv.FormatBool(s.vNatFeature.RegularPortsChange)))

	cIPs := slices.Compact(parseIPs(s.clientMsg.MappedAddrs))
	if len(cIPs) > 0 {
		hash.Write([]byte(cIPs[0]))
	}
	hash.Write([]byte(s.cNatFeature.NatType))
	hash.Write([]byte(s.cNatFeature.Behavior))
	hash.Write([]byte(strconv.FormatBool(s.cNatFeature.RegularPortsChange)))
	s.analysisKey = hex.EncodeToString(hash.Sum(nil))
}

// Controller 控制器
type Controller struct {
	// clientCfgs 客户端配置映射
	clientCfgs map[string]*ClientCfg
	// sessions 会话映射
	sessions map[string]*Session
	// analyzer 分析器
	analyzer *Analyzer

	mu sync.RWMutex
}

// NewController 创建新的控制器
func NewController(analysisDataReserveDuration time.Duration) (*Controller, error) {
	return &Controller{
		clientCfgs: make(map[string]*ClientCfg),
		sessions:   make(map[string]*Session),
		analyzer:   NewAnalyzer(analysisDataReserveDuration),
	}, nil
}

// CleanWorker 清理工作协程
func (c *Controller) CleanWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			count, total := c.analyzer.Clean()
			log.Tracef("清理 %d/%d NAT 穿透分析数据，耗时 %v", count, total, time.Since(start))
		case <-ctx.Done():
			return
		}
	}
}

// ListenClient 监听客户端
func (c *Controller) ListenClient(name string, sk string, allowUsers []string) (chan string, error) {
	cfg := &ClientCfg{
		name:       name,
		sk:         sk,
		allowUsers: allowUsers,
		sidCh:      make(chan string),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.clientCfgs[name]; ok {
		return nil, fmt.Errorf("代理 [%s] 重复", name)
	}
	c.clientCfgs[name] = cfg
	return cfg.sidCh, nil
}

// CloseClient 关闭客户端
func (c *Controller) CloseClient(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clientCfgs, name)
}

// GenSid 生成会话 ID
func (c *Controller) GenSid() string {
	t := time.Now().Unix()
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", t, id)
}

// HandleVisitor 处理访问者
func (c *Controller) HandleVisitor(m *msg.NatHoleVisitor, transporter transport.MessageTransporter, visitorUser string) {
	if m.PreCheck {
		cfg, ok := c.clientCfgs[m.ProxyName]
		if !ok {
			_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, fmt.Sprintf("[%s] 的 xtcp 服务器不存在", m.ProxyName)))
			return
		}
		if !slices.Contains(cfg.allowUsers, visitorUser) && !slices.Contains(cfg.allowUsers, "*") {
			_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, fmt.Sprintf("xtcp 访问者用户 [%s] 不允许访问 [%s]", visitorUser, m.ProxyName)))
			return
		}
		_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, ""))
		return
	}

	sid := c.GenSid()
	session := &Session{
		sid:                sid,
		visitorMsg:         m,
		visitorTransporter: transporter,
		notifyCh:           make(chan struct{}, 1),
	}
	var (
		clientCfg *ClientCfg
		ok        bool
	)
	err := func() error {
		c.mu.Lock()
		defer c.mu.Unlock()

		clientCfg, ok = c.clientCfgs[m.ProxyName]
		if !ok {
			return fmt.Errorf("[%s] 的 xtcp 服务器不存在", m.ProxyName)
		}
		if !util.ConstantTimeEqString(m.SignKey, util.GetAuthKey(clientCfg.sk, m.Timestamp)) {
			return fmt.Errorf("[%s] 的 xtcp 连接身份验证失败", m.ProxyName)
		}
		c.sessions[sid] = session
		return nil
	}()
	if err != nil {
		log.Warnf("处理访问者消息错误: %v", err)
		_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, err.Error()))
		return
	}
	log.Tracef("处理访问者消息，sid [%s]，服务器名称: %s", sid, m.ProxyName)

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.sessions, sid)
	}()

	if err := errors.PanicToError(func() {
		clientCfg.sidCh <- sid
	}); err != nil {
		return
	}

	// 等待 NatHoleClient 消息
	select {
	case <-session.notifyCh:
	case <-time.After(time.Duration(NatHoleTimeout) * time.Second):
		log.Debugf("等待 NatHoleClient 消息超时，sid [%s]", sid)
		return
	}

	// 根据客户端和访问者的 NAT 信息进行打洞决策
	vResp, cResp, err := c.analysis(session)
	if err != nil {
		log.Debugf("sid [%s] 分析错误: %v", err)
		vResp = c.GenNatHoleResponse(session.visitorMsg.TransactionID, nil, err.Error())
		cResp = c.GenNatHoleResponse(session.clientMsg.TransactionID, nil, err.Error())
	}
	session.cResp = cResp
	session.vResp = vResp

	// 向访问者和客户端发送响应
	var g errgroup.Group
	g.Go(func() error {
		// 如果是发送方，等待一段时间以确保客户端已发送检测消息
		if vResp.DetectBehavior.Role == "sender" {
			time.Sleep(1 * time.Second)
		}
		_ = session.visitorTransporter.Send(vResp)
		return nil
	})
	g.Go(func() error {
		// 如果是发送方，等待一段时间以确保客户端已发送检测消息
		if cResp.DetectBehavior.Role == "sender" {
			time.Sleep(1 * time.Second)
		}
		_ = session.clientTransporter.Send(cResp)
		return nil
	})
	_ = g.Wait()

	time.Sleep(time.Duration(cResp.DetectBehavior.ReadTimeoutMs+30000) * time.Millisecond)
}

// HandleClient 处理客户端
func (c *Controller) HandleClient(m *msg.NatHoleClient, transporter transport.MessageTransporter) {
	c.mu.RLock()
	session, ok := c.sessions[m.Sid]
	c.mu.RUnlock()
	if !ok {
		return
	}
	log.Tracef("处理客户端消息，sid [%s]，服务器名称: %s", session.sid, m.ProxyName)
	session.clientMsg = m
	session.clientTransporter = transporter
	select {
	case session.notifyCh <- struct{}{}:
	default:
	}
}

// HandleReport 处理报告
func (c *Controller) HandleReport(m *msg.NatHoleReport) {
	c.mu.RLock()
	session, ok := c.sessions[m.Sid]
	c.mu.RUnlock()
	if !ok {
		log.Tracef("sid [%s] 报告打洞成功: %v，但会话未找到", m.Sid, m.Success)
		return
	}
	if m.Success {
		c.analyzer.ReportSuccess(session.analysisKey, session.recommandMode, session.recommandIndex)
	}
	log.Infof("sid [%s] 报告打洞成功: %v，模式 %v，索引 %v",
		m.Sid, m.Success, session.recommandMode, session.recommandIndex)
}

// GenNatHoleResponse 生成 NAT 穿透响应
func (c *Controller) GenNatHoleResponse(transactionID string, session *Session, errInfo string) *msg.NatHoleResp {
	var sid string
	if session != nil {
		sid = session.sid
	}
	return &msg.NatHoleResp{
		TransactionID: transactionID,
		Sid:           sid,
		Error:         errInfo,
	}
}

// analysis 分析访问者和客户端的 NAT 类型和行为，然后进行打洞决策
// 返回给访问者和客户端的响应
func (c *Controller) analysis(session *Session) (*msg.NatHoleResp, *msg.NatHoleResp, error) {
	cm := session.clientMsg
	vm := session.visitorMsg

	cNatFeature, err := ClassifyNATFeature(cm.MappedAddrs, parseIPs(cm.AssistedAddrs))
	if err != nil {
		return nil, nil, fmt.Errorf("分类客户端 NAT 特征错误: %v", err)
	}

	vNatFeature, err := ClassifyNATFeature(vm.MappedAddrs, parseIPs(vm.AssistedAddrs))
	if err != nil {
		return nil, nil, fmt.Errorf("分类访问者 NAT 特征错误: %v", err)
	}
	session.cNatFeature = cNatFeature
	session.vNatFeature = vNatFeature
	session.genAnalysisKey()

	mode, index, cBehavior, vBehavior := c.analyzer.GetRecommandBehaviors(session.analysisKey, cNatFeature, vNatFeature)
	session.recommandMode = mode
	session.recommandIndex = index
	session.cBehavior = cBehavior
	session.vBehavior = vBehavior

	timeoutMs := max(cBehavior.SendDelayMs, vBehavior.SendDelayMs) + 5000
	if cBehavior.ListenRandomPorts > 0 || vBehavior.ListenRandomPorts > 0 {
		timeoutMs += 30000
	}

	protocol := vm.Protocol
	vResp := &msg.NatHoleResp{
		TransactionID:  vm.TransactionID,
		Sid:            session.sid,
		Protocol:       protocol,
		CandidateAddrs: slices.Compact(cm.MappedAddrs),
		AssistedAddrs:  slices.Compact(cm.AssistedAddrs),
		DetectBehavior: msg.NatHoleDetectBehavior{
			Mode:              mode,
			Role:              vBehavior.Role,
			TTL:               vBehavior.TTL,
			SendDelayMs:       vBehavior.SendDelayMs,
			ReadTimeoutMs:     timeoutMs - vBehavior.SendDelayMs,
			SendRandomPorts:   vBehavior.PortsRandomNumber,
			ListenRandomPorts: vBehavior.ListenRandomPorts,
			CandidatePorts:    getRangePorts(cm.MappedAddrs, cNatFeature.PortsDifference, vBehavior.PortsRangeNumber),
		},
	}
	cResp := &msg.NatHoleResp{
		TransactionID:  cm.TransactionID,
		Sid:            session.sid,
		Protocol:       protocol,
		CandidateAddrs: slices.Compact(vm.MappedAddrs),
		AssistedAddrs:  slices.Compact(vm.AssistedAddrs),
		DetectBehavior: msg.NatHoleDetectBehavior{
			Mode:              mode,
			Role:              cBehavior.Role,
			TTL:               cBehavior.TTL,
			SendDelayMs:       cBehavior.SendDelayMs,
			ReadTimeoutMs:     timeoutMs - cBehavior.SendDelayMs,
			SendRandomPorts:   cBehavior.PortsRandomNumber,
			ListenRandomPorts: cBehavior.ListenRandomPorts,
			CandidatePorts:    getRangePorts(vm.MappedAddrs, vNatFeature.PortsDifference, cBehavior.PortsRangeNumber),
		},
	}

	log.Debugf("sid [%s] 访问者 NAT: %+v，候选地址: %v；客户端 NAT: %+v，候选地址: %v，协议: %s",
		session.sid, *vNatFeature, vm.MappedAddrs, *cNatFeature, cm.MappedAddrs, protocol)
	log.Debugf("sid [%s] 访问者检测行为: %+v", session.sid, vResp.DetectBehavior)
	log.Debugf("sid [%s] 客户端检测行为: %+v", session.sid, cResp.DetectBehavior)
	return vResp, cResp, nil
}

// getRangePorts 获取端口范围
func getRangePorts(addrs []string, difference, maxNumber int) []msg.PortsRange {
	if maxNumber <= 0 {
		return nil
	}

	addr, isLast := lo.Last(addrs)
	if !isLast {
		return nil
	}
	var ports []msg.PortsRange
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil
	}
	ports = append(ports, msg.PortsRange{
		From: max(port-difference-5, port-maxNumber, 1),
		To:   min(port+difference+5, port+maxNumber, 65535),
	})
	return ports
}
