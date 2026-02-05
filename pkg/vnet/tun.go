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
	"context"
	"io"

	"github.com/fatedier/golib/pool"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	offset            = 16   // 偏移量
	defaultPacketSize = 1420 // 默认数据包大小
)

// TunDevice TUN 设备接口，继承 io.ReadWriteCloser 接口
type TunDevice interface {
	io.ReadWriteCloser
}

// OpenTun 打开 TUN 设备
// ctx: 上下文
// addr: 地址字符串
// 返回打开的 TUN 设备和可能的错误
func OpenTun(ctx context.Context, addr string) (TunDevice, error) {
	// 调用平台特定的 openTun 函数
	td, err := openTun(ctx, addr)
	if err != nil {
		return nil, err
	}

	// 获取 MTU，如果失败则使用默认值
	mtu, err := td.MTU()
	if err != nil {
		mtu = defaultPacketSize
	}

	// 计算缓冲区大小和批处理大小
	bufferSize := max(mtu, defaultPacketSize)
	batchSize := td.BatchSize()

	// 创建 TUN 设备包装器
	device := &tunDeviceWrapper{
		dev:         td,
		bufferSize:  bufferSize,
		readBuffers: make([][]byte, batchSize),
		sizeBuffer:  make([]int, batchSize),
	}

	// 初始化读取缓冲区
	for i := range device.readBuffers {
		device.readBuffers[i] = make([]byte, offset+bufferSize)
	}

	return device, nil
}

// tunDeviceWrapper TUN 设备包装器，用于批量读取和缓冲区管理
type tunDeviceWrapper struct {
	dev           tun.Device // 底层 TUN 设备
	bufferSize    int        // 缓冲区大小
	readBuffers   [][]byte   // 读取缓冲区数组
	packetBuffers [][]byte   // 数据包缓冲区数组
	sizeBuffer    []int      // 大小缓冲区数组
}

// Read 从 TUN 设备读取数据
// p: 目标字节切片
// 返回读取的字节数和可能的错误
func (d *tunDeviceWrapper) Read(p []byte) (int, error) {
	// 如果有缓存的数据包，直接返回
	if len(d.packetBuffers) > 0 {
		n := copy(p, d.packetBuffers[0])
		d.packetBuffers = d.packetBuffers[1:]
		return n, nil
	}

	// 从设备批量读取
	n, err := d.dev.Read(d.readBuffers, d.sizeBuffer, offset)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}

	// 处理读取的数据包
	for i := range n {
		if d.sizeBuffer[i] <= 0 {
			continue
		}
		d.packetBuffers = append(d.packetBuffers, d.readBuffers[i][offset:offset+d.sizeBuffer[i]])
	}

	// 复制第一个数据包到目标缓冲区
	dataSize := copy(p, d.packetBuffers[0])
	d.packetBuffers = d.packetBuffers[1:]

	return dataSize, nil
}

// Write 向 TUN 设备写入数据
// p: 要写入的字节切片
// 返回写入的字节数和可能的错误
func (d *tunDeviceWrapper) Write(p []byte) (int, error) {
	// 从池中获取缓冲区
	buf := pool.GetBuf(offset + d.bufferSize)
	defer pool.PutBuf(buf)

	// 复制数据到缓冲区
	n := copy(buf[offset:], p)
	_, err := d.dev.Write([][]byte{buf[:offset+n]}, offset)
	return n, err
}

// Close 关闭 TUN 设备
// 返回可能的错误
func (d *tunDeviceWrapper) Close() error {
	return d.dev.Close()
}
