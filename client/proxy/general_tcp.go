// 版权所有 2023 The frp Authors
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的管理权限和
// 限制，请参阅许可证。

package proxy

import (
	"reflect"

	v1 "github.com/purpose168/frp/pkg/config/v1"
)

func init() {
	pxyConfs := []v1.ProxyConfigurer{
		&v1.TCPProxyConfig{},
		&v1.HTTPProxyConfig{},
		&v1.HTTPSProxyConfig{},
		&v1.STCPProxyConfig{},
		&v1.TCPMuxProxyConfig{},
	}
	for _, cfg := range pxyConfs {
		RegisterProxyFactory(reflect.TypeOf(cfg), NewGeneralTCPProxy)
	}
}

// GeneralTCPProxy 是 TCP 协议的 Proxy 接口的通用实现
// 如果默认的 GeneralTCPProxy 无法满足需求，可以自定义
// Proxy 接口的实现
type GeneralTCPProxy struct {
	*BaseProxy
}

// NewGeneralTCPProxy 创建新的通用 TCP 代理
func NewGeneralTCPProxy(baseProxy *BaseProxy, _ v1.ProxyConfigurer) Proxy {
	return &GeneralTCPProxy{
		BaseProxy: baseProxy,
	}
}
