package ports

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/purpose168/frp/pkg/config/types"
)

const (
	// MinPort 最小可用端口
	MinPort = 1
	// MaxPort 最大可用端口
	MaxPort = 65535
	// MaxPortReservedDuration 端口最大保留时间
	// 超过这个时间未使用的保留端口将被清理
	MaxPortReservedDuration = time.Duration(24) * time.Hour
	// CleanReservedPortsInterval 清理保留端口的时间间隔
	CleanReservedPortsInterval = time.Hour
)

var (
	// ErrPortAlreadyUsed 端口已被使用
	ErrPortAlreadyUsed = errors.New("端口已被使用")
	// ErrPortNotAllowed 端口不被允许使用
	ErrPortNotAllowed = errors.New("端口不被允许使用")
	// ErrPortUnAvailable 端口不可用
	ErrPortUnAvailable = errors.New("端口不可用")
	// ErrNoAvailablePort 没有可用端口
	ErrNoAvailablePort = errors.New("没有可用端口")
)

// PortCtx 端口上下文信息
// 记录端口的使用情况和相关信息
type PortCtx struct {
	// ProxyName 代理名称
	ProxyName string
	// Port 端口号
	Port int
	// Closed 端口是否已关闭
	Closed bool
	// UpdateTime 端口信息最后更新时间
	UpdateTime time.Time
}

// Manager 端口管理器
// 负责管理端口的分配、释放和清理
type Manager struct {
	// reservedPorts 保留的端口，键为代理名称
	reservedPorts map[string]*PortCtx
	// usedPorts 正在使用的端口，键为端口号
	usedPorts map[int]*PortCtx
	// freePorts 可用的端口，键为端口号
	freePorts map[int]struct{}

	// bindAddr 绑定地址
	bindAddr string
	// netType 网络类型，如"tcp"或"udp"
	netType string
	// mu 保护并发访问
	mu sync.Mutex
}

// NewManager 创建一个新的端口管理器
// 参数netType是网络类型，如"tcp"或"udp"
// 参数bindAddr是绑定地址
// 参数allowPorts是允许使用的端口范围
func NewManager(netType string, bindAddr string, allowPorts []types.PortsRange) *Manager {
	pm := &Manager{
		reservedPorts: make(map[string]*PortCtx),
		usedPorts:     make(map[int]*PortCtx),
		freePorts:     make(map[int]struct{}),
		bindAddr:      bindAddr,
		netType:       netType,
	}
	if len(allowPorts) > 0 {
		for _, pair := range allowPorts {
			if pair.Single > 0 {
				pm.freePorts[pair.Single] = struct{}{}
			} else {
				for i := pair.Start; i <= pair.End; i++ {
					pm.freePorts[i] = struct{}{}
				}
			}
		}
	} else {
		for i := MinPort; i <= MaxPort; i++ {
			pm.freePorts[i] = struct{}{}
		}
	}
	go pm.cleanReservedPortsWorker()
	return pm
}

// Acquire 分配一个端口
// 参数name是代理名称
// 参数port是请求的端口号，0表示随机分配
// 返回值是实际分配的端口号和可能的错误
func (pm *Manager) Acquire(name string, port int) (realPort int, err error) {
	portCtx := &PortCtx{
		ProxyName:  name,
		Closed:     false,
		UpdateTime: time.Now(),
	}

	var ok bool

	pm.mu.Lock()
	defer func() {
		if err == nil {
			portCtx.Port = realPort
		}
		pm.mu.Unlock()
	}()

	// 首先检查保留的端口
	if port == 0 {
		if ctx, ok := pm.reservedPorts[name]; ok {
			if pm.isPortAvailable(ctx.Port) {
				realPort = ctx.Port
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
				return
			}
		}
	}

	if port == 0 {
		// 随机获取一个端口
		count := 0
		maxTryTimes := 5
		for k := range pm.freePorts {
			count++
			if count > maxTryTimes {
				break
			}
			if pm.isPortAvailable(k) {
				realPort = k
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
				break
			}
		}
		if realPort == 0 {
			err = ErrNoAvailablePort
		}
	} else {
		// 指定端口
		if _, ok = pm.freePorts[port]; ok {
			if pm.isPortAvailable(port) {
				realPort = port
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
			} else {
				err = ErrPortUnAvailable
			}
		} else {
			if _, ok = pm.usedPorts[port]; ok {
				err = ErrPortAlreadyUsed
			} else {
				err = ErrPortNotAllowed
			}
		}
	}
	return
}

// isPortAvailable 检查端口是否可用
// 参数port是要检查的端口号
// 返回值是端口是否可用
func (pm *Manager) isPortAvailable(port int) bool {
	if pm.netType == "udp" {
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(pm.bindAddr, strconv.Itoa(port)))
		if err != nil {
			return false
		}
		l, err := net.ListenUDP("udp", addr)
		if err != nil {
			return false
		}
		l.Close()
		return true
	}

	l, err := net.Listen(pm.netType, net.JoinHostPort(pm.bindAddr, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// Release 释放一个端口
// 参数port是要释放的端口号
func (pm *Manager) Release(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if ctx, ok := pm.usedPorts[port]; ok {
		pm.freePorts[port] = struct{}{}
		delete(pm.usedPorts, port)
		ctx.Closed = true
		ctx.UpdateTime = time.Now()
	}
}

// cleanReservedPortsWorker 清理保留端口的工作协程
// 如果保留端口在过去24小时内未使用，则释放该端口
func (pm *Manager) cleanReservedPortsWorker() {
	for {
		time.Sleep(CleanReservedPortsInterval)
		pm.mu.Lock()
		for name, ctx := range pm.reservedPorts {
			if ctx.Closed && time.Since(ctx.UpdateTime) > MaxPortReservedDuration {
				delete(pm.reservedPorts, name)
			}
		}
		pm.mu.Unlock()
	}
}
