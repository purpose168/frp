// Copyright 2018 fatedier, fatedier@gmail.com
//
// 依据 Apache License, Version 2.0 许可协议授权；
// 除非符合许可协议的规定，否则不得使用此文件。
// 您可以在以下网址获取许可协议的副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或者书面同意，否则本软件按"原样"分发，
// 不附带任何明示或暗示的担保或条件。
// 请参阅许可协议以了解管理权限和限制的特定语言。

package group

import (
	"errors"
)

var (
	// ErrGroupAuthFailed 组认证失败
	ErrGroupAuthFailed = errors.New("组认证失败")
	// ErrGroupParamsInvalid 组参数无效
	ErrGroupParamsInvalid = errors.New("组参数无效")
	// ErrListenerClosed 组监听器已关闭
	ErrListenerClosed = errors.New("组监听器已关闭")
	// ErrGroupDifferentPort 组应具有相同的远程端口
	ErrGroupDifferentPort = errors.New("组应具有相同的远程端口")
	// ErrProxyRepeated 组代理重复
	ErrProxyRepeated = errors.New("组代理重复")
)
