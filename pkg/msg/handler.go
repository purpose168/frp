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

package msg

import (
	"io"
	"reflect"
)

// AsyncHandler 异步处理器包装函数，将同步处理函数转换为异步处理函数
func AsyncHandler(f func(Message)) func(Message) {
	return func(m Message) {
		go f(m)
	}
}

// Dispatcher 用于向 net.Conn 发送消息或注册从 net.Conn 读取的消息处理器
type Dispatcher struct {
	rw io.ReadWriter

	sendCh         chan Message
	doneCh         chan struct{}
	msgHandlers    map[reflect.Type]func(Message)
	defaultHandler func(Message)
}

// NewDispatcher 创建新的消息分发器
func NewDispatcher(rw io.ReadWriter) *Dispatcher {
	return &Dispatcher{
		rw:          rw,
		sendCh:      make(chan Message, 100),
		doneCh:      make(chan struct{}),
		msgHandlers: make(map[reflect.Type]func(Message)),
	}
}

// Run 启动消息分发器，会阻塞直到发生 io.EOF 或其他错误
func (d *Dispatcher) Run() {
	go d.sendLoop()
	go d.readLoop()
}

// sendLoop 发送循环，从发送通道中读取消息并写入连接
func (d *Dispatcher) sendLoop() {
	for {
		select {
		case <-d.doneCh:
			return
		case m := <-d.sendCh:
			_ = WriteMsg(d.rw, m)
		}
	}
}

// readLoop 读取循环，从连接中读取消息并调用相应的处理器
func (d *Dispatcher) readLoop() {
	for {
		m, err := ReadMsg(d.rw)
		if err != nil {
			close(d.doneCh)
			return
		}

		if handler, ok := d.msgHandlers[reflect.TypeOf(m)]; ok {
			handler(m)
		} else if d.defaultHandler != nil {
			d.defaultHandler(m)
		}
	}
}

// Send 发送消息，如果分发器已关闭则返回 io.EOF
func (d *Dispatcher) Send(m Message) error {
	select {
	case <-d.doneCh:
		return io.EOF
	case d.sendCh <- m:
		return nil
	}
}

// RegisterHandler 注册消息处理器
func (d *Dispatcher) RegisterHandler(msg Message, handler func(Message)) {
	d.msgHandlers[reflect.TypeOf(msg)] = handler
}

// RegisterDefaultHandler 注册默认消息处理器
func (d *Dispatcher) RegisterDefaultHandler(handler func(Message)) {
	d.defaultHandler = handler
}

// Done 返回完成通道，当分发器关闭时该通道会被关闭
func (d *Dispatcher) Done() chan struct{} {
	return d.doneCh
}
