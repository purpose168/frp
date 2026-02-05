// Copyright 2016 fatedier, fatedier@gmail.com
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

package net

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/fatedier/golib/crypto"
	quic "github.com/quic-go/quic-go"

	"github.com/fatedier/frp/pkg/util/xlog"
)

// ContextGetter 是上下文获取器接口
type ContextGetter interface {
	// Context 返回上下文
	Context() context.Context
}

// ContextSetter 是上下文设置器接口
type ContextSetter interface {
	// WithContext 设置上下文
	WithContext(ctx context.Context)
}

// NewLogFromConn 从连接创建日志记录器
// 参数 conn 是网络连接
// 返回日志记录器
func NewLogFromConn(conn net.Conn) *xlog.Logger {
	if c, ok := conn.(ContextGetter); ok {
		return xlog.FromContextSafe(c.Context())
	}
	return xlog.New()
}

// NewContextFromConn 从连接创建上下文
// 参数 conn 是网络连接
// 返回上下文
func NewContextFromConn(conn net.Conn) context.Context {
	if c, ok := conn.(ContextGetter); ok {
		return c.Context()
	}
	return context.Background()
}

// ContextConn 是带上下文的连接
type ContextConn struct {
	net.Conn

	// ctx 是上下文
	ctx context.Context
}

// NewContextConn 创建带上下文的连接
// 参数 ctx 是上下文
// 参数 c 是网络连接
// 返回带上下文的连接
func NewContextConn(ctx context.Context, c net.Conn) *ContextConn {
	return &ContextConn{
		Conn: c,
		ctx:  ctx,
	}
}

// WithContext 设置上下文
// 参数 ctx 是上下文
func (c *ContextConn) WithContext(ctx context.Context) {
	c.ctx = ctx
}

// Context 返回上下文
func (c *ContextConn) Context() context.Context {
	return c.ctx
}

// WrapReadWriteCloserConn 包装读写关闭器为连接
type WrapReadWriteCloserConn struct {
	io.ReadWriteCloser

	// underConn 是底层连接
	underConn net.Conn

	// remoteAddr 是远程地址
	remoteAddr net.Addr
}

// WrapReadWriteCloserToConn 包装读写关闭器为连接
// 参数 rwc 是读写关闭器
// 参数 underConn 是底层连接
// 返回包装后的连接
func WrapReadWriteCloserToConn(rwc io.ReadWriteCloser, underConn net.Conn) *WrapReadWriteCloserConn {
	return &WrapReadWriteCloserConn{
		ReadWriteCloser: rwc,
		underConn:       underConn,
	}
}

// LocalAddr 返回本地地址
func (conn *WrapReadWriteCloserConn) LocalAddr() net.Addr {
	if conn.underConn != nil {
		return conn.underConn.LocalAddr()
	}
	return (*net.TCPAddr)(nil)
}

// SetRemoteAddr 设置远程地址
// 参数 addr 是远程地址
func (conn *WrapReadWriteCloserConn) SetRemoteAddr(addr net.Addr) {
	conn.remoteAddr = addr
}

// RemoteAddr 返回远程地址
func (conn *WrapReadWriteCloserConn) RemoteAddr() net.Addr {
	if conn.remoteAddr != nil {
		return conn.remoteAddr
	}
	if conn.underConn != nil {
		return conn.underConn.RemoteAddr()
	}
	return (*net.TCPAddr)(nil)
}

// SetDeadline 设置读写截止时间
// 参数 t 是截止时间
// 返回可能的错误
func (conn *WrapReadWriteCloserConn) SetDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("不支持设置截止时间")}
}

// SetReadDeadline 设置读截止时间
// 参数 t 是截止时间
// 返回可能的错误
func (conn *WrapReadWriteCloserConn) SetReadDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetReadDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("不支持设置截止时间")}
}

// SetWriteDeadline 设置写截止时间
// 参数 t 是截止时间
// 返回可能的错误
func (conn *WrapReadWriteCloserConn) SetWriteDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetWriteDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("不支持设置截止时间")}
}

// CloseNotifyConn 是带关闭通知的连接
type CloseNotifyConn struct {
	net.Conn

	// closeFlag 是关闭标志，1 表示已关闭
	// 1 means closed
	closeFlag int32

	// closeFn 是关闭回调函数
	closeFn func(error)
}

