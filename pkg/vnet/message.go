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

package vnet

import (
	"encoding/binary"
	"fmt"
	"io"
)

// 最大消息大小
const (
	maxMessageSize = 1024 * 1024 // 1MB
)

// 消息格式：[长度(4字节)][数据(长度字节)]

// ReadMessage 从读取器中读取带长度前缀的消息
// r: 消息读取器
// 返回读取的消息数据和可能的错误
func ReadMessage(r io.Reader) ([]byte, error) {
	// 读取长度（4字节）
	var length uint32
	err := binary.Read(r, binary.LittleEndian, &length)
	if err != nil {
		return nil, fmt.Errorf("读取消息长度错误: %w", err)
	}

	// 检查长度以防止 DoS 攻击
	if length == 0 {
		return nil, fmt.Errorf("消息长度为 0")
	}
	if length > maxMessageSize {
		return nil, fmt.Errorf("消息太大: %d > %d", length, maxMessageSize)
	}

	// 读取消息数据
	data := make([]byte, length)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, fmt.Errorf("读取消息数据错误: %w", err)
	}

	return data, nil
}

// WriteMessage 向写入器中写入带长度前缀的消息
// w: 消息写入器
// data: 要写入的数据
// 返回可能的错误
func WriteMessage(w io.Writer, data []byte) error {
	// 获取数据长度
	length := uint32(len(data))
	if length == 0 {
		return fmt.Errorf("消息数据长度为 0")
	}
	if length > maxMessageSize {
		return fmt.Errorf("消息太大: %d > %d", length, maxMessageSize)
	}

	// 写入长度
	err := binary.Write(w, binary.LittleEndian, length)
	if err != nil {
		return fmt.Errorf("写入消息长度错误: %w", err)
	}

	// 写入消息数据
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("写入消息数据错误: %w", err)
	}

	return nil
}
