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

package validation

import (
	"errors"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	splugin "github.com/purpose168/frp/pkg/plugin/server"
)

var (
	// SupportedTransportProtocols 支持的传输协议列表
	SupportedTransportProtocols = []string{
		"tcp",
		"kcp",
		"quic",
		"websocket",
		"wss",
	}

	// SupportedAuthMethods 支持的认证方法列表
	SupportedAuthMethods = []v1.AuthMethod{
		"token",
		"oidc",
	}

	// SupportedAuthAdditionalScopes 支持的认证额外作用域列表
	SupportedAuthAdditionalScopes = []v1.AuthScope{
		"HeartBeats",
		"NewWorkConns",
	}

	// SupportedLogLevels 支持的日志级别列表
	SupportedLogLevels = []string{
		"trace",
		"debug",
		"info",
		"warn",
		"error",
	}

	// SupportedHTTPPluginOps 支持的 HTTP 插件操作列表
	SupportedHTTPPluginOps = []string{
		splugin.OpLogin,
		splugin.OpNewProxy,
		splugin.OpCloseProxy,
		splugin.OpPing,
		splugin.OpNewWorkConn,
		splugin.OpNewUserConn,
	}
)

// Warning 警告类型，用于表示验证过程中的警告信息
type Warning error

// AppendError 将多个错误合并为一个错误
// 参数 err 为基础错误
// 参数 errs 为要合并的错误列表
// 返回合并后的错误
func AppendError(err error, errs ...error) error {
	// 如果没有额外的错误，直接返回基础错误
	if len(errs) == 0 {
		return err
	}
	// 将所有错误合并为一个
	return errors.Join(append([]error{err}, errs...)...)
}
