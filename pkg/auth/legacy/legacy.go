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

package legacy

// BaseConfig 定义基础认证配置结构
type BaseConfig struct {
	// AuthenticationMethod 指定用于 frpc 与 frps 身份验证的身份验证方法。
	// 如果指定为 "token"，则将令牌读取到登录消息中。
	// 如果指定为 "oidc"，则使用 OIDC 设置颁发 OIDC（Open ID Connect）令牌。
	// 默认情况下，此值为 "token"。
	AuthenticationMethod string `ini:"authentication_method" json:"authentication_method"`
	// AuthenticateHeartBeats 指定是否在发送到 frps 的心跳中包含身份验证令牌。
	// 默认情况下，此值为 false。
	AuthenticateHeartBeats bool `ini:"authenticate_heartbeats" json:"authenticate_heartbeats"`
	// AuthenticateNewWorkConns 指定是否在发送到 frps 的新工作连接中包含身份验证令牌。
	// 默认情况下，此值为 false。
	AuthenticateNewWorkConns bool `ini:"authenticate_new_work_conns" json:"authenticate_new_work_conns"`
}

// getDefaultBaseConf 返回默认的基础配置
func getDefaultBaseConf() BaseConfig {
	return BaseConfig{
		AuthenticationMethod:     "token",
		AuthenticateHeartBeats:   false,
		AuthenticateNewWorkConns: false,
	}
}

// ClientConfig 定义客户端认证配置结构
type ClientConfig struct {
	BaseConfig       `ini:",extends"`
	OidcClientConfig `ini:",extends"`
	TokenConfig      `ini:",extends"`
}

// GetDefaultClientConf 返回默认的客户端配置
func GetDefaultClientConf() ClientConfig {
	return ClientConfig{
		BaseConfig:       getDefaultBaseConf(),
		OidcClientConfig: getDefaultOidcClientConf(),
		TokenConfig:      getDefaultTokenConf(),
	}
}

// ServerConfig 定义服务端认证配置结构
type ServerConfig struct {
	BaseConfig       `ini:",extends"`
	OidcServerConfig `ini:",extends"`
	TokenConfig      `ini:",extends"`
}

// GetDefaultServerConf 返回默认的服务端配置
func GetDefaultServerConf() ServerConfig {
	return ServerConfig{
		BaseConfig:       getDefaultBaseConf(),
		OidcServerConfig: getDefaultOidcServerConf(),
		TokenConfig:      getDefaultTokenConf(),
	}
}

// OidcClientConfig 定义 OIDC 客户端配置结构
type OidcClientConfig struct {
	// OidcClientID 指定在 AuthenticationMethod == "oidc" 时用于在 OIDC 身份验证中获取令牌的客户端 ID。
	// 默认情况下，此值为 ""。
	OidcClientID string `ini:"oidc_client_id" json:"oidc_client_id"`
	// OidcClientSecret 指定在 AuthenticationMethod == "oidc" 时用于在 OIDC 身份验证中获取令牌的客户端密钥。
	// 默认情况下，此值为 ""。
	OidcClientSecret string `ini:"oidc_client_secret" json:"oidc_client_secret"`
	// OidcAudience 指定在 AuthenticationMethod == "oidc" 时 OIDC 身份验证中令牌的受众。
	// 默认情况下，此值为 ""。
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcScope 指定在 AuthenticationMethod == "oidc" 时 OIDC 身份验证中令牌的范围。
	// 默认情况下，此值为 ""。
	OidcScope string `ini:"oidc_scope" json:"oidc_scope"`
	// OidcTokenEndpointURL 指定实现 OIDC 令牌端点的 URL。
	// 如果 AuthenticationMethod == "oidc"，将使用它来获取 OIDC 令牌。
	// 默认情况下，此值为 ""。
	OidcTokenEndpointURL string `ini:"oidc_token_endpoint_url" json:"oidc_token_endpoint_url"`

	// OidcAdditionalEndpointParams 指定要发送的附加参数
	// 此字段将在 OIDC 令牌生成器中转换为 map[string][]string
	// 该字段将通过前缀 "oidc_additional_" 设置
	OidcAdditionalEndpointParams map[string]string `ini:"-" json:"oidc_additional_endpoint_params"`
}

// getDefaultOidcClientConf 返回默认的 OIDC 客户端配置
func getDefaultOidcClientConf() OidcClientConfig {
	return OidcClientConfig{
		OidcClientID:                 "",
		OidcClientSecret:             "",
		OidcAudience:                 "",
		OidcScope:                    "",
		OidcTokenEndpointURL:         "",
		OidcAdditionalEndpointParams: make(map[string]string),
	}
}

// OidcServerConfig 定义 OIDC 服务端配置结构
type OidcServerConfig struct {
	// OidcIssuer 指定用于验证 OIDC 令牌的颁发者。
	// 此颁发者将用于加载公钥以验证签名，并将与 OIDC 令牌中的颁发者声明进行比较。
	// 如果 AuthenticationMethod == "oidc"，将使用它。
	// 默认情况下，此值为 ""。
	OidcIssuer string `ini:"oidc_issuer" json:"oidc_issuer"`
	// OidcAudience 指定 OIDC 令牌在验证时应包含的受众。
	// 如果此值为空，将跳过受众（"客户端 ID"）验证。
	// 当 AuthenticationMethod == "oidc" 时将使用它。
	// 默认情况下，此值为 ""。
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcSkipExpiryCheck 指定是否跳过检查 OIDC 令牌是否已过期。
	// 当 AuthenticationMethod == "oidc" 时将使用它。
	// 默认情况下，此值为 false。
	OidcSkipExpiryCheck bool `ini:"oidc_skip_expiry_check" json:"oidc_skip_expiry_check"`
	// OidcSkipIssuerCheck 指定是否跳过检查 OIDC 令牌的颁发者声明是否与 OidcIssuer 中指定的颁发者匹配。
	// 当 AuthenticationMethod == "oidc" 时将使用它。
	// 默认情况下，此值为 false。
	OidcSkipIssuerCheck bool `ini:"oidc_skip_issuer_check" json:"oidc_skip_issuer_check"`
}

// getDefaultOidcServerConf 返回默认的 OIDC 服务端配置
func getDefaultOidcServerConf() OidcServerConfig {
	return OidcServerConfig{
		OidcIssuer:          "",
		OidcAudience:        "",
		OidcSkipExpiryCheck: false,
		OidcSkipIssuerCheck: false,
	}
}

// TokenConfig 定义令牌配置结构
type TokenConfig struct {
	// Token 指定用于创建要发送到服务器的密钥的授权令牌。
	// 服务器必须具有匹配的令牌才能使授权成功。
	// 默认情况下，此值为 ""。
	Token string `ini:"token" json:"token"`
}

// getDefaultTokenConf 返回默认的令牌配置
func getDefaultTokenConf() TokenConfig {
	return TokenConfig{
		Token: "",
	}
}
