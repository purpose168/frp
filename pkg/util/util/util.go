// Copyright 2017 fatedier, fatedier@gmail.com
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

package util

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	mathrand "math/rand/v2"
	"net"
	"strconv"
	"strings"
	"time"
)

// RandID 返回一个在 frp 中使用的随机字符串
func RandID() (id string, err error) {
	return RandIDWithLen(16)
}

// RandIDWithLen 返回一个指定长度的随机字符串
func RandIDWithLen(idLen int) (id string, err error) {
	// 如果长度小于等于0，返回空字符串
	if idLen <= 0 {
		return "", nil
	}
	// 创建字节数组
	b := make([]byte, idLen/2+1)
	// 读取随机字节
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	// 将字节数组转换为十六进制字符串
	id = fmt.Sprintf("%x", b)
	// 截取指定长度
	return id[:idLen], nil
}

// GetAuthKey 根据令牌和时间戳生成认证密钥
func GetAuthKey(token string, timestamp int64) (key string) {
	// 创建 MD5 哈希上下文
	md5Ctx := md5.New()
	// 写入令牌
	md5Ctx.Write([]byte(token))
	// 写入时间戳
	md5Ctx.Write([]byte(strconv.FormatInt(timestamp, 10)))
	// 计算哈希值
	data := md5Ctx.Sum(nil)
	// 返回十六进制编码的字符串
	return hex.EncodeToString(data)
}

// CanonicalAddr 根据主机和端口返回规范化的地址
// 如果端口是 80 或 443，则只返回主机名
func CanonicalAddr(host string, port int) (addr string) {
	if port == 80 || port == 443 {
		addr = host
	} else {
		// 拼接主机和端口
		addr = net.JoinHostPort(host, strconv.Itoa(port))
	}
	return
}

// ParseRangeNumbers 解析范围数字字符串
// 支持格式：单个数字、范围数字（如 1000-2000）或它们的组合（如 1000-2000,2001,2002,3000-4000）
func ParseRangeNumbers(rangeStr string) (numbers []int64, err error) {
	// 去除首尾空格
	rangeStr = strings.TrimSpace(rangeStr)
	numbers = make([]int64, 0)
	// 例如：1000-2000,2001,2002,3000-4000
	numRanges := strings.Split(rangeStr, ",")
	for _, numRangeStr := range numRanges {
		// 1000-2000 或 2001
		numArray := strings.Split(numRangeStr, "-")
		// 长度：只有 1 或 2 是正确的
		rangeType := len(numArray)
		switch rangeType {
		case 1:
			// 单个数字
			singleNum, errRet := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("范围数字无效，%v", errRet)
				return
			}
			numbers = append(numbers, singleNum)
		case 2:
			// 范围数字
			minValue, errRet := strconv.ParseInt(strings.TrimSpace(numArray[0]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("范围数字无效，%v", errRet)
				return
			}
			maxValue, errRet := strconv.ParseInt(strings.TrimSpace(numArray[1]), 10, 64)
			if errRet != nil {
				err = fmt.Errorf("范围数字无效，%v", errRet)
				return
			}
			// 检查最大值是否小于最小值
			if maxValue < minValue {
				err = fmt.Errorf("范围数字无效")
				return
			}
			// 生成范围内的所有数字
			for i := minValue; i <= maxValue; i++ {
				numbers = append(numbers, i)
			}
		default:
			err = fmt.Errorf("范围数字无效")
			return
		}
	}
	return
}

// GenerateResponseErrorString 生成响应错误字符串
// 如果 detailed 为 true，返回详细的错误信息；否则返回摘要
func GenerateResponseErrorString(summary string, err error, detailed bool) string {
	if detailed {
		return err.Error()
	}
	return summary
}

// RandomSleep 随机休眠一段时间
// duration: 基础时长
// minRatio: 最小比例
// maxRatio: 最大比例
func RandomSleep(duration time.Duration, minRatio, maxRatio float64) time.Duration {
	// 计算最小值和最大值
	minValue := int64(minRatio * 1000.0)
	maxValue := int64(maxRatio * 1000.0)
	var n int64
	// 如果最大值小于等于最小值，使用最小值
	if maxValue <= minValue {
		n = minValue
	} else {
		// 生成随机数
		n = mathrand.Int64N(maxValue-minValue) + minValue
	}
	// 计算实际休眠时间
	d := duration * time.Duration(n) / time.Duration(1000)
	// 休眠
	time.Sleep(d)
	return d
}

// ConstantTimeEqString 常量时间比较两个字符串是否相等
// 用于防止时序攻击
func ConstantTimeEqString(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
