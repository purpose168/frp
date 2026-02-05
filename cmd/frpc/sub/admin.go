// 版权所有 2023 frp 作者
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

package sub

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/purpose168/frp/pkg/config"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	clientsdk "github.com/purpose168/frp/pkg/sdk/client"
)

var adminAPITimeout = 30 * time.Second

// init 初始化管理命令
func init() {
	commands := []struct {
		name        string
		description string
		handler     func(*v1.ClientCommonConfig) error
	}{
		{"reload", "热重载 frpc 配置", ReloadHandler},
		{"status", "所有代理状态概览", StatusHandler},
		{"stop", "停止正在运行的 frpc", StopHandler},
	}

	for _, cmdConfig := range commands {
		cmd := NewAdminCommand(cmdConfig.name, cmdConfig.description, cmdConfig.handler)
		cmd.Flags().DurationVar(&adminAPITimeout, "api-timeout", adminAPITimeout, "管理 API 调用的超时时间")
		rootCmd.AddCommand(cmd)
	}
}

// NewAdminCommand 创建新的管理命令
func NewAdminCommand(name, short string, handler func(*v1.ClientCommonConfig) error) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _, _, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if cfg.WebServer.Port <= 0 {
				fmt.Println("如果要使用此功能，应设置 Web 服务器端口")
				os.Exit(1)
			}

			if err := handler(cfg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
}

// ReloadHandler 处理重载命令
func ReloadHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	ctx, cancel := context.WithTimeout(context.Background(), adminAPITimeout)
	defer cancel()
	if err := client.Reload(ctx, strictConfigMode); err != nil {
		return err
	}
	fmt.Println("重载成功")
	return nil
}

// StatusHandler 处理状态命令
func StatusHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	ctx, cancel := context.WithTimeout(context.Background(), adminAPITimeout)
	defer cancel()
	res, err := client.GetAllProxyStatus(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("代理状态...\n\n")
	for _, typ := range proxyTypes {
		arrs := res[string(typ)]
		if len(arrs) == 0 {
			continue
		}

		fmt.Println(strings.ToUpper(string(typ)))
		tbl := table.New("名称", "状态", "本地地址", "插件", "远程地址", "错误")
		for _, ps := range arrs {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	return nil
}

// StopHandler 处理停止命令
func StopHandler(clientCfg *v1.ClientCommonConfig) error {
	client := clientsdk.New(clientCfg.WebServer.Addr, clientCfg.WebServer.Port)
	client.SetAuth(clientCfg.WebServer.User, clientCfg.WebServer.Password)
	ctx, cancel := context.WithTimeout(context.Background(), adminAPITimeout)
	defer cancel()
	if err := client.Stop(ctx); err != nil {
		return err
	}
	fmt.Println("停止成功")
	return nil
}
