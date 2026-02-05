// 版权所有 2018 fatedier, fatedier@gmail.com
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

package health

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/util/xlog"
)

// ErrHealthCheckType 健康检查类型错误
var ErrHealthCheckType = errors.New("健康检查类型错误")

// Monitor 健康检查监控器
type Monitor struct {
	checkType      string
	interval       time.Duration
	timeout        time.Duration
	maxFailedTimes int

	// 用于 TCP 检查
	addr string

	// 用于 HTTP 检查
	url            string
	header         http.Header
	failedTimes    uint64
	statusOK       bool
	statusNormalFn func()
	statusFailedFn func()

	ctx    context.Context
	cancel context.CancelFunc
}

// NewMonitor 创建新的监控器
func NewMonitor(ctx context.Context, cfg v1.HealthCheckConfig, addr string,
	statusNormalFn func(), statusFailedFn func(),
) *Monitor {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 10
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 3
	}
	if cfg.MaxFailed <= 0 {
		cfg.MaxFailed = 1
	}
	newctx, cancel := context.WithCancel(ctx)

	var url string
	if cfg.Type == "http" && cfg.Path != "" {
		s := "http://" + addr
		if !strings.HasPrefix(cfg.Path, "/") {
			s += "/"
		}
		url = s + cfg.Path
	}
	header := make(http.Header)
	for _, h := range cfg.HTTPHeaders {
		header.Set(h.Name, h.Value)
	}

	return &Monitor{
		checkType:      cfg.Type,
		interval:       time.Duration(cfg.IntervalSeconds) * time.Second,
		timeout:        time.Duration(cfg.TimeoutSeconds) * time.Second,
		maxFailedTimes: cfg.MaxFailed,
		addr:           addr,
		url:            url,
		header:         header,
		statusOK:       false,
		statusNormalFn: statusNormalFn,
		statusFailedFn: statusFailedFn,
		ctx:            newctx,
		cancel:         cancel,
	}
}

// Start 启动监控器
func (monitor *Monitor) Start() {
	go monitor.checkWorker()
}

// Stop 停止监控器
func (monitor *Monitor) Stop() {
	monitor.cancel()
}

// checkWorker 检查工作器，定期执行健康检查
func (monitor *Monitor) checkWorker() {
	xl := xlog.FromContextSafe(monitor.ctx)
	for {
		doCtx, cancel := context.WithDeadline(monitor.ctx, time.Now().Add(monitor.timeout))
		err := monitor.doCheck(doCtx)

		// 检查此监控器是否已关闭
		select {
		case <-monitor.ctx.Done():
			cancel()
			return
		default:
			cancel()
		}

		if err == nil {
			xl.Tracef("执行一次健康检查成功")
			if !monitor.statusOK && monitor.statusNormalFn != nil {
				xl.Infof("健康检查状态变为成功")
				monitor.statusOK = true
				monitor.statusNormalFn()
			}
		} else {
			xl.Warnf("执行一次健康检查失败: %v", err)
			monitor.failedTimes++
			if monitor.statusOK && int(monitor.failedTimes) >= monitor.maxFailedTimes && monitor.statusFailedFn != nil {
				xl.Warnf("健康检查状态变为失败")
				monitor.statusOK = false
				monitor.statusFailedFn()
			}
		}

		time.Sleep(monitor.interval)
	}
}

// doCheck 执行健康检查
func (monitor *Monitor) doCheck(ctx context.Context) error {
	switch monitor.checkType {
	case "tcp":
		return monitor.doTCPCheck(ctx)
	case "http":
		return monitor.doHTTPCheck(ctx)
	default:
		return ErrHealthCheckType
	}
}

// doTCPCheck 执行 TCP 健康检查
func (monitor *Monitor) doTCPCheck(ctx context.Context) error {
	// 如果未指定 TCP 地址，则始终返回 nil
	if monitor.addr == "" {
		return nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", monitor.addr)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// doHTTPCheck 执行 HTTP 健康检查
func (monitor *Monitor) doHTTPCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", monitor.url, nil)
	if err != nil {
		return err
	}
	req.Header = monitor.header
	req.Host = monitor.header.Get("Host")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("执行 HTTP 健康检查，状态码为 [%d] 不是 2xx", resp.StatusCode)
	}
	return nil
}
