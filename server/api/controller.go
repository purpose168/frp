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

package api

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

type Controller struct {
	// dependencies
	serverCfg      *v1.ServerConfig
	clientRegistry *registry.ClientRegistry
	pxyManager     ProxyManager
}

type ProxyManager interface {
	GetByName(name string) (proxy.Proxy, bool)
}

func NewController(
	serverCfg *v1.ServerConfig,
	clientRegistry *registry.ClientRegistry,
	pxyManager ProxyManager,
) *Controller {
	return &Controller{
		serverCfg:      serverCfg,
		clientRegistry: clientRegistry,
		pxyManager:     pxyManager,
	}
}

// /api/serverinfo - 获取服务器信息API
// 返回服务器的版本、端口配置、流量统计等完整信息
func (c *Controller) APIServerInfo(ctx *httppkg.Context) (any, error) {
	serverStats := mem.StatsCollector.GetServer()
	svrResp := ServerInfoResp{
		Version:               version.Full(),
		BindPort:              c.serverCfg.BindPort,
		VhostHTTPPort:         c.serverCfg.VhostHTTPPort,
		VhostHTTPSPort:        c.serverCfg.VhostHTTPSPort,
		TCPMuxHTTPConnectPort: c.serverCfg.TCPMuxHTTPConnectPort,
		KCPBindPort:           c.serverCfg.KCPBindPort,
		QUICBindPort:          c.serverCfg.QUICBindPort,
		SubdomainHost:         c.serverCfg.SubDomainHost,
		MaxPoolCount:          c.serverCfg.Transport.MaxPoolCount,
		MaxPortsPerClient:     c.serverCfg.MaxPortsPerClient,
		HeartBeatTimeout:      c.serverCfg.Transport.HeartbeatTimeout,
		AllowPortsStr:         types.PortsRangeSlice(c.serverCfg.AllowPorts).String(),
		TLSForce:              c.serverCfg.Transport.TLS.Force,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}
	// 对于返回结构体的API，我们可以直接返回
	// 但当前遗留代码中的GeneralResponse.Msg期望JSON字符串
	// 由于MakeHTTPHandlerFunc通过编码为JSON处理结构体，我们可以直接返回svrResp
	// 原始代码将其包装在GeneralResponse{Msg: string(json)}中
	// 如果返回svrResp，响应体将是svrResp的JSON
	// 我们需要检查前端期望{ "code": 200, "msg": "{...}" }还是{...}
	// 查看之前的代码：
	// res := GeneralResponse{Code: 200}
	// buf, _ := json.Marshal(&svrResp)
	// res.Msg = string(buf)
	// 响应体：{"code": 200, "msg": "{\"version\":...}"}
	// 这是双重编码的JSON！是的，看起来是这样
	// 再次检查dashboard_api.go原始代码
	// 是的：res.Msg = string(buf)
	// 因此前端期望{ "code": 200, "msg": "JSON_STRING" }
	// 虽然不太优雅，但我们必须保持兼容性

	return svrResp, nil
}

// /api/clients - 获取客户端列表API
// 返回所有连接的客户端信息，支持多种筛选条件
func (c *Controller) APIClientList(ctx *httppkg.Context) (any, error) {
	if c.clientRegistry == nil {
		return nil, fmt.Errorf("客户端注册表不可用")
	}

	userFilter := ctx.Query("user")
	clientIDFilter := ctx.Query("clientId")
	runIDFilter := ctx.Query("runId")
	statusFilter := strings.ToLower(ctx.Query("status"))

	records := c.clientRegistry.List()
	items := make([]ClientInfoResp, 0, len(records))
	for _, info := range records {
		if userFilter != "" && info.User != userFilter {
			continue
		}
		if clientIDFilter != "" && info.ClientID() != clientIDFilter {
			continue
		}
		if runIDFilter != "" && info.RunID != runIDFilter {
			continue
		}
		if !matchStatusFilter(info.Online, statusFilter) {
			continue
		}
		items = append(items, buildClientInfoResp(info))
	}

	slices.SortFunc(items, func(a, b ClientInfoResp) int {
		if v := cmp.Compare(a.User, b.User); v != 0 {
			return v
		}
		if v := cmp.Compare(a.ClientID, b.ClientID); v != 0 {
			return v
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return items, nil
}

// /api/clients/{key} - 获取指定客户端详细信息API
// 根据客户端密钥返回该客户端的完整连接信息
func (c *Controller) APIClientDetail(ctx *httppkg.Context) (any, error) {
	key := ctx.Param("key")
	if key == "" {
		return nil, fmt.Errorf("缺少客户端密钥")
	}

	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}

	info, ok := c.clientRegistry.GetByKey(key)
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("客户端 %s 未找到", key))
	}

	return buildClientInfoResp(info), nil
}

