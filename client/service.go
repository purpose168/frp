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
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/fatedier/golib/crypto"
	"github.com/samber/lo"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/policy/security"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

func init() {
	crypto.DefaultSalt = "frp"
	// 禁用 quic-go 的接收缓冲区警告
	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")
	// 默认禁用 quic-go 的 ECN 支持，它可能在某些操作系统上导致问题
	if os.Getenv("QUIC_GO_DISABLE_ECN") == "" {
		os.Setenv("QUIC_GO_DISABLE_ECN", "true")
	}
}

type cancelErr struct {
	Err error
}

func (e cancelErr) Error() string {
	return e.Err.Error()
}

// ServiceOptions 包含用于创建新客户端服务的选项
type ServiceOptions struct {
	Common      *v1.ClientCommonConfig
	ProxyCfgs   []v1.ProxyConfigurer
	VisitorCfgs []v1.VisitorConfigurer

	UnsafeFeatures *security.UnsafeFeatures

	// ConfigFilePath 是用于初始化的配置文件路径
	// 如果为空，表示未使用配置文件进行初始化
	// 可能使用命令行参数初始化或直接调用
	ConfigFilePath string

	// ClientSpec 是控制客户端行为的客户端规范
	ClientSpec *msg.ClientSpec

	// ConnectorCreator 是创建新连接器以建立与服务端连接的函数
	// 连接器屏蔽了底层连接细节，无论是通过 TCP 还是 QUIC 连接
	// 以及是否使用了多路复用
	//
	// 如果未设置，将使用默认的 frpc 连接器
	// 通过使用自定义连接器，可以实现 VirtualClient，它通过管道而不是真实的物理连接连接到 frps
	ConnectorCreator func(context.Context, *v1.ClientCommonConfig) Connector

	// HandleWorkConnCb 是创建新工作连接时调用的回调函数
	//
	// 如果未设置，将使用默认的 frpc 实现
	HandleWorkConnCb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool
}

// setServiceOptionsDefault 为 ServiceOptions 设置默认值
func setServiceOptionsDefault(options *ServiceOptions) error {
	if options.Common != nil {
		if err := options.Common.Complete(); err != nil {
			return err
		}
	}
	if options.ConnectorCreator == nil {
		options.ConnectorCreator = NewConnector
	}
	return nil
}

// Service 是连接到 frps 并提供代理服务的客户端服务
type Service struct {
	ctlMu sync.RWMutex
	// 管理与服务端的控制连接
	ctl *Control
	// 从 frps 获取的唯一 ID，它将被附加到 loginMsg
	runID string

	// 身份验证运行时和加密材料
	auth *auth.ClientAuth

	// 用于管理 UI 和 API 的 Web 服务器
	webServer *httppkg.Server

	vnetController *vnet.Controller

	cfgMu       sync.RWMutex
	common      *v1.ClientCommonConfig
	proxyCfgs   []v1.ProxyConfigurer
	visitorCfgs []v1.VisitorConfigurer
	clientSpec  *msg.ClientSpec

	unsafeFeatures *security.UnsafeFeatures

	// 用于初始化此客户端的配置文件，如果未使用配置文件则为空字符串
	configFilePath string

	// 服务上下文
	ctx context.Context
	// 调用 cancel 以停止服务
	cancel                   context.CancelCauseFunc
	gracefulShutdownDuration time.Duration

	connectorCreator func(context.Context, *v1.ClientCommonConfig) Connector
	handleWorkConnCb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool
}

// NewService 创建新的客户端服务实例
func NewService(options ServiceOptions) (*Service, error) {
	if err := setServiceOptionsDefault(&options); err != nil {
		return nil, err
	}

	var webServer *httppkg.Server
	if options.Common.WebServer.Port > 0 {
		ws, err := httppkg.NewServer(options.Common.WebServer)
		if err != nil {
			return nil, err
		}
		webServer = ws
	}

	authRuntime, err := auth.BuildClientAuth(&options.Common.Auth)
	if err != nil {
		return nil, err
	}

	s := &Service{
		ctx:              context.Background(),
		auth:             authRuntime,
		webServer:        webServer,
		common:           options.Common,
		configFilePath:   options.ConfigFilePath,
		unsafeFeatures:   options.UnsafeFeatures,
		proxyCfgs:        options.ProxyCfgs,
		visitorCfgs:      options.VisitorCfgs,
		clientSpec:       options.ClientSpec,
		connectorCreator: options.ConnectorCreator,
		handleWorkConnCb: options.HandleWorkConnCb,
	}
	if webServer != nil {
		webServer.RouteRegister(s.registerRouteHandlers)
	}
	if options.Common.VirtualNet.Address != "" {
		s.vnetController = vnet.NewController(options.Common.VirtualNet)
	}
	return s, nil
}

