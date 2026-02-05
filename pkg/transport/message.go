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

package transport

import (
	"context"
	"reflect"
	"sync"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/msg"
)

// MessageTransporter 是消息传输器接口
type MessageTransporter interface {
	// Send 发送消息
	Send(msg.Message) error
	// Recv 接收消息
	// Recv(ctx context.Context, laneKey string, msgType string) (Message, error)
	// Do 将首先发送消息，然后使用相同的 laneKey 和指定的 msgType 接收消息。
	Do(ctx context.Context, req msg.Message, laneKey, recvMsgType string) (msg.Message, error)
	// Dispatch 将消息分发到在 Do 函数中通过其消息类型和 laneKey 注册的相关通道。
	Dispatch(m msg.Message, laneKey string) bool
	// 与 Dispatch 相同，但使用指定的消息类型。
	DispatchWithType(m msg.Message, msgType, laneKey string) bool
}

// MessageSender 是消息发送器接口
type MessageSender interface {
	Send(msg.Message) error
}

// NewMessageTransporter 创建一个新的消息传输器
// 参数 sender 是消息发送器
// 返回消息传输器实例
func NewMessageTransporter(sender MessageSender) MessageTransporter {
	return &transporterImpl{
		sender:   sender,
		registry: make(map[string]map[string]chan msg.Message),
	}
}

// transporterImpl 是消息传输器的实现
type transporterImpl struct {
	// sender 是消息发送器
	sender MessageSender

	// 第一个键是消息类型，第二个键是 lane 键。
	// Dispatch 将通过消息类型和 lane 键将消息分发到相关通道。
	registry map[string]map[string]chan msg.Message
	// mu 是读写锁
	mu sync.RWMutex
}

// Send 发送消息
func (impl *transporterImpl) Send(m msg.Message) error {
	return impl.sender.Send(m)
}

// Do 执行发送和接收操作
// 参数 ctx 是上下文
// 参数 req 是请求消息
// 参数 laneKey 是通道键
// 参数 recvMsgType 是接收消息类型
// 返回响应消息和可能的错误
func (impl *transporterImpl) Do(ctx context.Context, req msg.Message, laneKey, recvMsgType string) (msg.Message, error) {
	ch := make(chan msg.Message, 1)
	defer close(ch)
	unregisterFn := impl.registerMsgChan(ch, laneKey, recvMsgType)
	defer unregisterFn()

	if err := impl.Send(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

// DispatchWithType 使用指定的消息类型分发消息
// 参数 m 是消息
// 参数 msgType 是消息类型
// 参数 laneKey 是通道键
// 返回是否成功分发
func (impl *transporterImpl) DispatchWithType(m msg.Message, msgType, laneKey string) bool {
	var ch chan msg.Message
	impl.mu.RLock()
	byLaneKey, ok := impl.registry[msgType]
	if ok {
		ch = byLaneKey[laneKey]
	}
	impl.mu.RUnlock()

	if ch == nil {
		return false
	}

	if err := errors.PanicToError(func() {
		ch <- m
	}); err != nil {
		return false
	}
	return true
}

// Dispatch 分发消息
// 参数 m 是消息
// 参数 laneKey 是通道键
// 返回是否成功分发
func (impl *transporterImpl) Dispatch(m msg.Message, laneKey string) bool {
	msgType := reflect.TypeOf(m).Elem().Name()
	return impl.DispatchWithType(m, msgType, laneKey)
}

// registerMsgChan 注册消息通道
// 参数 recvCh 是接收通道
// 参数 laneKey 是通道键
// 参数 msgType 是消息类型
// 返回取消注册函数
func (impl *transporterImpl) registerMsgChan(recvCh chan msg.Message, laneKey string, msgType string) (unregister func()) {
	impl.mu.Lock()
	byLaneKey, ok := impl.registry[msgType]
	if !ok {
		byLaneKey = make(map[string]chan msg.Message)
		impl.registry[msgType] = byLaneKey
	}
	byLaneKey[laneKey] = recvCh
	impl.mu.Unlock()

	unregister = func() {
		impl.mu.Lock()
		delete(byLaneKey, laneKey)
		impl.mu.Unlock()
	}
	return
}
