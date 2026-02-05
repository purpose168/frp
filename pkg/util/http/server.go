// Copyright 2023 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/fatedier/frp/assets"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

var (
	// 默认读取超时时间
	defaultReadTimeout = 60 * time.Second
	// 默认写入超时时间
	defaultWriteTimeout = 60 * time.Second
)

// Server 是 HTTP 服务器
type Server struct {
	// addr 是服务器监听地址
	addr string
	// ln 是网络监听器
	ln net.Listener
	// tlsCfg 是 TLS 配置
	tlsCfg *tls.Config

	// router 是路由器
	router *mux.Router
	// hs 是 HTTP 服务器
	hs *http.Server

	// authMiddleware 是认证中间件
	authMiddleware mux.MiddlewareFunc
}

// NewServer 创建新的 HTTP 服务器
// 参数 cfg 是 Web 服务器配置
// 返回服务器实例和可能的错误
func NewServer(cfg v1.WebServerConfig) (*Server, error) {
	assets.Load(cfg.AssetsDir)

	addr := net.JoinHostPort(cfg.Addr, strconv.Itoa(cfg.Port))
	if addr == ":" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	hs := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}
	s := &Server{
		addr:   addr,
		ln:     ln,
		hs:     hs,
		router: router,
	}
	if cfg.PprofEnable {
		s.registerPprofHandlers()
	}
	if cfg.TLS != nil {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, err
		}
		s.tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	s.authMiddleware = netpkg.NewHTTPAuthMiddleware(cfg.User, cfg.Password).SetAuthFailDelay(200 * time.Millisecond).Middleware
	return s, nil
}

// Address 返回服务器地址
func (s *Server) Address() string {
	return s.addr
}

// Run 运行服务器
// 返回可能的错误
func (s *Server) Run() error {
	ln := s.ln
	if s.tlsCfg != nil {
		ln = tls.NewListener(ln, s.tlsCfg)
	}
	return s.hs.Serve(ln)
}

// Close 关闭服务器
// 返回可能的错误
func (s *Server) Close() error {
	return s.hs.Close()
}

// RouterRegisterHelper 是路由注册辅助器
type RouterRegisterHelper struct {
	// Router 是路由器
	Router *mux.Router
	// AssetsFS 是资源文件系统
	AssetsFS http.FileSystem
	// AuthMiddleware 是认证中间件
	AuthMiddleware mux.MiddlewareFunc
}

// RouteRegister 注册路由
// 参数 register 是路由注册函数
func (s *Server) RouteRegister(register func(helper *RouterRegisterHelper)) {
	register(&RouterRegisterHelper{
		Router:         s.router,
		AssetsFS:       assets.FileSystem,
		AuthMiddleware: s.authMiddleware,
	})
}

// registerPprofHandlers 注册 pprof 处理器
func (s *Server) registerPprofHandlers() {
	s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s.router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
}
