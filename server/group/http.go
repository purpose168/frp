package group

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPGroupController struct {
	// 按组名索引的组
	groups map[string]*HTTPGroup

	// 为每个组向vhostRouter注册createConn
	// createConn将从组中的一个代理获取连接
	vhostRouter *vhost.Routers

	mu sync.Mutex
}

func NewHTTPGroupController(vhostRouter *vhost.Routers) *HTTPGroupController {
	return &HTTPGroupController{
		groups:      make(map[string]*HTTPGroup),
		vhostRouter: vhostRouter,
	}
}

func (ctl *HTTPGroupController) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (err error) {
	indexKey := group
	ctl.mu.Lock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		g = NewHTTPGroup(ctl)
		ctl.groups[indexKey] = g
	}
	ctl.mu.Unlock()

	return g.Register(proxyName, group, groupKey, routeConfig)
}

func (ctl *HTTPGroupController) UnRegister(proxyName, group string, _ vhost.RouteConfig) {
	indexKey := group
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		return
	}

	isEmpty := g.UnRegister(proxyName)
	if isEmpty {
		delete(ctl.groups, indexKey)
	}
}

type HTTPGroup struct {
	group           string
	groupKey        string
	domain          string
	location        string
	routeByHTTPUser string

	// 按代理名称索引的CreateConnFuncs
	createFuncs map[string]vhost.CreateConnFunc
	pxyNames    []string
	index       uint64
	ctl         *HTTPGroupController
	mu          sync.RWMutex
}

func NewHTTPGroup(ctl *HTTPGroupController) *HTTPGroup {
	return &HTTPGroup{
		createFuncs: make(map[string]vhost.CreateConnFunc),
		pxyNames:    make([]string, 0),
		ctl:         ctl,
	}
}

func (g *HTTPGroup) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.createFuncs) == 0 {
		// 该组中的第一个代理
		tmp := routeConfig // 复制对象
		tmp.CreateConnFn = g.createConn
		tmp.ChooseEndpointFn = g.chooseEndpoint
		tmp.CreateConnByEndpointFn = g.createConnByEndpoint
		err = g.ctl.vhostRouter.Add(routeConfig.Domain, routeConfig.Location, routeConfig.RouteByHTTPUser, &tmp)
		if err != nil {
			return
		}

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
		g.location = routeConfig.Location
		g.routeByHTTPUser = routeConfig.RouteByHTTPUser
	} else {
		if g.group != group || g.domain != routeConfig.Domain ||
			g.location != routeConfig.Location || g.routeByHTTPUser != routeConfig.RouteByHTTPUser {
			err = ErrGroupParamsInvalid
			return
		}
		if g.groupKey != groupKey {
			err = ErrGroupAuthFailed
			return
		}
	}
	if _, ok := g.createFuncs[proxyName]; ok {
		err = ErrProxyRepeated
		return
	}
	g.createFuncs[proxyName] = routeConfig.CreateConnFn
	g.pxyNames = append(g.pxyNames, proxyName)
	return nil
}

func (g *HTTPGroup) UnRegister(proxyName string) (isEmpty bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.createFuncs, proxyName)
	for i, name := range g.pxyNames {
		if name == proxyName {
			g.pxyNames = append(g.pxyNames[:i], g.pxyNames[i+1:]...)
			break
		}
	}

	if len(g.createFuncs) == 0 {
		isEmpty = true
		g.ctl.vhostRouter.Del(g.domain, g.location, g.routeByHTTPUser)
	}
	return
}

func (g *HTTPGroup) createConn(remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	newIndex := atomic.AddUint64(&g.index, 1)

	g.mu.RLock()
	group := g.group
	domain := g.domain
	location := g.location
	routeByHTTPUser := g.routeByHTTPUser
	if len(g.pxyNames) > 0 {
		name := g.pxyNames[int(newIndex)%len(g.pxyNames)]
		f = g.createFuncs[name]
	}
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("http组[%s]、域[%s]、位置[%s]、routeByHTTPUser[%s]没有CreateConnFunc",
			group, domain, location, routeByHTTPUser)
	}

	return f(remoteAddr)
}

func (g *HTTPGroup) chooseEndpoint() (string, error) {
	newIndex := atomic.AddUint64(&g.index, 1)
	name := ""

	g.mu.RLock()
	group := g.group
	domain := g.domain
	location := g.location
	routeByHTTPUser := g.routeByHTTPUser
	if len(g.pxyNames) > 0 {
		name = g.pxyNames[int(newIndex)%len(g.pxyNames)]
	}
	g.mu.RUnlock()

	if name == "" {
		return "", fmt.Errorf("http组[%s]、域[%s]、位置[%s]、routeByHTTPUser[%s]没有健康的端点",
			group, domain, location, routeByHTTPUser)
	}
	return name, nil
}

func (g *HTTPGroup) createConnByEndpoint(endpoint, remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	g.mu.RLock()
	f = g.createFuncs[endpoint]
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("组[%s]中的端点[%s]没有CreateConnFunc", endpoint, g.group)
	}
	return f(remoteAddr)
}