// /api/proxy/:type - 按类型获取代理列表API
// 返回指定类型的所有代理统计信息
func (c *Controller) APIProxyByType(ctx *httppkg.Context) (any, error) {
	proxyType := ctx.Param("type")

	proxyInfoResp := GetProxyInfoResp{}
	proxyInfoResp.Proxies = c.getProxyStatsByType(proxyType)
	slices.SortFunc(proxyInfoResp.Proxies, func(a, b *ProxyStatsInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return proxyInfoResp, nil
}

// /api/proxy/:type/:name - 按类型和名称获取特定代理信息API
// 返回指定类型和名称的代理详细信息
func (c *Controller) APIProxyByTypeAndName(ctx *httppkg.Context) (any, error) {
	proxyType := ctx.Param("type")
	name := ctx.Param("name")

	proxyStatsResp, code, msg := c.getProxyStatsByTypeAndName(proxyType, name)
	if code != 200 {
		return nil, httppkg.NewError(code, msg)
	}

	return proxyStatsResp, nil
}

// /api/traffic/:name - 获取指定代理流量信息API
// 返回指定代理名称的入站和出站流量数据
func (c *Controller) APIProxyTraffic(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")

	trafficResp := GetProxyTrafficResp{}
	trafficResp.Name = name
	proxyTrafficInfo := mem.StatsCollector.GetProxyTraffic(name)

	if proxyTrafficInfo == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "未找到代理信息")
	}
	trafficResp.TrafficIn = proxyTrafficInfo.TrafficIn
	trafficResp.TrafficOut = proxyTrafficInfo.TrafficOut

	return trafficResp, nil
}

// /api/proxies/:name - 获取指定代理完整信息API
// 返回指定名称代理的完整配置和状态信息
func (c *Controller) APIProxyByName(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")

	ps := mem.StatsCollector.GetProxyByName(name)
	if ps == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "未找到代理信息")
	}

	proxyInfo := GetProxyStatsResp{
		Name:            ps.Name,
		User:            ps.User,
		ClientID:        ps.ClientID,
		TodayTrafficIn:  ps.TodayTrafficIn,
		TodayTrafficOut: ps.TodayTrafficOut,
		CurConns:        ps.CurConns,
		LastStartTime:   ps.LastStartTime,
		LastCloseTime:   ps.LastCloseTime,
	}

	if pxy, ok := c.pxyManager.GetByName(name); ok {
		content, err := json.Marshal(pxy.GetConfigurer())
		if err != nil {
			log.Warnf("序列化代理 [%s] 配置信息错误: %v", name, err)
			return nil, httppkg.NewError(http.StatusBadRequest, "解析配置错误")
		}
		proxyInfo.Conf = getConfByType(ps.Type)
		if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
			log.Warnf("反序列化代理 [%s] 配置信息错误: %v", name, err)
			return nil, httppkg.NewError(http.StatusBadRequest, "解析配置错误")
		}
		proxyInfo.Status = "online"
		c.fillProxyClientInfo(&proxyClientInfo{
			clientVersion: &proxyInfo.ClientVersion,
		}, pxy)
	} else {
		proxyInfo.Status = "offline"
	}

	return proxyInfo, nil
}

// DELETE /api/proxies?status=offline - 删除离线代理API
// 仅支持删除状态为offline的代理
func (c *Controller) DeleteProxies(ctx *httppkg.Context) (any, error) {
	status := ctx.Query("status")
	if status != "offline" {
		return nil, httppkg.NewError(http.StatusBadRequest, "状态参数仅支持offline")
	}
	cleared, total := mem.StatsCollector.ClearOfflineProxies()
	log.Infof("已清除 [%d] 个离线代理，总计 [%d] 个代理", cleared, total)
	return nil, nil
}