// Run 运行客户端服务
func (svr *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	svr.ctx = xlog.NewContext(ctx, xlog.FromContextSafe(ctx))
	svr.cancel = cancel

	// 设置自定义 DNS 服务器
	if svr.common.DNSServer != "" {
		netpkg.SetDefaultDNSAddress(svr.common.DNSServer)
	}

	if svr.vnetController != nil {
		if err := svr.vnetController.Init(); err != nil {
			log.Errorf("初始化虚拟网络控制器错误: %v", err)
			return err
		}
		go func() {
			log.Infof("虚拟网络控制器启动中...")
			if err := svr.vnetController.Run(); err != nil {
				log.Warnf("虚拟网络控制器退出并出现错误: %v", err)
			}
		}()
	}

	if svr.webServer != nil {
		go func() {
			log.Infof("管理服务器监听在 %s", svr.webServer.Address())
			if err := svr.webServer.Run(); err != nil {
				log.Warnf("管理服务器退出并出现错误: %v", err)
			}
		}()
	}

	// 首次登录到 frps
	svr.loopLoginUntilSuccess(10*time.Second, lo.FromPtr(svr.common.LoginFailExit))
	if svr.ctl == nil {
		cancelCause := cancelErr{}
		_ = errors.As(context.Cause(svr.ctx), &cancelCause)
		return fmt.Errorf("登录到服务端失败: %v。启用 loginFailExit 后，将不会尝试额外的重试", cancelCause.Err)
	}

	go svr.keepControllerWorking()

	<-svr.ctx.Done()
	svr.stop()
	return nil
}

// keepControllerWorking 保持控制器工作，在控制器退出时重新连接
func (svr *Service) keepControllerWorking() {
	<-svr.ctl.Done()

	// 存在一种情况，登录成功但由于某些原因，控制器立即退出
	// 在这种情况下需要限制重连频率
	// 1 分钟内前三次重试的间隔将非常短，然后呈指数增长
	// 最大间隔为 20 秒
	wait.BackoffUntil(func() (bool, error) {
		// loopLoginUntilSuccess 是另一层循环，将持续尝试
		// 登录到服务端直到成功
		svr.loopLoginUntilSuccess(20*time.Second, false)
		if svr.ctl != nil {
			<-svr.ctl.Done()
			return false, errors.New("控制器已关闭，尝试另一个循环")
		}
		// 如果控制器为 nil，表示登录失败且服务也已关闭
		return false, nil
	}, wait.NewFastBackoffManager(
		wait.FastBackoffOptions{
			Duration:        time.Second,
			Factor:          2,
			Jitter:          0.1,
			MaxDuration:     20 * time.Second,
			FastRetryCount:  3,
			FastRetryDelay:  200 * time.Millisecond,
			FastRetryWindow: time.Minute,
			FastRetryJitter: 0.5,
		},
	), true, svr.ctx.Done())
}

// login 创建到 frps 的连接并将自身注册为客户端
// conn: 控制连接
// session: 如果不为 nil，则使用 tcp 多路复用
func (svr *Service) login() (conn net.Conn, connector Connector, err error) {
	xl := xlog.FromContextSafe(svr.ctx)
	connector = svr.connectorCreator(svr.ctx, svr.common)
	if err = connector.Open(); err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			connector.Close()
		}
	}()

	conn, err = connector.Connect()
	if err != nil {
		return
	}

	hostname, _ := os.Hostname()

	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		Hostname:  hostname,
		PoolCount: svr.common.Transport.PoolCount,
		User:      svr.common.User,
		ClientID:  svr.common.ClientID,
		Version:   version.Full(),
		Timestamp: time.Now().Unix(),
		RunID:     svr.runID,
		Metas:     svr.common.Metadatas,
	}
	if svr.clientSpec != nil {
		loginMsg.ClientSpec = *svr.clientSpec
	}

	// 添加认证
	if err = svr.auth.Setter.SetLogin(loginMsg); err != nil {
		return
	}

	if err = msg.WriteMsg(conn, loginMsg); err != nil {
		return
	}

	var loginRespMsg msg.LoginResp
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err = msg.ReadMsgInto(conn, &loginRespMsg); err != nil {
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	if loginRespMsg.Error != "" {
		err = fmt.Errorf("%s", loginRespMsg.Error)
		xl.Errorf("%s", loginRespMsg.Error)
		return
	}

	svr.runID = loginRespMsg.RunID
	xl.AddPrefix(xlog.LogPrefix{Name: "runID", Value: svr.runID})

	xl.Infof("登录到服务端成功，获取运行 ID [%s]", loginRespMsg.RunID)
	return
}

