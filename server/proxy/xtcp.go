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
	"fmt"
	"reflect"
	"sync"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
)

// init 注册XTCP代理工厂
func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.XTCPProxyConfig{}), NewXTCPProxy)
}

// XTCPProxy XTCP代理
// 实现了Proxy接口，用于处理XTCP代理请求
type XTCPProxy struct {
	*BaseProxy
	// cfg XTCP代理配置
	cfg *v1.XTCPProxyConfig

	// closeCh 关闭通道
	closeCh chan struct{}
	// closeOnce 确保只关闭一次
	closeOnce sync.Once
}

// NewXTCPProxy 创建一个新的XTCP代理
// 参数baseProxy是基础代理
func NewXTCPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.XTCPProxyConfig)
	if !ok {
		return nil
	}
	return &XTCPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		closeCh:   make(chan struct{}),
	}
}

// Run 启动XTCP代理
// 返回值是远程地址和可能的错误
func (pxy *XTCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	if pxy.rc.NatHoleController == nil {
		err = fmt.Errorf("frps不支持xtcp")
		return
	}
	allowUsers := pxy.cfg.AllowUsers
	// 如果allowUsers为空，只允许同一用户的代理访问
	if len(allowUsers) == 0 {
		allowUsers = []string{pxy.GetUserInfo().User}
	}
	sidCh, err := pxy.rc.NatHoleController.ListenClient(pxy.GetName(), pxy.cfg.Secretkey, allowUsers)
	if err != nil {
		return "", err
	}
	go func() {
		for {
			select {
			case <-pxy.closeCh:
				return
			case sid := <-sidCh:
				workConn, errRet := pxy.GetWorkConnFromPool(nil, nil)
				if errRet != nil {
					continue
				}
				m := &msg.NatHoleSid{
					Sid: sid,
				}
				errRet = msg.WriteMsg(workConn, m)
				if errRet != nil {
					xl.Warnf("写入nat hole sid包错误, %v", errRet)
				}
				workConn.Close()
			}
		}
	}()
	return
}

// Close 关闭XTCP代理
func (pxy *XTCPProxy) Close() {
	pxy.closeOnce.Do(func() {
		pxy.BaseProxy.Close()
		pxy.rc.NatHoleController.CloseClient(pxy.GetName())
		close(pxy.closeCh)
	})
}
