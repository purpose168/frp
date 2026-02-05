// 版权所有 2017 fatedier, fatedier@gmail.com
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"基础分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的管理权限和
// 限制，请参阅许可证。

package client

import (
	"net/http"

	"github.com/fatedier/frp/client/api"
	"github.com/fatedier/frp/client/proxy"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// registerRouteHandlers 注册路由处理器，用于处理管理API请求
func (svr *Service) registerRouteHandlers(helper *httppkg.RouterRegisterHelper) {
	apiController := newAPIController(svr)

	// 健康检查端点，无需身份验证
	helper.Router.HandleFunc("/healthz", healthz)

	// 需要身份验证的API路由和静态文件
	subRouter := helper.Router.NewRoute().Subrouter()
	subRouter.Use(helper.AuthMiddleware)
	subRouter.Use(httppkg.NewRequestLogger)
	subRouter.HandleFunc("/api/reload", httppkg.MakeHTTPHandlerFunc(apiController.Reload)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/stop", httppkg.MakeHTTPHandlerFunc(apiController.Stop)).Methods(http.MethodPost)
	subRouter.HandleFunc("/api/status", httppkg.MakeHTTPHandlerFunc(apiController.Status)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/config", httppkg.MakeHTTPHandlerFunc(apiController.GetConfig)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/config", httppkg.MakeHTTPHandlerFunc(apiController.PutConfig)).Methods(http.MethodPut)
	subRouter.Handle("/favicon.ico", http.FileServer(helper.AssetsFS)).Methods("GET")
	subRouter.PathPrefix("/static/").Handler(
		netpkg.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(helper.AssetsFS))),
	).Methods("GET")
	subRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})
}

// healthz 健康检查端点，返回HTTP 200状态码表示服务正常
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// newAPIController 创建新的API控制器实例，用于处理管理API请求
func newAPIController(svr *Service) *api.Controller {
	return api.NewController(api.ControllerParams{
		GetProxyStatus: svr.getAllProxyStatus,
		ServerAddr:     svr.common.ServerAddr,
		ConfigFilePath: svr.configFilePath,
		UnsafeFeatures: svr.unsafeFeatures,
		UpdateConfig:   svr.UpdateAllConfigurer,
		GracefulClose:  svr.GracefulClose,
	})
}

// getAllProxyStatus 获取所有代理的工作状态
func (svr *Service) getAllProxyStatus() []*proxy.WorkingStatus {
	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()
	if ctl == nil {
		return nil
	}
	return ctl.pm.GetAllProxyStatus()
}
