// 版权所有 2017 fatedier, fatedier@gmail.com
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
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/msg"
	"github.com/purpose168/frp/pkg/util/util"
	"github.com/purpose168/frp/pkg/util/xlog"
)

// STCPVisitor STCP 访问者结构
type STCPVisitor struct {
	*BaseVisitor

	cfg *v1.STCPVisitorConfig
}

// Run 运行 STCP 访问者
func (sv *STCPVisitor) Run() (err error) {
	if sv.cfg.BindPort > 0 {
		sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return
		}
		go sv.worker()
	}

	go sv.internalConnWorker()

	if sv.plugin != nil {
		sv.plugin.Start()
	}
	return
}

// Close 关闭 STCP 访问者
func (sv *STCPVisitor) Close() {
	sv.BaseVisitor.Close()
}

// worker 处理本地连接
func (sv *STCPVisitor) worker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			xl.Warnf("stcp 本地监听器已关闭")
			return
		}
		go sv.handleConn(conn)
	}
}

// internalConnWorker 处理内部连接
func (sv *STCPVisitor) internalConnWorker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.internalLn.Accept()
		if err != nil {
			xl.Warnf("stcp 内部监听器已关闭")
			return
		}
		go sv.handleConn(conn)
	}
}

// handleConn 处理用户连接
func (sv *STCPVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	var tunnelErr error
	defer func() {
		// 如果有错误且连接支持 CloseWithError，则使用它
		if tunnelErr != nil {
			if eConn, ok := userConn.(interface{ CloseWithError(error) error }); ok {
				_ = eConn.CloseWithError(tunnelErr)
				return
			}
		}
		userConn.Close()
	}()

	xl.Debugf("获取新的 stcp 用户连接")
	visitorConn, err := sv.helper.ConnectServer()
	if err != nil {
		tunnelErr = err
		return
	}
	defer visitorConn.Close()

	now := time.Now().Unix()
	newVisitorConnMsg := &msg.NewVisitorConn{
		RunID:          sv.helper.RunID(),
		ProxyName:      sv.cfg.ServerName,
		SignKey:        util.GetAuthKey(sv.cfg.SecretKey, now),
		Timestamp:      now,
		UseEncryption:  sv.cfg.Transport.UseEncryption,
		UseCompression: sv.cfg.Transport.UseCompression,
	}
	err = msg.WriteMsg(visitorConn, newVisitorConnMsg)
	if err != nil {
		xl.Warnf("向服务器发送 newVisitorConnMsg 错误: %v", err)
		tunnelErr = err
		return
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	_ = visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		xl.Warnf("获取 newVisitorConnRespMsg 错误: %v", err)
		tunnelErr = err
		return
	}
	_ = visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		xl.Warnf("启动新的访问者连接错误: %s", newVisitorConnRespMsg.Error)
		tunnelErr = fmt.Errorf("%s", newVisitorConnRespMsg.Error)
		return
	}

	var remote io.ReadWriteCloser
	remote = visitorConn
	if sv.cfg.Transport.UseEncryption {
		remote, err = libio.WithEncryption(remote, []byte(sv.cfg.SecretKey))
		if err != nil {
			xl.Errorf("创建加密流错误: %v", err)
			tunnelErr = err
			return
		}
	}

	if sv.cfg.Transport.UseCompression {
		var recycleFn func()
		remote, recycleFn = libio.WithCompressionFromPool(remote)
		defer recycleFn()
	}

	libio.Join(userConn, remote)
}
