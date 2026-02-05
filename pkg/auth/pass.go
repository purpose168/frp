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

package auth

import (
	"github.com/purpose168/frp/pkg/msg"
)

// AlwaysPassVerifier 总是通过验证器
var AlwaysPassVerifier = &alwaysPass{}

var _ Verifier = &alwaysPass{}

// alwaysPass 总是通过验证结构
type alwaysPass struct{}

// VerifyLogin 验证登录消息，总是返回 nil（通过）
func (*alwaysPass) VerifyLogin(*msg.Login) error { return nil }

// VerifyPing 验证心跳消息，总是返回 nil（通过）
func (*alwaysPass) VerifyPing(*msg.Ping) error { return nil }

// VerifyNewWorkConn 验证新工作连接消息，总是返回 nil（通过）
func (*alwaysPass) VerifyNewWorkConn(*msg.NewWorkConn) error { return nil }
