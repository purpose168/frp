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

package ssh

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/transport"
	"github.com/purpose168/frp/pkg/util/log"
	netpkg "github.com/purpose168/frp/pkg/util/net"
)

// Gateway 是 SSH 网关，用于管理 SSH 隧道连接
type Gateway struct {
	// bindPort 是绑定的端口号
	bindPort int
	// ln 是网络监听器
	ln net.Listener

	// peerServerListener 是对端服务器监听器
	peerServerListener *netpkg.InternalListener

	// sshConfig 是 SSH 服务器配置
	sshConfig *ssh.ServerConfig
}

// NewGateway 创建一个新的 SSH 网关
// 参数 cfg 是 SSH 隧道网关配置
// 参数 bindAddr 是绑定的地址
// 参数 peerServerListener 是对端服务器监听器
// 返回网关实例和可能的错误
func NewGateway(
	cfg v1.SSHTunnelGateway, bindAddr string,
	peerServerListener *netpkg.InternalListener,
) (*Gateway, error) {
	sshConfig := &ssh.ServerConfig{}

	// 私钥处理
	var (
		privateKeyBytes []byte
		err             error
	)
	if cfg.PrivateKeyFile != "" {
		privateKeyBytes, err = os.ReadFile(cfg.PrivateKeyFile)
	} else {
		if cfg.AutoGenPrivateKeyPath != "" {
			privateKeyBytes, _ = os.ReadFile(cfg.AutoGenPrivateKeyPath)
		}
		if len(privateKeyBytes) == 0 {
			privateKeyBytes, err = transport.NewRandomPrivateKey()
			if err == nil && cfg.AutoGenPrivateKeyPath != "" {
				err = os.WriteFile(cfg.AutoGenPrivateKeyPath, privateKeyBytes, 0o600)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	sshConfig.AddHostKey(privateKey)

	sshConfig.NoClientAuth = cfg.AuthorizedKeysFile == ""
	sshConfig.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		authorizedKeysMap, err := loadAuthorizedKeysFromFile(cfg.AuthorizedKeysFile)
		if err != nil {
			log.Errorf("加载授权密钥文件错误：%v", err)
			return nil, fmt.Errorf("内部错误")
		}

		user, ok := authorizedKeysMap[string(key.Marshal())]
		if !ok {
			return nil, fmt.Errorf("远程地址 %q 的公钥未知", conn.RemoteAddr())
		}
		return &ssh.Permissions{
			Extensions: map[string]string{
				"user": user,
			},
		}, nil
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(bindAddr, strconv.Itoa(cfg.BindPort)))
	if err != nil {
		return nil, err
	}
	return &Gateway{
		bindPort:           cfg.BindPort,
		ln:                 ln,
		peerServerListener: peerServerListener,
		sshConfig:          sshConfig,
	}, nil
}

// Run 运行网关，接受并处理传入的连接
func (g *Gateway) Run() {
	for {
		conn, err := g.ln.Accept()
		if err != nil {
			return
		}
		go g.handleConn(conn)
	}
}

// Close 关闭网关
func (g *Gateway) Close() error {
	return g.ln.Close()
}

// handleConn 处理传入的连接
// 参数 conn 是传入的网络连接
func (g *Gateway) handleConn(conn net.Conn) {
	defer conn.Close()

	ts, err := NewTunnelServer(conn, g.sshConfig, g.peerServerListener)
	if err != nil {
		return
	}
	if err := ts.Run(); err != nil {
		log.Errorf("SSH 隧道服务器运行错误：%v", err)
	}
}

// loadAuthorizedKeysFromFile 从文件加载授权密钥
// 参数 path 是授权密钥文件路径
// 返回公钥到用户名的映射和可能的错误
func loadAuthorizedKeysFromFile(path string) (map[string]string, error) {
	authorizedKeysMap := make(map[string]string) // 值是用户名
	authorizedKeysBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for len(authorizedKeysBytes) > 0 {
		pubKey, comment, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return nil, err
		}

		authorizedKeysMap[string(pubKey.Marshal())] = strings.TrimSpace(comment)
		authorizedKeysBytes = rest
	}
	return authorizedKeysMap, nil
}