// WrapCloseNotifyConn 包装连接为带关闭通知的连接
// closeFn 只会被调用一次，参数为 nil 表示 Close() 被调用，非 nil 表示 CloseWithError() 被调用
// 参数 c 是网络连接
// 参数 closeFn 是关闭回调函数
// 返回带关闭通知的连接
func WrapCloseNotifyConn(c net.Conn, closeFn func(error)) *CloseNotifyConn {
	return &CloseNotifyConn{
		Conn:    c,
		closeFn: closeFn,
	}
}

// Close 关闭连接
// 返回可能的错误
func (cc *CloseNotifyConn) Close() (err error) {
	pflag := atomic.SwapInt32(&cc.closeFlag, 1)
	if pflag == 0 {
		err = cc.Conn.Close()
		if cc.closeFn != nil {
			cc.closeFn(nil)
		}
	}
	return
}

// CloseWithError 关闭连接并将错误传递给关闭回调
// 参数 err 是错误信息
// 返回可能的错误
func (cc *CloseNotifyConn) CloseWithError(err error) error {
	pflag := atomic.SwapInt32(&cc.closeFlag, 1)
	if pflag == 0 {
		closeErr := cc.Conn.Close()
		if cc.closeFn != nil {
			cc.closeFn(err)
		}
		return closeErr
	}
	return nil
}

// StatsConn 是带统计信息的连接
type StatsConn struct {
	net.Conn

	// closed 是关闭标志，1 表示已关闭
	closed int64 // 1 means closed
	// totalRead 是总读取字节数
	totalRead int64
	// totalWrite 是总写入字节数
	totalWrite int64
	// statsFunc 是统计回调函数
	statsFunc func(totalRead, totalWrite int64)
}

// WrapStatsConn 包装连接为带统计信息的连接
// 参数 conn 是网络连接
// 参数 statsFunc 是统计回调函数
// 返回带统计信息的连接
func WrapStatsConn(conn net.Conn, statsFunc func(total, totalWrite int64)) *StatsConn {
	return &StatsConn{
		Conn:      conn,
		statsFunc: statsFunc,
	}
}

// Read 读取数据并统计
// 参数 p 是读取缓冲区
// 返回读取的字节数和可能的错误
func (statsConn *StatsConn) Read(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Read(p)
	statsConn.totalRead += int64(n)
	return
}

// Write 写入数据并统计
// 参数 p 是要写入的数据
// 返回写入的字节数和可能的错误
func (statsConn *StatsConn) Write(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Write(p)
	statsConn.totalWrite += int64(n)
	return
}

// Close 关闭连接并调用统计回调
// 返回可能的错误
func (statsConn *StatsConn) Close() (err error) {
	old := atomic.SwapInt64(&statsConn.closed, 1)
	if old != 1 {
		err = statsConn.Conn.Close()
		if statsConn.statsFunc != nil {
			statsConn.statsFunc(statsConn.totalRead, statsConn.totalWrite)
		}
	}
	return
}

// wrapQuicStream 包装 QUIC 流为网络连接
type wrapQuicStream struct {
	*quic.Stream
	// c 是 QUIC 连接
	c *quic.Conn
}

// QuicStreamToNetConn 将 QUIC 流转换为网络连接
// 参数 s 是 QUIC 流
// 参数 c 是 QUIC 连接
// 返回网络连接
func QuicStreamToNetConn(s *quic.Stream, c *quic.Conn) net.Conn {
	return &wrapQuicStream{
		Stream: s,
		c:      c,
	}
}

// LocalAddr 返回本地地址
func (conn *wrapQuicStream) LocalAddr() net.Addr {
	if conn.c != nil {
		return conn.c.LocalAddr()
	}
	return (*net.TCPAddr)(nil)
}

// RemoteAddr 返回远程地址
func (conn *wrapQuicStream) RemoteAddr() net.Addr {
	if conn.c != nil {
		return conn.c.RemoteAddr()
	}
	return (*net.TCPAddr)(nil)
}

// Close 关闭连接
func (conn *wrapQuicStream) Close() error {
	conn.CancelRead(0)
	return conn.Stream.Close()
}

// NewCryptoReadWriter 创建加密读写器
// 参数 rw 是读写器
// 参数 key 是加密密钥
// 返回加密读写器和可能的错误
func NewCryptoReadWriter(rw io.ReadWriter, key []byte) (io.ReadWriter, error) {
	encReader := crypto.NewReader(rw, key)
	encWriter, err := crypto.NewWriter(rw, key)
	if err != nil {
		return nil, err
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: encReader,
		Writer: encWriter,
	}, nil
}
