// Copyright 2020 guylewin, guy@lewin.co.il
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
	"context"
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

// Setter 定义设置认证信息的接口
type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

// ClientAuth 定义客户端认证结构
type ClientAuth struct {
	Setter Setter
	key    []byte
}

// EncryptionKey 返回加密密钥
func (a *ClientAuth) EncryptionKey() []byte {
	return a.key
}

// BuildClientAuth 解析任何动态认证值并返回准备好的认证运行时。
// 调用者必须在调用此函数之前运行验证。
func BuildClientAuth(cfg *v1.AuthClientConfig) (*ClientAuth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("认证配置为空")
	}
	resolved := *cfg
	if resolved.Method == v1.AuthMethodToken && resolved.TokenSource != nil {
		token, err := resolved.TokenSource.Resolve(context.Background())
		if err != nil {
			return nil, fmt.Errorf("无法解析 auth.tokenSource: %w", err)
		}
		resolved.Token = token
	}
	setter, err := NewAuthSetter(resolved)
	if err != nil {
		return nil, err
	}
	return &ClientAuth{
		Setter: setter,
		key:    []byte(resolved.Token),
	}, nil
}

// NewAuthSetter 创建认证设置器
func NewAuthSetter(cfg v1.AuthClientConfig) (authProvider Setter, err error) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authProvider = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodOIDC:
		if cfg.OIDC.TokenSource != nil {
			authProvider = NewOidcTokenSourceAuthSetter(cfg.AdditionalScopes, cfg.OIDC.TokenSource)
		} else {
			authProvider, err = NewOidcAuthSetter(cfg.AdditionalScopes, cfg.OIDC)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("不支持的认证方法: %s", cfg.Method)
	}
	return authProvider, nil
}

// Verifier 定义验证认证信息的接口
type Verifier interface {
	VerifyLogin(*msg.Login) error
	VerifyPing(*msg.Ping) error
	VerifyNewWorkConn(*msg.NewWorkConn) error
}

// ServerAuth 定义服务端认证结构
type ServerAuth struct {
	Verifier Verifier
	key      []byte
}

// EncryptionKey 返回加密密钥
func (a *ServerAuth) EncryptionKey() []byte {
	return a.key
}

// BuildServerAuth 解析任何动态认证值并返回准备好的认证运行时。
// 调用者必须在调用此函数之前运行验证。
func BuildServerAuth(cfg *v1.AuthServerConfig) (*ServerAuth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("认证配置为空")
	}
	resolved := *cfg
	if resolved.Method == v1.AuthMethodToken && resolved.TokenSource != nil {
		token, err := resolved.TokenSource.Resolve(context.Background())
		if err != nil {
			return nil, fmt.Errorf("无法解析 auth.tokenSource: %w", err)
		}
		resolved.Token = token
	}
	return &ServerAuth{
		Verifier: NewAuthVerifier(resolved),
		key:      []byte(resolved.Token),
	}, nil
}

// NewAuthVerifier 创建认证验证器
func NewAuthVerifier(cfg v1.AuthServerConfig) (authVerifier Verifier) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authVerifier = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodOIDC:
		tokenVerifier := NewTokenVerifier(cfg.OIDC)
		authVerifier = NewOidcAuthVerifier(cfg.AdditionalScopes, tokenVerifier)
	}
	return authVerifier
}
