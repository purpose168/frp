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
	"slices"
	"strconv"
)

const (
	// EasyNAT 简单 NAT 类型
	EasyNAT = "EasyNAT"
	// HardNAT 困难 NAT 类型
	HardNAT = "HardNAT"

	// BehaviorNoChange 行为无变化
	BehaviorNoChange = "BehaviorNoChange"
	// BehaviorIPChanged 行为 IP 变化
	BehaviorIPChanged = "BehaviorIPChanged"
	// BehaviorPortChanged 行为端口变化
	BehaviorPortChanged = "BehaviorPortChanged"
	// BehaviorBothChanged 行为 IP 和端口都变化
	BehaviorBothChanged = "BehaviorBothChanged"
)

// NatFeature NAT 特征
type NatFeature struct {
	// NatType NAT 类型
	NatType string
	// Behavior 行为
	Behavior string
	// PortsDifference 端口差异
	PortsDifference int
	// RegularPortsChange 端口是否规律变化
	RegularPortsChange bool
	// PublicNetwork 是否为公网网络
	PublicNetwork bool
}

// ClassifyNATFeature 分类 NAT 特征
func ClassifyNATFeature(addresses []string, localIPs []string) (*NatFeature, error) {
	if len(addresses) <= 1 {
		return nil, fmt.Errorf("地址数量不足")
	}
	natFeature := &NatFeature{}
	ipChanged := false
	portChanged := false

	var baseIP, basePort string
	var portMax, portMin int
	for _, addr := range addresses {
		ip, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		if slices.Contains(localIPs, ip) {
			natFeature.PublicNetwork = true
		}

		if baseIP == "" {
			baseIP = ip
			basePort = port
			portMax = portNum
			portMin = portNum
			continue
		}

		if portNum > portMax {
			portMax = portNum
		}
		if portNum < portMin {
			portMin = portNum
		}
		if baseIP != ip {
			ipChanged = true
		}
		if basePort != port {
			portChanged = true
		}
	}

	switch {
	case ipChanged && portChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorBothChanged
	case ipChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorIPChanged
	case portChanged:
		natFeature.NatType = HardNAT
		natFeature.Behavior = BehaviorPortChanged
	default:
		natFeature.NatType = EasyNAT
		natFeature.Behavior = BehaviorNoChange
	}
	if natFeature.Behavior == BehaviorPortChanged {
		natFeature.PortsDifference = portMax - portMin
		if natFeature.PortsDifference <= 5 && natFeature.PortsDifference >= 1 {
			natFeature.RegularPortsChange = true
		}
	}
	return natFeature, nil
}

// ClassifyFeatureCount 分类特征计数
func ClassifyFeatureCount(features []*NatFeature) (int, int, int) {
	easyCount := 0
	hardCount := 0
	// 对于 HardNAT
	portsChangedRegularCount := 0
	for _, feature := range features {
		if feature.NatType == EasyNAT {
			easyCount++
			continue
		}

		hardCount++
		if feature.RegularPortsChange {
			portsChangedRegularCount++
		}
	}
	return easyCount, hardCount, portsChangedRegularCount
}
