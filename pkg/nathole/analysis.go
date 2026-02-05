// Copyright 2023 The frp Authors
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

package nathole

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/samber/lo"
)

var (
	// mode 0，双方都是 EasyNAT，公网网络始终是接收方
	// sender | receiver, ttl 7
	// receiver, ttl 7 | sender
	// sender | receiver, ttl 4
	// receiver, ttl 4 | sender
	// sender | receiver
	// receiver | sender
	// sender, sendDelayMs 5000 | receiver
	// sender, sendDelayMs 10000 | receiver
	// receiver | sender, sendDelayMs 5000
	// receiver | sender, sendDelayMs 10000
	mode0Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 7}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 4}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 5000}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 10000}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 5000}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 10000}),
	}

	// mode 1，HardNAT 是发送方，EasyNAT 是接收方，端口变化是规律的
	// sender | receiver, ttl 7, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, ttl 7, portsRangeNumber max 10
	// sender | receiver, ttl 4, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, ttl 4, portsRangeNumber max 10
	// sender | receiver, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, portsRangeNumber max 10
	mode1Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
	}

	// mode 2，HardNAT 是接收方，EasyNAT 是发送方
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports, ttl 7
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports, ttl 4
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports
	mode2Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 7},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 4},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256},
		),
	}

	// mode 3，对于 HardNAT & HardNAT，双方的端口变化都是规律的
	// sender, portsRangeNumber 10 | receiver, ttl 7, portsRangeNumber 10
	// sender, portsRangeNumber 10 | receiver, ttl 4, portsRangeNumber 10
	// sender, portsRangeNumber 10 | receiver, portsRangeNumber 10
	// receiver, ttl 7, portsRangeNumber 10 | sender, portsRangeNumber 10
	// receiver, ttl 4, portsRangeNumber 10 | sender, portsRangeNumber 10
	// receiver, portsRangeNumber 10 | sender, portsRangeNumber 10
	mode3Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
	}

	// mode 4，规律端口变化通常是发送方
	// sender, portsRandomNumber 1000, sendDelayMs: 2000 | receiver, listen 256 ports, ttl 7, portsRangeNumber 2
	// sender, portsRandomNumber 1000, sendDelayMs: 2000 | receiver, listen 256 ports, ttl 4, portsRangeNumber 2
	// sender, portsRandomNumber 1000, SendDelayMs: 2000 | receiver, listen 256 ports, portsRangeNumber 2
	mode4Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 7, PortsRangeNumber: 2},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 4, PortsRangeNumber: 2},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, PortsRangeNumber: 2},
		),
	}
)

// getBehaviorByMode 根据模式获取行为列表
func getBehaviorByMode(mode int) []lo.Tuple2[RecommandBehavior, RecommandBehavior] {
	switch mode {
	case 0:
		return mode0Behaviors
	case 1:
		return mode1Behaviors
	case 2:
		return mode2Behaviors
	case 3:
		return mode3Behaviors
	case 4:
		return mode4Behaviors
	}
	// 默认
	return mode0Behaviors
}

// getBehaviorByModeAndIndex 根据模式和索引获取行为
func getBehaviorByModeAndIndex(mode int, index int) (RecommandBehavior, RecommandBehavior) {
	behaviors := getBehaviorByMode(mode)
	if index >= len(behaviors) {
		return RecommandBehavior{}, RecommandBehavior{}
	}
	return behaviors[index].A, behaviors[index].B
}

// getBehaviorScoresByMode 根据模式获取行为分数列表
func getBehaviorScoresByMode(mode int, defaultScore int) []*BehaviorScore {
	return getBehaviorScoresByMode2(mode, defaultScore, defaultScore)
}

// getBehaviorScoresByMode2 根据模式和分数获取行为分数列表
func getBehaviorScoresByMode2(mode int, senderScore, receiverScore int) []*BehaviorScore {
	behaviors := getBehaviorByMode(mode)
	scores := make([]*BehaviorScore, 0, len(behaviors))
	for i := 0; i < len(behaviors); i++ {
		score := receiverScore
		if behaviors[i].A.Role == DetectRoleSender {
			score = senderScore
		}
		scores = append(scores, &BehaviorScore{Mode: mode, Index: i, Score: score})
	}
	return scores
}

// RecommandBehavior 推荐行为
type RecommandBehavior struct {
	// Role 角色
	Role string
	// TTL 生存时间
	TTL int
	// SendDelayMs 发送延迟（毫秒）
	SendDelayMs int
	// PortsRangeNumber 端口范围数量
	PortsRangeNumber int
	// PortsRandomNumber 随机端口数量
	PortsRandomNumber int
	// ListenRandomPorts 监听随机端口数量
	ListenRandomPorts int
}

// MakeHoleRecords 打洞记录
type MakeHoleRecords struct {
	mu             sync.Mutex
	scores         []*BehaviorScore
	LastUpdateTime time.Time
}

