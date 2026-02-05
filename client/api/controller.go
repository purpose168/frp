// 版权所有 2025 The frp Authors
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

package api

import (
	"cmp"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/purpose168/frp/client/proxy"
	"github.com/purpose168/frp/pkg/config"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/config/v1/validation"
	"github.com/purpose168/frp/pkg/policy/security"
	httppkg "github.com/purpose168/frp/pkg/util/http"
	"github.com/purpose168/frp/pkg/util/log"
)

// Controller 处理 frpc 的 HTTP API 请求
type Controller struct {
	// getProxyStatus 返回当前的代理状态
	// 如果控制连接未建立则返回 nil
	getProxyStatus func() []*proxy.WorkingStatus

	// serverAddr 是用于显示的 frps 服务端地址
	serverAddr string

	// configFilePath 是配置文件的路径
	configFilePath string

	// unsafeFeatures 用于验证
	unsafeFeatures *security.UnsafeFeatures

	// updateConfig 更新代理和访问者配置
	updateConfig func(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error

	// gracefulClose 优雅地停止服务
	gracefulClose func(d time.Duration)
}

// ControllerParams 包含创建 APIController 的参数
type ControllerParams struct {
	GetProxyStatus func() []*proxy.WorkingStatus
	ServerAddr     string
	ConfigFilePath string
	UnsafeFeatures *security.UnsafeFeatures
	UpdateConfig   func(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error
	GracefulClose  func(d time.Duration)
}

// NewController 创建新的控制器
func NewController(params ControllerParams) *Controller {
	return &Controller{
		getProxyStatus: params.GetProxyStatus,
		serverAddr:     params.ServerAddr,
		configFilePath: params.ConfigFilePath,
		unsafeFeatures: params.UnsafeFeatures,
		updateConfig:   params.UpdateConfig,
		gracefulClose:  params.GracefulClose,
	}
}

// Reload 处理 GET /api/reload
func (c *Controller) Reload(ctx *httppkg.Context) (any, error) {
	strictConfigMode := false
	strictStr := ctx.Query("strictConfig")
	if strictStr != "" {
		strictConfigMode, _ = strconv.ParseBool(strictStr)
	}

	cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(c.configFilePath, strictConfigMode)
	if err != nil {
		log.Warnf("重新加载 frpc 代理配置错误: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	if _, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs, c.unsafeFeatures); err != nil {
		log.Warnf("重新加载 frpc 代理配置错误: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	if err := c.updateConfig(proxyCfgs, visitorCfgs); err != nil {
		log.Warnf("重新加载 frpc 代理配置错误: %s", err.Error())
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	log.Infof("成功重新加载配置")
	return nil, nil
}

// Stop 处理 POST /api/stop
func (c *Controller) Stop(ctx *httppkg.Context) (any, error) {
	go c.gracefulClose(100 * time.Millisecond)
	return nil, nil
}

// Status 处理 GET /api/status
func (c *Controller) Status(ctx *httppkg.Context) (any, error) {
	res := make(StatusResp)
	ps := c.getProxyStatus()
	if ps == nil {
		return res, nil
	}

	for _, status := range ps {
		res[status.Type] = append(res[status.Type], c.buildProxyStatusResp(status))
	}

	for _, arrs := range res {
		if len(arrs) <= 1 {
			continue
		}
		slices.SortFunc(arrs, func(a, b ProxyStatusResp) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}
	return res, nil
}

// GetConfig 处理 GET /api/config
func (c *Controller) GetConfig(ctx *httppkg.Context) (any, error) {
	if c.configFilePath == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "frpc 没有配置文件路径")
	}

	content, err := os.ReadFile(c.configFilePath)
	if err != nil {
		log.Warnf("加载 frpc 配置文件错误: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	return string(content), nil
}

// PutConfig 处理 PUT /api/config
func (c *Controller) PutConfig(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("读取请求体错误: %v", err))
	}

	if len(body) == 0 {
		return nil, httppkg.NewError(http.StatusBadRequest, "请求体不能为空")
	}

	if err := os.WriteFile(c.configFilePath, body, 0o600); err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, fmt.Sprintf("写入内容到 frpc 配置文件错误: %v", err))
	}
	return nil, nil
}

// buildProxyStatusResp 从 proxy.WorkingStatus 创建 ProxyStatusResp
func (c *Controller) buildProxyStatusResp(status *proxy.WorkingStatus) ProxyStatusResp {
	psr := ProxyStatusResp{
		Name:   status.Name,
		Type:   status.Type,
		Status: status.Phase,
		Err:    status.Err,
	}
	baseCfg := status.Cfg.GetBaseConfig()
	if baseCfg.LocalPort != 0 {
		psr.LocalAddr = net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort))
	}
	psr.Plugin = baseCfg.Plugin.Type

	if status.Err == "" {
		psr.RemoteAddr = status.RemoteAddr
		if slices.Contains([]string{"tcp", "udp"}, status.Type) {
			psr.RemoteAddr = c.serverAddr + psr.RemoteAddr
		}
	}
	return psr
}
