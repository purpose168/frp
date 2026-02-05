// Copyright 2024 The frp Authors
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
// See License for the specific language governing permissions and
// limitations under License.

package config

import (
	"fmt"

	"github.com/fatedier/frp/pkg/util/util"
)

// NumberPair 数字对结构体
type NumberPair struct {
	First  int64
	Second int64
}

// parseNumberRangePair 解析数字范围对
func parseNumberRangePair(firstRangeStr, secondRangeStr string) ([]NumberPair, error) {
	firstRangeNumbers, err := util.ParseRangeNumbers(firstRangeStr)
	if err != nil {
		return nil, err
	}
	secondRangeNumbers, err := util.ParseRangeNumbers(secondRangeStr)
	if err != nil {
		return nil, err
	}
	if len(firstRangeNumbers) != len(secondRangeNumbers) {
		return nil, fmt.Errorf("第一个范围和第二个范围的数字数量不是成对的")
	}
	pairs := make([]NumberPair, 0, len(firstRangeNumbers))
	for i := 0; i < len(firstRangeNumbers); i++ {
		pairs = append(pairs, NumberPair{
			First:  firstRangeNumbers[i],
			Second: secondRangeNumbers[i],
		})
	}
	return pairs, nil
}

// parseNumberRange 解析数字范围
func parseNumberRange(firstRangeStr string) ([]int64, error) {
	return util.ParseRangeNumbers(firstRangeStr)
}