// NewMakeHoleRecords 创建新的打洞记录
func NewMakeHoleRecords(c, v *NatFeature) *MakeHoleRecords {
	scores := []*BehaviorScore{}
	easyCount, hardCount, portsChangedRegularCount := ClassifyFeatureCount([]*NatFeature{c, v})
	appendMode0 := func() {
		switch {
		case c.PublicNetwork:
			scores = append(scores, getBehaviorScoresByMode2(DetectMode0, 0, 1)...)
		case v.PublicNetwork:
			scores = append(scores, getBehaviorScoresByMode2(DetectMode0, 1, 0)...)
		default:
			scores = append(scores, getBehaviorScoresByMode(DetectMode0, 0)...)
		}
	}

	switch {
	case easyCount == 2:
		appendMode0()
	case hardCount == 1 && portsChangedRegularCount == 1:
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode2, 0)...)
		appendMode0()
	case hardCount == 1 && portsChangedRegularCount == 0:
		scores = append(scores, getBehaviorScoresByMode(DetectMode2, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 0)...)
		appendMode0()
	case hardCount == 2 && portsChangedRegularCount == 2:
		scores = append(scores, getBehaviorScoresByMode(DetectMode3, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode4, 0)...)
	case hardCount == 2 && portsChangedRegularCount == 1:
		scores = append(scores, getBehaviorScoresByMode(DetectMode4, 0)...)
	default:
		// 难以打洞，只是尝试一下
		scores = append(scores, getBehaviorScoresByMode(DetectMode0, 1)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 1)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode3, 1)...)
	}
	return &MakeHoleRecords{scores: scores, LastUpdateTime: time.Now()}
}

// ReportSuccess 报告成功
func (mhr *MakeHoleRecords) ReportSuccess(mode int, index int) {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()
	mhr.LastUpdateTime = time.Now()
	for i := range mhr.scores {
		score := mhr.scores[i]
		if score.Mode != mode || score.Index != index {
			continue
		}

		score.Score += 2
		score.Score = min(score.Score, 10)
		return
	}
}

// Recommand 推荐行为
func (mhr *MakeHoleRecords) Recommand() (mode, index int) {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	if len(mhr.scores) == 0 {
		return 0, 0
	}
	maxScore := slices.MaxFunc(mhr.scores, func(a, b *BehaviorScore) int {
		return cmp.Compare(a.Score, b.Score)
	})
	maxScore.Score--
	mhr.LastUpdateTime = time.Now()
	return maxScore.Mode, maxScore.Index
}

// BehaviorScore 行为分数
type BehaviorScore struct {
	// Mode 模式
	Mode int
	// Index 索引
	Index int
	// Score 分数，范围 -10 到 10
	Score int
}

// Analyzer 分析器
type Analyzer struct {
	// key 是客户端 IP + 访问者 IP
	records             map[string]*MakeHoleRecords
	dataReserveDuration time.Duration

	mu sync.Mutex
}

// NewAnalyzer 创建新的分析器
func NewAnalyzer(dataReserveDuration time.Duration) *Analyzer {
	return &Analyzer{
		records:             make(map[string]*MakeHoleRecords),
		dataReserveDuration: dataReserveDuration,
	}
}

// GetRecommandBehaviors 获取推荐行为
func (a *Analyzer) GetRecommandBehaviors(key string, c, v *NatFeature) (mode, index int, _ RecommandBehavior, _ RecommandBehavior) {
	a.mu.Lock()
	records, ok := a.records[key]
	if !ok {
		records = NewMakeHoleRecords(c, v)
		a.records[key] = records
	}
	a.mu.Unlock()

	mode, index = records.Recommand()
	cBehavior, vBehavior := getBehaviorByModeAndIndex(mode, index)

	switch mode {
	case DetectMode1:
		// HardNAT 始终是发送方
		if c.NatType == EasyNAT {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	case DetectMode2:
		// HardNAT 始终是接收方
		if c.NatType == HardNAT {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	case DetectMode4:
		// 规律端口变化始终是发送方
		if !c.RegularPortsChange {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	}
	return mode, index, cBehavior, vBehavior
}

// ReportSuccess 报告成功
func (a *Analyzer) ReportSuccess(key string, mode, index int) {
	a.mu.Lock()
	records, ok := a.records[key]
	a.mu.Unlock()
	if !ok {
		return
	}
	records.ReportSuccess(mode, index)
}

// Clean 清理过期数据
func (a *Analyzer) Clean() (int, int) {
	now := time.Now()
	total := 0
	count := 0

	// 清理 10 万条记录可能需要 5 毫秒
	a.mu.Lock()
	defer a.mu.Unlock()
	total = len(a.records)
	// 清理一段时间内未使用的记录
	for key, records := range a.records {
		if now.Sub(records.LastUpdateTime) > a.dataReserveDuration {
			delete(a.records, key)
			count++
		}
	}
	return count, total
}