func (c *Controller) getProxyStatsByType(proxyType string) (proxyInfos []*ProxyStatsInfo) {
	proxyStats := mem.StatsCollector.GetProxiesByType(proxyType)
	proxyInfos = make([]*ProxyStatsInfo, 0, len(proxyStats))
	for _, ps := range proxyStats {
		proxyInfo := &ProxyStatsInfo{
			User:     ps.User,
			ClientID: ps.ClientID,
		}
		if pxy, ok := c.pxyManager.GetByName(ps.Name); ok {
			content, err := json.Marshal(pxy.GetConfigurer())
			if err != nil {
				log.Warnf("序列化代理 [%s] 配置信息错误: %v", ps.Name, err)
				continue
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warnf("反序列化代理 [%s] 配置信息错误: %v", ps.Name, err)
				continue
			}
			proxyInfo.Status = "online"
			c.fillProxyClientInfo(&proxyClientInfo{
				clientVersion: &proxyInfo.ClientVersion,
			}, pxy)
		} else {
			proxyInfo.Status = "offline"
		}
		proxyInfo.Name = ps.Name
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
		proxyInfos = append(proxyInfos, proxyInfo)
	}
	return
}

func (c *Controller) getProxyStatsByTypeAndName(proxyType string, proxyName string) (proxyInfo GetProxyStatsResp, code int, msg string) {
	proxyInfo.Name = proxyName
	ps := mem.StatsCollector.GetProxiesByTypeAndName(proxyType, proxyName)
	if ps == nil {
		code = 404
		msg = "未找到代理信息"
	} else {
		proxyInfo.User = ps.User
		proxyInfo.ClientID = ps.ClientID
		if pxy, ok := c.pxyManager.GetByName(proxyName); ok {
			content, err := json.Marshal(pxy.GetConfigurer())
			if err != nil {
				log.Warnf("序列化代理 [%s] 配置信息错误: %v", ps.Name, err)
				code = 400
				msg = "解析配置错误"
				return
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warnf("反序列化代理 [%s] 配置信息错误: %v", ps.Name, err)
				code = 400
				msg = "解析配置错误"
				return
			}
			proxyInfo.Status = "online"
		} else {
			proxyInfo.Status = "offline"
		}
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
		code = 200
	}

	return
}

func buildClientInfoResp(info registry.ClientInfo) ClientInfoResp {
	resp := ClientInfoResp{
		Key:              info.Key,
		User:             info.User,
		ClientID:         info.ClientID(),
		RunID:            info.RunID,
		Hostname:         info.Hostname,
		ClientIP:         info.IP,
		FirstConnectedAt: toUnix(info.FirstConnectedAt),
		LastConnectedAt:  toUnix(info.LastConnectedAt),
		Online:           info.Online,
	}
	if !info.DisconnectedAt.IsZero() {
		resp.DisconnectedAt = info.DisconnectedAt.Unix()
	}
	return resp
}

type proxyClientInfo struct {
	user          *string
	clientID      *string
	clientVersion *string
}

func (c *Controller) fillProxyClientInfo(proxyInfo *proxyClientInfo, pxy proxy.Proxy) {
	loginMsg := pxy.GetLoginMsg()
	if loginMsg == nil {
		return
	}
	if proxyInfo.user != nil {
		*proxyInfo.user = loginMsg.User
	}
	if proxyInfo.clientVersion != nil {
		*proxyInfo.clientVersion = loginMsg.Version
	}
	if info, ok := c.clientRegistry.GetByRunID(loginMsg.RunID); ok {
		if proxyInfo.clientID != nil {
			*proxyInfo.clientID = info.ClientID()
		}
		return
	}
	if proxyInfo.clientID != nil {
		*proxyInfo.clientID = loginMsg.ClientID
		if *proxyInfo.clientID == "" {
			*proxyInfo.clientID = loginMsg.RunID
		}
	}
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func matchStatusFilter(online bool, filter string) bool {
	switch strings.ToLower(filter) {
	case "", "all":
		return true
	case "online":
		return online
	case "offline":
		return !online
	default:
		return true
	}
}

func getConfByType(proxyType string) any {
	switch v1.ProxyType(proxyType) {
	case v1.ProxyTypeTCP:
		return &TCPOutConf{}
	case v1.ProxyTypeTCPMUX:
		return &TCPMuxOutConf{}
	case v1.ProxyTypeUDP:
		return &UDPOutConf{}
	case v1.ProxyTypeHTTP:
		return &HTTPOutConf{}
	case v1.ProxyTypeHTTPS:
		return &HTTPSOutConf{}
	case v1.ProxyTypeSTCP:
		return &STCPOutConf{}
	case v1.ProxyTypeXTCP:
		return &XTCPOutConf{}
	default:
		return nil
	}
}
