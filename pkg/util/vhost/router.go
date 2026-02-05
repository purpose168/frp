package vhost

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"sync"
)

// ErrRouterConfigConflict 路由配置冲突错误
var ErrRouterConfigConflict = errors.New("路由配置冲突")

// routerByHTTPUser 按 HTTP 用户分组的路由映射
type routerByHTTPUser map[string][]*Router

// Routers 路由集合
type Routers struct {
	indexByDomain map[string]routerByHTTPUser

	mutex sync.RWMutex
}

// Router 路由
type Router struct {
	domain   string
	location string
	httpUser string

	// 在这里存储任何对象
	payload any
}

// NewRouters 创建一个新的路由集合
func NewRouters() *Routers {
	return &Routers{
		indexByDomain: make(map[string]routerByHTTPUser),
	}
}

// Add 添加路由到路由集合
func (r *Routers) Add(domain, location, httpUser string, payload any) error {
	// 将域名转换为小写
	domain = strings.ToLower(domain)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查路由是否已存在
	if _, exist := r.exist(domain, location, httpUser); exist {
		return ErrRouterConfigConflict
	}

	// 获取或创建按 HTTP 用户分组的路由映射
	routersByHTTPUser, found := r.indexByDomain[domain]
	if !found {
		routersByHTTPUser = make(map[string][]*Router)
	}
	// 获取或创建路由列表
	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		vrs = make([]*Router, 0, 1)
	}

	// 创建新路由
	vr := &Router{
		domain:   domain,
		location: location,
		httpUser: httpUser,
		payload:  payload,
	}
	// 添加到路由列表
	vrs = append(vrs, vr)

	// 按位置降序排序
	slices.SortFunc(vrs, func(a, b *Router) int {
		return -cmp.Compare(a.location, b.location)
	})

	// 更新路由映射
	routersByHTTPUser[httpUser] = vrs
	r.indexByDomain[domain] = routersByHTTPUser
	return nil
}

// Del 从路由集合中删除路由
func (r *Routers) Del(domain, location, httpUser string) {
	// 将域名转换为小写
	domain = strings.ToLower(domain)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 获取按 HTTP 用户分组的路由映射
	routersByHTTPUser, found := r.indexByDomain[domain]
	if !found {
		return
	}

	// 获取路由列表
	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}
	// 创建新的路由列表，排除要删除的路由
	newVrs := make([]*Router, 0)
	for _, vr := range vrs {
		if vr.location != location {
			newVrs = append(newVrs, vr)
		}
	}
	// 更新路由映射
	routersByHTTPUser[httpUser] = newVrs
}

// Get 获取路由
func (r *Routers) Get(host, path, httpUser string) (vr *Router, exist bool) {
	// 将主机名转换为小写
	host = strings.ToLower(host)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 获取按 HTTP 用户分组的路由映射
	routersByHTTPUser, found := r.indexByDomain[host]
	if !found {
		return
	}

	// 获取路由列表
	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}

	// 查找匹配路径的路由
	for _, vr = range vrs {
		if strings.HasPrefix(path, vr.location) {
			return vr, true
		}
	}
	return
}

// exist 检查路由是否存在
func (r *Routers) exist(host, path, httpUser string) (route *Router, exist bool) {
	// 获取按 HTTP 用户分组的路由映射
	routersByHTTPUser, found := r.indexByDomain[host]
	if !found {
		return
	}
	// 获取路由列表
	routers, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}

	// 查找完全匹配路径的路由
	for _, route = range routers {
		if path == route.location {
			return route, true
		}
	}
	return
}
