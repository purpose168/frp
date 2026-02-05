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
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
)

var proxyTypes = []v1.ProxyType{
	v1.ProxyTypeTCP,
	v1.ProxyTypeUDP,
	v1.ProxyTypeTCPMUX,
	v1.ProxyTypeHTTP,
	v1.ProxyTypeHTTPS,
	v1.ProxyTypeSTCP,
	v1.ProxyTypeSUDP,
	v1.ProxyTypeXTCP,
}

var visitorTypes = []v1.VisitorType{
	v1.VisitorTypeSTCP,
	v1.VisitorTypeSUDP,
	v1.VisitorTypeXTCP,
}

// init 初始化代理命令
func init() {
	for _, typ := range proxyTypes {
		c := v1.NewProxyConfigurerByType(typ)
		if c == nil {
			panic("代理类型: " + typ + " 不支持")
		}
		clientCfg := v1.ClientCommonConfig{}
		cmd := NewProxyCommand(string(typ), c, &clientCfg)
		config.RegisterClientCommonConfigFlags(cmd, &clientCfg)
		config.RegisterProxyFlags(cmd, c)

		// 为访问者添加子命令
		if slices.Contains(visitorTypes, v1.VisitorType(typ)) {
			vc := v1.NewVisitorConfigurerByType(v1.VisitorType(typ))
			if vc == nil {
				panic("访问者类型: " + typ + " 不支持")
			}
			visitorCmd := NewVisitorCommand(string(typ), vc, &clientCfg)
			config.RegisterVisitorFlags(visitorCmd, vc)
			cmd.AddCommand(visitorCmd)
		}
		rootCmd.AddCommand(cmd)
	}
}

// NewProxyCommand 创建新的代理命令
func NewProxyCommand(name string, c v1.ProxyConfigurer, clientCfg *v1.ClientCommonConfig) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("使用单个 %s 代理运行 frpc", name),
		Run: func(cmd *cobra.Command, args []string) {
			if err := clientCfg.Complete(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)
			validator := validation.NewConfigValidator(unsafeFeatures)
			if _, err := validator.ValidateClientCommonConfig(clientCfg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			c.Complete(clientCfg.User)
			c.GetBaseConfig().Type = name
			if err := validation.ValidateProxyConfigurerForClient(c); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err := startService(clientCfg, []v1.ProxyConfigurer{c}, nil, unsafeFeatures, "")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
}

// NewVisitorCommand 创建新的访问者命令
func NewVisitorCommand(name string, c v1.VisitorConfigurer, clientCfg *v1.ClientCommonConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "visitor",
		Short: fmt.Sprintf("使用单个 %s 访问者运行 frpc", name),
		Run: func(cmd *cobra.Command, args []string) {
			if err := clientCfg.Complete(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)
			validator := validation.NewConfigValidator(unsafeFeatures)
			if _, err := validator.ValidateClientCommonConfig(clientCfg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			c.Complete(clientCfg)
			c.GetBaseConfig().Type = name
			if err := validation.ValidateVisitorConfigurer(c); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err := startService(clientCfg, nil, []v1.VisitorConfigurer{c}, unsafeFeatures, "")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
}