// loopLoginUntilSuccess 循环尝试登录到服务端直到成功
func (svr *Service) loopLoginUntilSuccess(maxInterval time.Duration, firstLoginExit bool) {
	xl := xlog.FromContextSafe(svr.ctx)

	loginFunc := func() (bool, error) {
		xl.Infof("尝试连接到服务端...")
		conn, connector, err := svr.login()
		if err != nil {
			xl.Warnf("连接到服务端错误: %v", err)
			if firstLoginExit {
				svr.cancel(cancelErr{Err: err})
			}
			return false, err
		}

		svr.cfgMu.RLock()
		proxyCfgs := svr.proxyCfgs
		visitorCfgs := svr.visitorCfgs
		svr.cfgMu.RUnlock()

		connEncrypted := svr.clientSpec == nil || svr.clientSpec.Type != "ssh-tunnel"

		sessionCtx := &SessionContext{
			Common:         svr.common,
			RunID:          svr.runID,
			Conn:           conn,
			ConnEncrypted:  connEncrypted,
			Auth:           svr.auth,
			Connector:      connector,
			VnetController: svr.vnetController,
		}
		ctl, err := NewControl(svr.ctx, sessionCtx)
		if err != nil {
			conn.Close()
			xl.Errorf("新建控制器错误: %v", err)
			return false, err
		}
		ctl.SetInWorkConnCallback(svr.handleWorkConnCb)

		ctl.Run(proxyCfgs, visitorCfgs)
		// 关闭并替换之前的控制器
		svr.ctlMu.Lock()
		if svr.ctl != nil {
			svr.ctl.Close()
		}
		svr.ctl = ctl
		svr.ctlMu.Unlock()
		return true, nil
	}

	// 尝试重新连接到服务端直到成功
	wait.BackoffUntil(loginFunc, wait.NewFastBackoffManager(
		wait.FastBackoffOptions{
			Duration:    time.Second,
			Factor:      2,
			Jitter:      0.1,
			MaxDuration: maxInterval,
		}), true, svr.ctx.Done())
}

// UpdateAllConfigurer 更新所有代理和访问者配置
func (svr *Service) UpdateAllConfigurer(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	svr.cfgMu.Lock()
	svr.proxyCfgs = proxyCfgs
	svr.visitorCfgs = visitorCfgs
	svr.cfgMu.Unlock()

	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()

	if ctl != nil {
		return svr.ctl.UpdateAllConfigurer(proxyCfgs, visitorCfgs)
	}
	return nil
}

// Close 立即关闭服务
func (svr *Service) Close() {
	svr.GracefulClose(time.Duration(0))
}

// GracefulClose 优雅地关闭服务，等待指定的时间
func (svr *Service) GracefulClose(d time.Duration) {
	svr.gracefulShutdownDuration = d
	svr.cancel(nil)
}

// stop 停止服务并清理资源
func (svr *Service) stop() {
	svr.ctlMu.Lock()
	defer svr.ctlMu.Unlock()
	if svr.ctl != nil {
		svr.ctl.GracefulClose(svr.gracefulShutdownDuration)
		svr.ctl = nil
	}
	if svr.webServer != nil {
		svr.webServer.Close()
		svr.webServer = nil
	}
}

// getProxyStatus 获取指定名称的代理状态
func (svr *Service) getProxyStatus(name string) (*proxy.WorkingStatus, bool) {
	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()

	if ctl == nil {
		return nil, false
	}
	return ctl.pm.GetProxyStatus(name)
}

// StatusExporter 返回状态导出器接口
func (svr *Service) StatusExporter() StatusExporter {
	return &statusExporterImpl{
		getProxyStatusFunc: svr.getProxyStatus,
	}
}

// StatusExporter 状态导出器接口，用于获取代理状态
type StatusExporter interface {
	GetProxyStatus(name string) (*proxy.WorkingStatus, bool)
}

// statusExporterImpl 状态导出器实现
type statusExporterImpl struct {
	getProxyStatusFunc func(name string) (*proxy.WorkingStatus, bool)
}

// GetProxyStatus 获取指定名称的代理状态
func (s *statusExporterImpl) GetProxyStatus(name string) (*proxy.WorkingStatus, bool) {
	return s.getProxyStatusFunc(name)
}
