package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/purpose168/frp/client/api"
	httppkg "github.com/purpose168/frp/pkg/util/http"
)

// Client 是 frp 服务端的 HTTP API 客户端
type Client struct {
	// address 是服务端地址
	address string
	// authUser 是认证用户名
	authUser string
	// authPwd 是认证密码
	authPwd string
}

// New 创建一个新的客户端实例
// 参数 host 是服务端主机名或 IP 地址
// 参数 port 是服务端端口号
func New(host string, port int) *Client {
	return &Client{
		address: net.JoinHostPort(host, strconv.Itoa(port)),
	}
}

// SetAuth 设置认证信息
// 参数 user 是认证用户名
// 参数 pwd 是认证密码
func (c *Client) SetAuth(user, pwd string) {
	c.authUser = user
	c.authPwd = pwd
}

// GetProxyStatus 获取指定代理的状态
// 参数 ctx 是上下文
// 参数 name 是代理名称
// 返回代理状态响应和可能的错误
func (c *Client) GetProxyStatus(ctx context.Context, name string) (*api.ProxyStatusResp, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+c.address+"/api/status", nil)
	if err != nil {
		return nil, err
	}
	content, err := c.do(req)
	if err != nil {
		return nil, err
	}
	allStatus := make(api.StatusResp)
	if err = json.Unmarshal([]byte(content), &allStatus); err != nil {
		return nil, fmt.Errorf("解析 HTTP 响应错误：%s", strings.TrimSpace(content))
	}
	for _, pss := range allStatus {
		for _, ps := range pss {
			if ps.Name == name {
				return &ps, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到代理状态")
}

// GetAllProxyStatus 获取所有代理的状态
// 参数 ctx 是上下文
// 返回所有代理状态响应和可能的错误
func (c *Client) GetAllProxyStatus(ctx context.Context) (api.StatusResp, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+c.address+"/api/status", nil)
	if err != nil {
		return nil, err
	}
	content, err := c.do(req)
	if err != nil {
		return nil, err
	}
	allStatus := make(api.StatusResp)
	if err = json.Unmarshal([]byte(content), &allStatus); err != nil {
		return nil, fmt.Errorf("解析 HTTP 响应错误：%s", strings.TrimSpace(content))
	}
	return allStatus, nil
}

// Reload 重新加载配置
// 参数 ctx 是上下文
// 参数 strictMode 是否启用严格模式
// 返回可能的错误
func (c *Client) Reload(ctx context.Context, strictMode bool) error {
	v := url.Values{}
	if strictMode {
		v.Set("strictConfig", "true")
	}
	queryStr := ""
	if len(v) > 0 {
		queryStr = "?" + v.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+c.address+"/api/reload"+queryStr, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

// Stop 停止服务端
// 参数 ctx 是上下文
// 返回可能的错误
func (c *Client) Stop(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", "http://"+c.address+"/api/stop", nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

// GetConfig 获取当前配置
// 参数 ctx 是上下文
// 返回配置内容和可能的错误
func (c *Client) GetConfig(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+c.address+"/api/config", nil)
	if err != nil {
		return "", err
	}
	return c.do(req)
}

// UpdateConfig 更新配置
// 参数 ctx 是上下文
// 参数 content 是新的配置内容
// 返回可能的错误
func (c *Client) UpdateConfig(ctx context.Context, content string) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", "http://"+c.address+"/api/config", strings.NewReader(content))
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

// setAuthHeader 设置认证头部
// 参数 req 是 HTTP 请求
func (c *Client) setAuthHeader(req *http.Request) {
	if c.authUser != "" || c.authPwd != "" {
		req.Header.Set("Authorization", httppkg.BasicAuth(c.authUser, c.authPwd))
	}
}

// do 执行 HTTP 请求并返回响应内容
// 参数 req 是 HTTP 请求
// 返回响应内容和可能的错误
func (c *Client) do(req *http.Request) (string, error) {
	c.setAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 状态码 [%d]", resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
