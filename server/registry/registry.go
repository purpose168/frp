// Copyright 2025 The frp Authors
//
// 依据 Apache License, Version 2.0 许可证授权；
// 除非符合许可证的要求，否则您不能使用此文件。
// 您可以在以下网址获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则软件
// 根据许可证分发是基于“按原样”基础，
// 不附带任何明示或暗示的担保或条件。
// 请参阅许可证中有关管理权限和
// 限制的特定语言。

package registry

import (
	"fmt"
	"sync"
	"time"
)

// ClientInfo 捕获已连接的frpc实例的元数据
type ClientInfo struct {
	// Key 客户端键
	Key string
	// User 用户名
	User string
	// RawClientID 原始客户端ID
	RawClientID string
	// RunID 运行ID
	RunID string
	// Hostname 主机名
	Hostname string
	// IP IP地址
	IP string
	// FirstConnectedAt 首次连接时间
	FirstConnectedAt time.Time
	// LastConnectedAt 最后连接时间
	LastConnectedAt time.Time
	// DisconnectedAt 断开连接时间
	DisconnectedAt time.Time
	// Online 是否在线
	Online bool
}

// ClientRegistry 跟踪活动客户端，键为"{user}.{clientID}"（原始客户端ID为空时使用runID作为回退）
// 没有显式原始客户端ID的条目在断开连接时会被删除，以避免陈旧的离线记录
type ClientRegistry struct {
	// mu 读写锁
	mu sync.RWMutex
	// clients 客户端信息映射
	clients map[string]*ClientInfo
	// runIndex 运行ID索引
	runIndex map[string]string
}

// NewClientRegistry 创建一个新的客户端注册表
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients:  make(map[string]*ClientInfo),
		runIndex: make(map[string]string),
	}
}

// Register 存储/更新客户端的元数据，并返回注册表键以及是否与在线客户端冲突
func (cr *ClientRegistry) Register(user, rawClientID, runID, hostname, remoteAddr string) (key string, conflict bool) {
	if runID == "" {
		return "", false
	}

	effectiveID := rawClientID
	if effectiveID == "" {
		effectiveID = runID
	}
	key = cr.composeClientKey(user, effectiveID)
	enforceUnique := rawClientID != ""

	now := time.Now()
	cr.mu.Lock()
	defer cr.mu.Unlock()

	info, exists := cr.clients[key]
	if enforceUnique && exists && info.Online && info.RunID != "" && info.RunID != runID {
		return key, true
	}

	if !exists {
		info = &ClientInfo{
			Key:              key,
			User:             user,
			FirstConnectedAt: now,
		}
		cr.clients[key] = info
	} else if info.RunID != "" {
		delete(cr.runIndex, info.RunID)
	}

	info.RawClientID = rawClientID
	info.RunID = runID
	info.Hostname = hostname
	info.IP = remoteAddr
	if info.FirstConnectedAt.IsZero() {
		info.FirstConnectedAt = now
	}
	info.LastConnectedAt = now
	info.DisconnectedAt = time.Time{}
	info.Online = true

	cr.runIndex[runID] = key
	return key, false
}

// MarkOfflineByRunID 当相应的控制连接断开时，将客户端标记为离线
func (cr *ClientRegistry) MarkOfflineByRunID(runID string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	key, ok := cr.runIndex[runID]
	if !ok {
		return
	}
	if info, ok := cr.clients[key]; ok && info.RunID == runID {
		if info.RawClientID == "" {
			delete(cr.clients, key)
		} else {
			info.RunID = ""
			info.Online = false
			now := time.Now()
			info.DisconnectedAt = now
		}
	}
	delete(cr.runIndex, runID)
}

// List 返回所有已知客户端的快照
func (cr *ClientRegistry) List() []ClientInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	result := make([]ClientInfo, 0, len(cr.clients))
	for _, info := range cr.clients {
		result = append(result, *info)
	}
	return result
}

// GetByKey 通过复合键检索客户端（{user}.{clientID}，使用runID作为回退）
func (cr *ClientRegistry) GetByKey(key string) (ClientInfo, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	info, ok := cr.clients[key]
	if !ok {
		return ClientInfo{}, false
	}
	return *info, true
}

// ClientID 返回解析后的客户端标识符，供外部使用
func (info ClientInfo) ClientID() string {
	if info.RawClientID != "" {
		return info.RawClientID
	}
	return info.RunID
}

// GetByRunID 通过运行ID检索客户端
func (cr *ClientRegistry) GetByRunID(runID string) (ClientInfo, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	key, ok := cr.runIndex[runID]
	if !ok {
		return ClientInfo{}, false
	}
	info, ok := cr.clients[key]
	if !ok {
		return ClientInfo{}, false
	}
	return *info, true
}

// composeClientKey 组合客户端键
func (cr *ClientRegistry) composeClientKey(user, id string) string {
	switch {
	case user == "":
		return id
	case id == "":
		return user
	default:
		return fmt.Sprintf("%s.%s", user, id)
	}
}
