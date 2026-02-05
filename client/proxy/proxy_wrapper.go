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
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/client/event"
	"github.com/fatedier/frp/client/health"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

const (
	ProxyPhaseNew         = "new"          // 新建
	ProxyPhaseWaitStart   = "wait start"   // 等待启动
	ProxyPhaseStartErr    = "start error"  // 启动错误
	ProxyPhaseRunning     = "running"      // 运行中
	ProxyPhaseCheckFailed = "check failed" // 检查失败
	ProxyPhaseClosed      = "closed"       // 已关闭
)

var (
	statusCheckInterval = 3 * time.Second  // 状态检查间隔
	waitResponseTimeout = 20 * time.Second // 等待响应超时
	startErrTimeout     = 30 * time.Second // 启动错误超时
)

// WorkingStatus 代理工作状态
type WorkingStatus struct {
	Name  string             `json:"name"`
	Type  string             `json:"type"`
	Phase string             `json:"status"`
	Err   string             `json:"err"`
	Cfg   v1.ProxyConfigurer `json:"cfg"`

	// 从服务端获取
	RemoteAddr string `json:"remote_addr"`
}

// Wrapper 代理包装器，封装代理实例和相关功能
type Wrapper struct {
	WorkingStatus

	// 底层代理
	pxy Proxy

	// 如果 ProxyConf 有健康检查配置
	// 监控器将监视其是否存活
	monitor *health.Monitor

	// 事件处理器
	handler event.Handler

	msgTransporter transport.MessageTransporter
	// 虚拟网络控制器
	vnetController *vnet.Controller

	health           uint32
	lastSendStartMsg time.Time
	lastStartErr     time.Time
	closeCh          chan struct{}
	healthNotifyCh   chan struct{}
	mu               sync.RWMutex

	xl  *xlog.Logger
	ctx context.Context
}

// NewWrapper 创建新的代理包装器
func NewWrapper(
	ctx context.Context,
	cfg v1.ProxyConfigurer,
	clientCfg *v1.ClientCommonConfig,
	encryptionKey []byte,
	eventHandler event.Handler,
	msgTransporter transport.MessageTransporter,
	vnetController *vnet.Controller,
) *Wrapper {
	baseInfo := cfg.GetBaseConfig()
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(baseInfo.Name)
	pw := &Wrapper{
		WorkingStatus: WorkingStatus{
			Name:  baseInfo.Name,
			Type:  baseInfo.Type,
			Phase: ProxyPhaseNew,
			Cfg:   cfg,
		},
		closeCh:        make(chan struct{}),
		healthNotifyCh: make(chan struct{}),
		handler:        eventHandler,
		msgTransporter: msgTransporter,
		vnetController: vnetController,
		xl:             xl,
		ctx:            xlog.NewContext(ctx, xl),
	}

	if baseInfo.HealthCheck.Type != "" && baseInfo.LocalPort > 0 {
		pw.health = 1 // 表示失败
		addr := net.JoinHostPort(baseInfo.LocalIP, strconv.Itoa(baseInfo.LocalPort))
		pw.monitor = health.NewMonitor(pw.ctx, baseInfo.HealthCheck, addr,
			pw.statusNormalCallback, pw.statusFailedCallback)
		xl.Tracef("启用健康检查监控器")
	}

	pw.pxy = NewProxy(pw.ctx, pw.Cfg, clientCfg, encryptionKey, pw.msgTransporter, pw.vnetController)
	return pw
}

// SetInWorkConnCallback 设置工作连接回调函数
func (pw *Wrapper) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pw.pxy.SetInWorkConnCallback(cb)
}

// SetRunningStatus 设置运行状态
func (pw *Wrapper) SetRunningStatus(remoteAddr string, respErr string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.Phase != ProxyPhaseWaitStart {
		return fmt.Errorf("状态不是等待启动，忽略启动消息")
	}

	pw.RemoteAddr = remoteAddr
	if respErr != "" {
		pw.Phase = ProxyPhaseStartErr
		pw.Err = respErr
		pw.lastStartErr = time.Now()
		return fmt.Errorf("%s", pw.Err)
	}

	if err := pw.pxy.Run(); err != nil {
		pw.close()
		pw.Phase = ProxyPhaseStartErr
		pw.Err = err.Error()
		pw.lastStartErr = time.Now()
		return err
	}

	pw.Phase = ProxyPhaseRunning
	pw.Err = ""
	return nil
}

