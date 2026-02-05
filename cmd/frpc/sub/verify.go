// 版权所有 2021 frp 作者
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

package sub

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
)

// init 初始化 verify 命令
func init() {
	rootCmd.AddCommand(verifyCmd)
}

// verifyCmd 是验证配置文件有效性的命令
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "验证配置是否有效",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgFile == "" {
			fmt.Println("frpc: 未指定配置文件")
			return nil
		}

		cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)
		warning, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs, unsafeFeatures)
		if warning != nil {
			fmt.Printf("警告: %v\n", warning)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("frpc: 配置文件 %s 语法正确\n", cfgFile)
		return nil
	},
}
