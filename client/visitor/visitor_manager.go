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

package visitor

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/samber/lo"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/transport"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/pkg/vnet"
)

// Manager 访问者管理器结构
type Manager struct {
	clientCfg *v1.ClientCommonConfig
	cfgs      map[string]v1.VisitorConfigurer
	visitors  map[string]Visitor
	helper    Helper

	checkInterval           time.Duration
	keepVisitorsRunningOnce sync.Once

	mu  sync.RWMutex
	ctx context.Context

	stopCh chan struct{}
}

// NewManager 创建新的访问者管理器实例
func NewManager(
	ctx context.Context,
	runID string,
	clientCfg *v1.ClientCommonConfig,
	connectServer func() (net.Conn, error),
	msgTransporter transport.MessageTransporter,
	vnetController *vnet.Controller,
) *Manager {
	m := &Manager{
		clientCfg:     clientCfg,
		cfgs:          make(map[string]v1.VisitorConfigurer),
		visitors:      make(map[string]Visitor),
		checkInterval: 10 * time.Second,
		ctx:           ctx,
		stopCh:        make(chan struct{}),
	}
	m.helper = &visitorHelperImpl{
		connectServerFn: connectServer,
		msgTransporter:  msgTransporter,
		vnetController:  vnetController,
		transferConnFn:  m.TransferConn,
		runID:           runID,
	}
	return m
}

// keepVisitorsRunning 定期检查所有访问者的状态，如果某个访问者未运行，则启动它
// 它只在调用 Reload 并添加新访问者后才会启动
func (vm *Manager) keepVisitorsRunning() {
	xl := xlog.FromContextSafe(vm.ctx)

	ticker := time.NewTicker(vm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-vm.stopCh:
			xl.Tracef("优雅关闭访问者管理器")
			return
		case <-ticker.C:
			vm.mu.Lock()
			for _, cfg := range vm.cfgs {
				name := cfg.GetBaseConfig().Name
				if _, exist := vm.visitors[name]; !exist {
					xl.Infof("尝试启动访问者 [%s]", name)
					_ = vm.startVisitor(cfg)
				}
			}
			vm.mu.Unlock()
		}
	}
}

// Close 关闭访问者管理器
func (vm *Manager) Close() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	for _, v := range vm.visitors {
		v.Close()
	}
	select {
	case <-vm.stopCh:
	default:
		close(vm.stopCh)
	}
}

// startVisitor 启动访问者（调用前需持有锁）
func (vm *Manager) startVisitor(cfg v1.VisitorConfigurer) (err error) {
	xl := xlog.FromContextSafe(vm.ctx)
	name := cfg.GetBaseConfig().Name
	visitor, err := NewVisitor(vm.ctx, cfg, vm.clientCfg, vm.helper)
	if err != nil {
		xl.Warnf("创建访问者错误: %v", err)
		return
	}
	err = visitor.Run()
	if err != nil {
		xl.Warnf("启动错误: %v", err)
	} else {
		vm.visitors[name] = visitor
		xl.Infof("启动访问者成功")
	}
	return
}

// UpdateAll 更新所有访问者配置
func (vm *Manager) UpdateAll(cfgs []v1.VisitorConfigurer) {
	if len(cfgs) > 0 {
		// 只启动 keepVisitorsRunning 协程一次，并且仅当至少有一个访问者时
		vm.keepVisitorsRunningOnce.Do(func() {
			go vm.keepVisitorsRunning()
		})
	}

	xl := xlog.FromContextSafe(vm.ctx)
	cfgsMap := lo.KeyBy(cfgs, func(c v1.VisitorConfigurer) string {
		return c.GetBaseConfig().Name
	})
	vm.mu.Lock()
	defer vm.mu.Unlock()

	delNames := make([]string, 0)
	for name, oldCfg := range vm.cfgs {
		del := false
		cfg, ok := cfgsMap[name]
		if !ok || !reflect.DeepEqual(oldCfg, cfg) {
			del = true
		}

		if del {
			delNames = append(delNames, name)
			delete(vm.cfgs, name)
			if visitor, ok := vm.visitors[name]; ok {
				visitor.Close()
			}
			delete(vm.visitors, name)
		}
	}
	if len(delNames) > 0 {
		xl.Infof("访问者已移除: %v", delNames)
	}

	addNames := make([]string, 0)
	for _, cfg := range cfgs {
		name := cfg.GetBaseConfig().Name
		if _, ok := vm.cfgs[name]; !ok {
			vm.cfgs[name] = cfg
			addNames = append(addNames, name)
			_ = vm.startVisitor(cfg)
		}
	}
	if len(addNames) > 0 {
		xl.Infof("访问者已添加: %v", addNames)
	}
}

// TransferConn 将连接转移到访问者
func (vm *Manager) TransferConn(name string, conn net.Conn) error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	v, ok := vm.visitors[name]
	if !ok {
		return fmt.Errorf("访问者 [%s] 未找到", name)
	}
	return v.AcceptConn(conn)
}

// visitorHelperImpl 访问者辅助接口实现
type visitorHelperImpl struct {
	connectServerFn func() (net.Conn, error)
	msgTransporter  transport.MessageTransporter
	vnetController  *vnet.Controller
	transferConnFn  func(name string, conn net.Conn) error
	runID           string
}

// ConnectServer 连接到服务器
func (v *visitorHelperImpl) ConnectServer() (net.Conn, error) {
	return v.connectServerFn()
}

// TransferConn 转移连接
func (v *visitorHelperImpl) TransferConn(name string, conn net.Conn) error {
	return v.transferConnFn(name, conn)
}

// MsgTransporter 获取消息传输器
func (v *visitorHelperImpl) MsgTransporter() transport.MessageTransporter {
	return v.msgTransporter
}

// VNetController 获取虚拟网络控制器
func (v *visitorHelperImpl) VNetController() *vnet.Controller {
	return v.vnetController
}

// RunID 获取运行 ID
func (v *visitorHelperImpl) RunID() string {
	return v.runID
}
