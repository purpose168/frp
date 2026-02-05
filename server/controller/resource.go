// 版权所有 2019 fatedier, fatedier@gmail.com
//
// 根据 Apache 许可证 2.0 版（简称"许可证"）授权；
// 除非遵守许可证，否则您不得使用本文件。
// 您可以从以下地址获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则本软件按"原样"分发，
// 不提供任何明示或暗示的保证或条件，包括但不限于对适销性或特定用途适用性的默示保证。
// 请参阅许可证了解具体的语言和权限限制。

package controller

import (
	"github.com/fatedier/frp/pkg/nathole"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/tcpmux"
	"github.com/fatedier/frp/pkg/util/vhost"
	"github.com/fatedier/frp/server/group"
	"github.com/fatedier/frp/server/ports"
	"github.com/fatedier/frp/server/visitor"
)

// All resource managers and controllers
type ResourceController struct {
	// 管理所有访问者监听器
	VisitorManager *visitor.Manager

	// TCP 组控制器
	TCPGroupCtl *group.TCPGroupCtl

	// HTTP 组控制器
	HTTPGroupCtl *group.HTTPGroupController

	// HTTPS 组控制器
	HTTPSGroupCtl *group.HTTPSGroupController

	// TCP Mux 组控制器
	TCPMuxGroupCtl *group.TCPMuxGroupCtl

	// 管理所有 TCP 端口
	TCPPortManager *ports.Manager

	// 管理所有 UDP 端口
	UDPPortManager *ports.Manager

	// 用于 HTTP 代理，转发 HTTP 请求
	HTTPReverseProxy *vhost.HTTPReverseProxy

	// 用于 HTTPS 代理，根据主机名和其他信息将请求路由到不同的客户端
	VhostHTTPSMuxer *vhost.HTTPSMuxer

	// NAT 穿透连接控制器
	NatHoleController *nathole.Controller

	// TCPMux HTTP CONNECT 多路复用器
	TCPMuxHTTPConnectMuxer *tcpmux.HTTPConnectTCPMuxer

	// 所有服务器管理插件
	PluginManager *plugin.Manager
}

// Close 关闭资源控制器
// 关闭所有相关的资源管理器和连接
func (rc *ResourceController) Close() error {
	if rc.VhostHTTPSMuxer != nil {
		rc.VhostHTTPSMuxer.Close()
	}
	if rc.TCPMuxHTTPConnectMuxer != nil {
		rc.TCPMuxHTTPConnectMuxer.Close()
	}
	return nil
}
