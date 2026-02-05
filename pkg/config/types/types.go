// Copyright 2019 fatedier, fatedier@gmail.com
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

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	MB = 1024 * 1024
	KB = 1024

	BandwidthLimitModeClient = "client"
	BandwidthLimitModeServer = "server"
)

// BandwidthQuantity 表示带宽数量，支持 MB 和 KB 单位。
type BandwidthQuantity struct {
	s string // MB 或 KB

	i int64 // 字节数
}

// NewBandwidthQuantity 从字符串创建 BandwidthQuantity。
// 返回创建的 BandwidthQuantity 和可能的错误。
func NewBandwidthQuantity(s string) (BandwidthQuantity, error) {
	q := BandwidthQuantity{}
	err := q.UnmarshalString(s)
	if err != nil {
		return q, err
	}
	return q, nil
}

// Equal 比较两个 BandwidthQuantity 是否相等。
// 返回 true 表示相等，false 表示不相等。
func (q *BandwidthQuantity) Equal(u *BandwidthQuantity) bool {
	if q == nil && u == nil {
		return true
	}
	if q != nil && u != nil {
		return q.i == u.i
	}
	return false
}

// String 返回 BandwidthQuantity 的字符串表示。
func (q *BandwidthQuantity) String() string {
	return q.s
}

// UnmarshalString 从字符串解析 BandwidthQuantity。
// 返回解析过程中可能出现的错误。
func (q *BandwidthQuantity) UnmarshalString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var (
		base int64
		f    float64
		err  error
	)
	switch {
	case strings.HasSuffix(s, "MB"):
		base = MB
		fstr := strings.TrimSuffix(s, "MB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	case strings.HasSuffix(s, "KB"):
		base = KB
		fstr := strings.TrimSuffix(s, "KB")
		f, err = strconv.ParseFloat(fstr, 64)
		if err != nil {
			return err
		}
	default:
		return errors.New("不支持的单位")
	}

	q.s = s
	q.i = int64(f * float64(base))
	return nil
}

// UnmarshalJSON 从 JSON 数据解析 BandwidthQuantity。
// 返回解析过程中可能出现的错误。
func (q *BandwidthQuantity) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	return q.UnmarshalString(str)
}

// MarshalJSON 将 BandwidthQuantity 序列化为 JSON 数据。
// 返回 JSON 字节数组和可能的错误。
func (q *BandwidthQuantity) MarshalJSON() ([]byte, error) {
	return []byte("\"" + q.s + "\""), nil
}

// Bytes 返回 BandwidthQuantity 的字节数。
func (q *BandwidthQuantity) Bytes() int64 {
	return q.i
}

// PortsRange 表示端口范围，可以是单个端口或端口范围。
type PortsRange struct {
	Start  int `json:"start,omitempty"`
	End    int `json:"end,omitempty"`
	Single int `json:"single,omitempty"`
}

// PortsRangeSlice 是端口范围的切片。
type PortsRangeSlice []PortsRange

// String 返回端口范围切片的字符串表示。
// 格式为 "1000-2000,3000"。
func (p PortsRangeSlice) String() string {
	if len(p) == 0 {
		return ""
	}
	strs := []string{}
	for _, v := range p {
		if v.Single > 0 {
			strs = append(strs, strconv.Itoa(v.Single))
		} else {
			strs = append(strs, strconv.Itoa(v.Start)+"-"+strconv.Itoa(v.End))
		}
	}
	return strings.Join(strs, ",")
}

// str 的格式类似于 "1000-2000,3000,4000-5000"
// NewPortsRangeSliceFromString 从字符串解析端口范围切片。
// 返回解析后的端口范围切片和可能的错误。
func NewPortsRangeSliceFromString(str string) ([]PortsRange, error) {
	str = strings.TrimSpace(str)
	out := []PortsRange{}
	numRanges := strings.Split(str, ",")
	for _, numRangeStr := range numRanges {
		// 1000-2000 或 2001
		numArray := strings.Split(numRangeStr, "-")
		// 长度：只有 1 或 2 是正确的
		rangeType := len(numArray)
		switch rangeType {
		case 1:
			// 单个数字
			singleNum, err := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("范围数字无效，%v", err)
			}
			out = append(out, PortsRange{Single: int(singleNum)})
		case 2:
			// 范围数字
			minNum, err := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("范围数字无效，%v", err)
			}
			maxNum, err := strconv.ParseInt(strings.TrimSpace(numArray[1]), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("范围数字无效，%v", err)
			}
			if maxNum < minNum {
				return nil, fmt.Errorf("范围数字无效")
			}
			out = append(out, PortsRange{Start: int(minNum), End: int(maxNum)})
		default:
			return nil, fmt.Errorf("范围数字无效")
		}
	}
	return out, nil
}
