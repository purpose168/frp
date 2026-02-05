package event

import (
	"errors"

	"github.com/purpose168/frp/pkg/msg"
)

// ErrPayloadType 负载类型错误
var ErrPayloadType = errors.New("负载类型错误")

// Handler 事件处理器函数类型
type Handler func(payload any) error

// StartProxyPayload 启动代理事件的负载
type StartProxyPayload struct {
	NewProxyMsg *msg.NewProxy
}

// CloseProxyPayload 关闭代理事件的负载
type CloseProxyPayload struct {
	CloseProxyMsg *msg.CloseProxy
}
