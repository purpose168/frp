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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

// createOIDCHTTPClient 为 OIDC 令牌请求创建具有自定义 TLS 和代理配置的 HTTP 客户端
func createOIDCHTTPClient(trustedCAFile string, insecureSkipVerify bool, proxyURL string) (*http.Client, error) {
	// 克隆默认传输以获取所有合理的默认值
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// 配置 TLS 设置
	if trustedCAFile != "" || insecureSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		}

		if trustedCAFile != "" && !insecureSkipVerify {
			caCert, err := os.ReadFile(trustedCAFile)
			if err != nil {
				return nil, fmt.Errorf("无法读取 OIDC CA 证书文件 %q: %w", trustedCAFile, err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("无法从文件 %q 解析 OIDC CA 证书", trustedCAFile)
			}

			tlsConfig.RootCAs = caCertPool
		}
		transport.TLSClientConfig = tlsConfig
	}

	// 配置代理设置
	if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("无法解析 OIDC 代理 URL %q: %w", proxyURL, err)
		}
		transport.Proxy = http.ProxyURL(parsedURL)
	} else {
		// 显式禁用代理以覆盖 DefaultTransport 的 ProxyFromEnvironment
		transport.Proxy = nil
	}

	return &http.Client{Transport: transport}, nil
}

// OidcAuthProvider 定义 OIDC 认证提供者结构
type OidcAuthProvider struct {
	additionalAuthScopes []v1.AuthScope

	tokenGenerator *clientcredentials.Config
	httpClient     *http.Client
}

// NewOidcAuthSetter 创建 OIDC 认证设置器
func NewOidcAuthSetter(additionalAuthScopes []v1.AuthScope, cfg v1.AuthOIDCClientConfig) (*OidcAuthProvider, error) {
	eps := make(map[string][]string)
	for k, v := range cfg.AdditionalEndpointParams {
		eps[k] = []string{v}
	}

	if cfg.Audience != "" {
		eps["audience"] = []string{cfg.Audience}
	}

	tokenGenerator := &clientcredentials.Config{
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		Scopes:         []string{cfg.Scope},
		TokenURL:       cfg.TokenEndpointURL,
		EndpointParams: eps,
	}

	// 如果需要，创建自定义 HTTP 客户端
	var httpClient *http.Client
	if cfg.TrustedCaFile != "" || cfg.InsecureSkipVerify || cfg.ProxyURL != "" {
		var err error
		httpClient, err = createOIDCHTTPClient(cfg.TrustedCaFile, cfg.InsecureSkipVerify, cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("无法创建 OIDC HTTP 客户端: %w", err)
		}
	}

	return &OidcAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		tokenGenerator:       tokenGenerator,
		httpClient:           httpClient,
	}, nil
}

// generateAccessToken 生成访问令牌
func (auth *OidcAuthProvider) generateAccessToken() (accessToken string, err error) {
	ctx := context.Background()
	if auth.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, auth.httpClient)
	}

	tokenObj, err := auth.tokenGenerator.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("无法为登录生成 OIDC 令牌: %v", err)
	}
	return tokenObj.AccessToken, nil
}

// SetLogin 设置登录消息的认证令牌
func (auth *OidcAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// SetPing 设置心跳消息的认证令牌
func (auth *OidcAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// SetNewWorkConn 设置新工作连接消息的认证令牌
func (auth *OidcAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// OidcTokenSourceAuthProvider 定义 OIDC 令牌源认证提供者结构
type OidcTokenSourceAuthProvider struct {
	additionalAuthScopes []v1.AuthScope

	valueSource *v1.ValueSource
}

// NewOidcTokenSourceAuthSetter 创建 OIDC 令牌源认证设置器
func NewOidcTokenSourceAuthSetter(additionalAuthScopes []v1.AuthScope, valueSource *v1.ValueSource) *OidcTokenSourceAuthProvider {
	return &OidcTokenSourceAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		valueSource:          valueSource,
	}
}

// generateAccessToken 生成访问令牌
func (auth *OidcTokenSourceAuthProvider) generateAccessToken() (accessToken string, err error) {
	ctx := context.Background()
	accessToken, err = auth.valueSource.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("无法为登录获取 OIDC 令牌: %v", err)
	}
	return
}

// SetLogin 设置登录消息的认证令牌
func (auth *OidcTokenSourceAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// SetPing 设置心跳消息的认证令牌
func (auth *OidcTokenSourceAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// SetNewWorkConn 设置新工作连接消息的认证令牌
func (auth *OidcTokenSourceAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

// TokenVerifier 定义令牌验证器接口
type TokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

// OidcAuthConsumer 定义 OIDC 认证消费者结构
type OidcAuthConsumer struct {
	additionalAuthScopes []v1.AuthScope

	verifier          TokenVerifier
	subjectsFromLogin []string
}

// NewTokenVerifier 创建令牌验证器
func NewTokenVerifier(cfg v1.AuthOIDCServerConfig) TokenVerifier {
	provider, err := oidc.NewProvider(context.Background(), cfg.Issuer)
	if err != nil {
		panic(err)
	}
	verifierConf := oidc.Config{
		ClientID:          cfg.Audience,
		SkipClientIDCheck: cfg.Audience == "",
		SkipExpiryCheck:   cfg.SkipExpiryCheck,
		SkipIssuerCheck:   cfg.SkipIssuerCheck,
	}
	return provider.Verifier(&verifierConf)
}

// NewOidcAuthVerifier 创建 OIDC 认证验证器
func NewOidcAuthVerifier(additionalAuthScopes []v1.AuthScope, verifier TokenVerifier) *OidcAuthConsumer {
	return &OidcAuthConsumer{
		additionalAuthScopes: additionalAuthScopes,
		verifier:             verifier,
		subjectsFromLogin:    []string{},
	}
}

// VerifyLogin 验证登录消息
func (auth *OidcAuthConsumer) VerifyLogin(loginMsg *msg.Login) (err error) {
	token, err := auth.verifier.Verify(context.Background(), loginMsg.PrivilegeKey)
	if err != nil {
		return fmt.Errorf("登录中的 OIDC 令牌无效: %v", err)
	}
	if !slices.Contains(auth.subjectsFromLogin, token.Subject) {
		auth.subjectsFromLogin = append(auth.subjectsFromLogin, token.Subject)
	}
	return nil
}

// verifyPostLoginToken 验证登录后的令牌
func (auth *OidcAuthConsumer) verifyPostLoginToken(privilegeKey string) (err error) {
	token, err := auth.verifier.Verify(context.Background(), privilegeKey)
	if err != nil {
		return fmt.Errorf("心跳中的 OIDC 令牌无效: %v", err)
	}
	if !slices.Contains(auth.subjectsFromLogin, token.Subject) {
		return fmt.Errorf("在登录和心跳中接收到不同的 OIDC 主题。 "+
			"原始主题: %s, "+
			"新主题: %s",
			auth.subjectsFromLogin, token.Subject)
	}
	return nil
}

// VerifyPing 验证心跳消息
func (auth *OidcAuthConsumer) VerifyPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	return auth.verifyPostLoginToken(pingMsg.PrivilegeKey)
}

// VerifyNewWorkConn 验证新工作连接消息
func (auth *OidcAuthConsumer) VerifyNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	return auth.verifyPostLoginToken(newWorkConnMsg.PrivilegeKey)
}
