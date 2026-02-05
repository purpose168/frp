// Copyright 2019 fatedier, fatedier@gmail.com
//
// 依据 Apache License, Version 2.0 许可证授权；
// 除非符合许可证的要求，否则您不能使用此文件。
// 您可以在以下网址获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则软件
// 根据许可证分发是基于“按原样”基础，
// 不附带任何明示或暗示的担保或条件。
// 请参阅许可证中有关管理权限和
// 限制的特定语言。

package proxy

import (
	"reflect"

	v1 "github.com/purpose168/frp/pkg/config/v1"
)

// init 注册SUDP代理工厂
func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.SUDPProxyConfig{}), NewSUDPProxy)
}

// SUDPProxy 安全UDP代理
// 实现了Proxy接口，用于处理安全UDP代理请求
type SUDPProxy struct {
	*BaseProxy
	// cfg SUDP代理配置
	cfg *v1.SUDPProxyConfig
}

// NewSUDPProxy 创建一个新的SUDP代理
// 参数baseProxy是基础代理
func NewSUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.SUDPProxyConfig)
	if !ok {
		return nil
	}
	return &SUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

// Run 启动SUDP代理
// 返回值是远程地址和可能的错误
func (pxy *SUDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	allowUsers := pxy.cfg.AllowUsers
	// 如果allowUsers为空，只允许同一用户的代理访问
	if len(allowUsers) == 0 {
		allowUsers = []string{pxy.GetUserInfo().User}
	}
	listener, errRet := pxy.rc.VisitorManager.Listen(pxy.GetName(), pxy.cfg.Secretkey, allowUsers)
	if errRet != nil {
		err = errRet
		return
	}
	pxy.listeners = append(pxy.listeners, listener)
	xl.Infof("sudp代理自定义监听成功")

	pxy.startCommonTCPListenersHandler()
	return
}

// Close 关闭SUDP代理
func (pxy *SUDPProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.VisitorManager.CloseListener(pxy.GetName())
}
