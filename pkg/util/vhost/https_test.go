package vhost

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGetHTTPSHostname 测试获取 HTTPS 主机名功能
func TestGetHTTPSHostname(t *testing.T) {
	require := require.New(t)

	// 创建 TCP 监听器
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(err)
	defer l.Close()

	var conn net.Conn
	// 启动 goroutine 接受连接
	go func() {
		conn, _ = l.Accept()
		require.NotNil(conn)
	}()

	// 启动 goroutine 建立 TLS 连接
	go func() {
		time.Sleep(100 * time.Millisecond)
		tls.Dial("tcp", l.Addr().String(), &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "example.com",
		})
	}()

	// 等待连接建立
	time.Sleep(200 * time.Millisecond)
	// 获取 HTTPS 主机名
	_, infos, err := GetHTTPSHostname(conn)
	require.NoError(err)
	// 验证主机名和协议
	require.Equal("example.com", infos["Host"])
	require.Equal("https", infos["Scheme"])
}
