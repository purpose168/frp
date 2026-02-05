// Copyright 2018 fatedier, fatedier@gmail.com
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

package sub

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/purpose168/frp/client"
	"github.com/purpose168/frp/pkg/config"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/config/v1/validation"
	"github.com/purpose168/frp/pkg/policy/featuregate"
	"github.com/purpose168/frp/pkg/policy/security"
	"github.com/purpose168/frp/pkg/util/log"
	"github.com/purpose168/frp/pkg/util/version"
)

var (
	cfgFile          string
	cfgDir           string
	showVersion      bool
	strictConfigMode bool
	allowUnsafe      []string
)

// init 初始化 root 命令的持久化标志
func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./frpc.ini", "frpc 配置文件")
	rootCmd.PersistentFlags().StringVarP(&cfgDir, "config_dir", "", "", "配置目录，为配置目录中的每个文件运行一个 frpc 服务")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "frpc 版本")
	rootCmd.PersistentFlags().BoolVarP(&strictConfigMode, "strict_config", "", true, "严格配置解析模式，未知字段将导致错误")

	rootCmd.PersistentFlags().StringSliceVarP(&allowUnsafe, "allow-unsafe", "", []string{},
		fmt.Sprintf("允许的不安全功能，以下一个或多个: %s", strings.Join(security.ClientUnsafeFeatures, ", ")))
}

// rootCmd 是 frpc 的根命令
var rootCmd = &cobra.Command{
	Use:   "frpc",
	Short: "frpc 是 frp 的客户端 (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)

		// 如果 cfgDir 不为空，则为 cfgDir 中的每个配置文件运行多个 frpc 服务
		// 注意：这仅用于测试，不保证稳定性
		if cfgDir != "" {
			_ = runMultipleClients(cfgDir, unsafeFeatures)
			return nil
		}

		// 此处不显示命令用法
		err := runClient(cfgFile, unsafeFeatures)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

// runMultipleClients 为配置目录中的每个配置文件运行多个 frpc 客户端
func runMultipleClients(cfgDir string, unsafeFeatures *security.UnsafeFeatures) error {
	var wg sync.WaitGroup
	err := filepath.WalkDir(cfgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		wg.Add(1)
		time.Sleep(time.Millisecond)
		go func() {
			defer wg.Done()
			err := runClient(path, unsafeFeatures)
			if err != nil {
				fmt.Printf("配置文件 [%s] 的 frpc 服务错误\n", path)
			}
		}()
		return nil
	})
	wg.Wait()
	return err
}

// Execute 执行 root 命令
func Execute() {
	rootCmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// handleTermSignal 处理终止信号
func handleTermSignal(svr *client.Service) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	svr.GracefulClose(500 * time.Millisecond)
}

// runClient 运行单个客户端
func runClient(cfgFilePath string, unsafeFeatures *security.UnsafeFeatures) error {
	cfg, proxyCfgs, visitorCfgs, isLegacyFormat, err := config.LoadClientConfig(cfgFilePath, strictConfigMode)
	if err != nil {
		return err
	}
	if isLegacyFormat {
		fmt.Printf("警告: ini 格式已弃用，未来将移除支持，" +
			"请使用 yaml/json/toml 格式替代!\n")
	}

	if len(cfg.FeatureGates) > 0 {
		if err := featuregate.SetFromMap(cfg.FeatureGates); err != nil {
			return err
		}
	}

	warning, err := validation.ValidateAllClientConfig(cfg, proxyCfgs, visitorCfgs, unsafeFeatures)
	if warning != nil {
		fmt.Printf("警告: %v\n", warning)
	}
	if err != nil {
		return err
	}

	return startService(cfg, proxyCfgs, visitorCfgs, unsafeFeatures, cfgFilePath)
}

// startService 启动 frpc 服务
func startService(
	cfg *v1.ClientCommonConfig,
	proxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	unsafeFeatures *security.UnsafeFeatures,
	cfgFile string,
) error {
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)

	if cfgFile != "" {
		log.Infof("为配置文件 [%s] 启动 frpc 服务", cfgFile)
		defer log.Infof("配置文件 [%s] 的 frpc 服务已停止", cfgFile)
	}
	svr, err := client.NewService(client.ServiceOptions{
		Common:         cfg,
		ProxyCfgs:      proxyCfgs,
		VisitorCfgs:    visitorCfgs,
		UnsafeFeatures: unsafeFeatures,
		ConfigFilePath: cfgFile,
	})
	if err != nil {
		return err
	}

	shouldGracefulClose := cfg.Transport.Protocol == "kcp" || cfg.Transport.Protocol == "quic"
	// 如果使用 kcp 或 quic，捕获退出信号
	if shouldGracefulClose {
		go handleTermSignal(svr)
	}
	return svr.Run(context.Background())
}
