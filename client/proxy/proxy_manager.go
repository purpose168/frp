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
	"reflect"
	"sync"

	"github.com/samber/lo"

	"github.com/purpose168/frp/client/event"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/transport"
	"github.com/purpose168/frp/pkg/util/xlog"
	"github.com/purpose168/frp/pkg/vnet"
)

// Manager 代理管理器，管理所有代理实例
type Manager struct {
	proxies            map[string]*Wrapper
	msgTransporter     transport.MessageTransporter
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool
	vnetController     *vnet.Controller

	closed bool
	mu     sync.RWMutex

	encryptionKey []byte
	clientCfg     *v1.ClientCommonConfig

	ctx context.Context
}

// NewManager 创建新的代理管理器
func NewManager(
	ctx context.Context,
	clientCfg *v1.ClientCommonConfig,
	encryptionKey []byte,
	msgTransporter transport.MessageTransporter,
	vnetController *vnet.Controller,
) *Manager {
	return &Manager{
		proxies:        make(map[string]*Wrapper),
		msgTransporter: msgTransporter,
		vnetController: vnetController,
		closed:         false,
		encryptionKey:  encryptionKey,
		clientCfg:      clientCfg,
		ctx:            ctx,
	}
}

// StartProxy 启动指定的代理
func (pm *Manager) StartProxy(name string, remoteAddr string, serverRespErr string) error {
	pm.mu.RLock()
	pxy, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("代理 [%s] 未找到", name)
	}

	err := pxy.SetRunningStatus(remoteAddr, serverRespErr)
	if err != nil {
		return err
	}
	return nil
}

// SetInWorkConnCallback 设置工作连接回调函数
func (pm *Manager) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pm.inWorkConnCallback = cb
}

// Close 关闭所有代理
func (pm *Manager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, pxy := range pm.proxies {
		pxy.Stop()
	}
	pm.proxies = make(map[string]*Wrapper)
}

// HandleWorkConn 处理工作连接
func (pm *Manager) HandleWorkConn(name string, workConn net.Conn, m *msg.StartWorkConn) {
	pm.mu.RLock()
	pw, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if ok {
		pw.InWorkConn(workConn, m)
	} else {
		workConn.Close()
	}
}

// HandleEvent 处理事件
func (pm *Manager) HandleEvent(payload any) error {
	var m msg.Message
	switch e := payload.(type) {
	case *event.StartProxyPayload:
		m = e.NewProxyMsg
	case *event.CloseProxyPayload:
		m = e.CloseProxyMsg
	default:
		return event.ErrPayloadType
	}

	return pm.msgTransporter.Send(m)
}

// GetAllProxyStatus 获取所有代理的状态
func (pm *Manager) GetAllProxyStatus() []*WorkingStatus {
	ps := make([]*WorkingStatus, 0)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		ps = append(ps, pxy.GetStatus())
	}
	return ps
}

// GetProxyStatus 获取指定代理的状态
func (pm *Manager) GetProxyStatus(name string) (*WorkingStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pxy, ok := pm.proxies[name]; ok {
		return pxy.GetStatus(), true
	}
	return nil, false
}

// UpdateAll 更新所有代理配置
func (pm *Manager) UpdateAll(proxyCfgs []v1.ProxyConfigurer) {
	xl := xlog.FromContextSafe(pm.ctx)
	proxyCfgsMap := lo.KeyBy(proxyCfgs, func(c v1.ProxyConfigurer) string {
		return c.GetBaseConfig().Name
	})
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delPxyNames := make([]string, 0)
	for name, pxy := range pm.proxies {
		del := false
		cfg, ok := proxyCfgsMap[name]
		if !ok || !reflect.DeepEqual(pxy.Cfg, cfg) {
			del = true
		}

		if del {
			delPxyNames = append(delPxyNames, name)
			delete(pm.proxies, name)
			pxy.Stop()
		}
	}
	if len(delPxyNames) > 0 {
		xl.Infof("proxy removed: %s", delPxyNames)
	}

	addPxyNames := make([]string, 0)
	for _, cfg := range proxyCfgs {
		name := cfg.GetBaseConfig().Name
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewWrapper(pm.ctx, cfg, pm.clientCfg, pm.encryptionKey, pm.HandleEvent, pm.msgTransporter, pm.vnetController)
			if pm.inWorkConnCallback != nil {
				pxy.SetInWorkConnCallback(pm.inWorkConnCallback)
			}
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)

			pxy.Start()
		}
	}
	if len(addPxyNames) > 0 {
		xl.Infof("proxy added: %s", addPxyNames)
	}
}