// Start 启动代理包装器
func (pw *Wrapper) Start() {
	go pw.checkWorker()
	if pw.monitor != nil {
		go pw.monitor.Start()
	}
}

// Stop 停止代理包装器
func (pw *Wrapper) Stop() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	close(pw.closeCh)
	close(pw.healthNotifyCh)
	pw.pxy.Close()
	if pw.monitor != nil {
		pw.monitor.Stop()
	}
	pw.Phase = ProxyPhaseClosed
	pw.close()
}

// close 关闭代理
func (pw *Wrapper) close() {
	_ = pw.handler(&event.CloseProxyPayload{
		CloseProxyMsg: &msg.CloseProxy{
			ProxyName: pw.Name,
		},
	})
}

// checkWorker 检查工作器，定期检查代理状态
func (pw *Wrapper) checkWorker() {
	xl := pw.xl
	if pw.monitor != nil {
		// 让监控器先执行检查请求
		time.Sleep(500 * time.Millisecond)
	}
	for {
		// 检查代理状态
		now := time.Now()
		if atomic.LoadUint32(&pw.health) == 0 {
			pw.mu.Lock()
			if pw.Phase == ProxyPhaseNew ||
				pw.Phase == ProxyPhaseCheckFailed ||
				(pw.Phase == ProxyPhaseWaitStart && now.After(pw.lastSendStartMsg.Add(waitResponseTimeout))) ||
				(pw.Phase == ProxyPhaseStartErr && now.After(pw.lastStartErr.Add(startErrTimeout))) {

				xl.Tracef("状态从 [%s] 变为 [%s]", pw.Phase, ProxyPhaseWaitStart)
				pw.Phase = ProxyPhaseWaitStart

				var newProxyMsg msg.NewProxy
				pw.Cfg.MarshalToMsg(&newProxyMsg)
				pw.lastSendStartMsg = now
				_ = pw.handler(&event.StartProxyPayload{
					NewProxyMsg: &newProxyMsg,
				})
			}
			pw.mu.Unlock()
		} else {
			pw.mu.Lock()
			if pw.Phase == ProxyPhaseRunning || pw.Phase == ProxyPhaseWaitStart {
				pw.close()
				xl.Tracef("状态从 [%s] 变为 [%s]", pw.Phase, ProxyPhaseCheckFailed)
				pw.Phase = ProxyPhaseCheckFailed
			}
			pw.mu.Unlock()
		}

		select {
		case <-pw.closeCh:
			return
		case <-time.After(statusCheckInterval):
		case <-pw.healthNotifyCh:
		}
	}
}

// statusNormalCallback 健康检查正常回调函数
func (pw *Wrapper) statusNormalCallback() {
	xl := pw.xl
	atomic.StoreUint32(&pw.health, 0)
	_ = errors.PanicToError(func() {
		select {
		case pw.healthNotifyCh <- struct{}{}:
		default:
		}
	})
	xl.Infof("健康检查成功")
}

// statusFailedCallback 健康检查失败回调函数
func (pw *Wrapper) statusFailedCallback() {
	xl := pw.xl
	atomic.StoreUint32(&pw.health, 1)
	_ = errors.PanicToError(func() {
		select {
		case pw.healthNotifyCh <- struct{}{}:
		default:
		}
	})
	xl.Infof("健康检查失败")
}

// InWorkConn 处理工作连接
func (pw *Wrapper) InWorkConn(workConn net.Conn, m *msg.StartWorkConn) {
	xl := pw.xl
	pw.mu.RLock()
	pxy := pw.pxy
	pw.mu.RUnlock()
	if pxy != nil && pw.Phase == ProxyPhaseRunning {
		xl.Debugf("启动新的工作连接，本地地址: %s 远程地址: %s", workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		go pxy.InWorkConn(workConn, m)
	} else {
		workConn.Close()
	}
}

// GetStatus 获取代理状态
func (pw *Wrapper) GetStatus() *WorkingStatus {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	ps := &WorkingStatus{
		Name:       pw.Name,
		Type:       pw.Type,
		Phase:      pw.Phase,
		Err:        pw.Err,
		Cfg:        pw.Cfg,
		RemoteAddr: pw.RemoteAddr,
	}
	return ps
}
