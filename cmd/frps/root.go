// 版权所有 2018 fatedier, fatedier@gmail.com
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"基础分发的，不附带任何明示或暗示的担保或条件。
// 有关许可证下特定语言的权限和限制，请参阅许可证。

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server"
)

var (
	cfgFile          string
	showVersion      bool
	strictConfigMode bool
	allowUnsafe      []string

	serverCfg v1.ServerConfig
)

// init 初始化 root 命令的持久化标志
func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "frps 配置文件")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "frps 版本")
	rootCmd.PersistentFlags().BoolVarP(&strictConfigMode, "strict_config", "", true, "严格配置解析模式，未知字段将导致错误")
	rootCmd.PersistentFlags().StringSliceVarP(&allowUnsafe, "allow-unsafe", "", []string{},
		fmt.Sprintf("允许的不安全功能，以下一个或多个: %s", strings.Join(security.ServerUnsafeFeatures, ", ")))

	config.RegisterServerConfigFlags(rootCmd, &serverCfg)
}

// rootCmd 是 frps 的根命令
var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps 是 frp 的服务器端 (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		var (
			svrCfg         *v1.ServerConfig
			isLegacyFormat bool
			err            error
		)
		if cfgFile != "" {
			svrCfg, isLegacyFormat, err = config.LoadServerConfig(cfgFile, strictConfigMode)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if isLegacyFormat {
				fmt.Printf("警告: ini 格式已弃用，未来将移除支持，" +
					"请使用 yaml/json/toml 格式替代!\n")
			}
		} else {
			if err := serverCfg.Complete(); err != nil {
				fmt.Printf("完成服务器配置失败: %v\n", err)
				os.Exit(1)
			}
			svrCfg = &serverCfg
		}

		unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)
		validator := validation.NewConfigValidator(unsafeFeatures)
		warning, err := validator.ValidateServerConfig(svrCfg)
		if warning != nil {
			fmt.Printf("警告: %v\n", warning)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := runServer(svrCfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

// Execute 执行 root 命令
func Execute() {
	rootCmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runServer 运行 frps 服务器
func runServer(cfg *v1.ServerConfig) (err error) {
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	if cfgFile != "" {
		log.Infof("frps 使用配置文件: %s", cfgFile)
	} else {
		log.Infof("frps 使用命令行参数进行配置")
	}

	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	log.Infof("frps 启动成功")
	svr.Run(context.Background())
	return
}
