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

	"github.com/spf13/cobra"

	"github.com/purpose168/frp/pkg/config"
	v1 "github.com/purpose168/frp/pkg/config/v1"
	"github.com/purpose168/frp/pkg/nathole"
)

var (
	natHoleSTUNServer string
	natHoleLocalAddr  string
)

// init 初始化NAT打洞命令
func init() {
	rootCmd.AddCommand(natholeCmd)
	natholeCmd.AddCommand(natholeDiscoveryCmd)

	natholeCmd.PersistentFlags().StringVarP(&natHoleSTUNServer, "nat_hole_stun_server", "", "", "NAT打洞的STUN服务器地址")
	natholeCmd.PersistentFlags().StringVarP(&natHoleLocalAddr, "nat_hole_local_addr", "l", "", "连接STUN服务器的本地地址")
}

var natholeCmd = &cobra.Command{
	Use:   "nathole",
	Short: "关于NAT打洞的操作",
}

var natholeDiscoveryCmd = &cobra.Command{
	Use:   "discover",
	Short: "从STUN服务器发现NAT打洞信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 在此处忽略错误，因为我们可以使用命令行参数
		cfg, _, _, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
		if err != nil {
			cfg = &v1.ClientCommonConfig{}
			if err := cfg.Complete(); err != nil {
				fmt.Printf("完成配置失败: %v\n", err)
				os.Exit(1)
			}
		}
		if natHoleSTUNServer != "" {
			cfg.NatHoleSTUNServer = natHoleSTUNServer
		}

		if err := validateForNatHoleDiscovery(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		addrs, localAddr, err := nathole.Discover([]string{cfg.NatHoleSTUNServer}, natHoleLocalAddr)
		if err != nil {
			fmt.Println("发现错误:", err)
			os.Exit(1)
		}
		if len(addrs) < 2 {
			fmt.Printf("发现错误: 无法获取足够的地址，需要 2 个，获得: %v\n", addrs)
			os.Exit(1)
		}

		localIPs, _ := nathole.ListLocalIPsForNatHole(10)

		natFeature, err := nathole.ClassifyNATFeature(addrs, localIPs)
		if err != nil {
			fmt.Println("分类NAT特征错误:", err)
			os.Exit(1)
		}
		fmt.Println("STUN服务器:", cfg.NatHoleSTUNServer)
		fmt.Println("您的NAT类型是:", natFeature.NatType)
		fmt.Println("行为是:", natFeature.Behavior)
		fmt.Println("外部地址是:", addrs)
		fmt.Println("本地地址是:", localAddr.String())
		fmt.Println("公共网络:", natFeature.PublicNetwork)
		return nil
	},
}

// validateForNatHoleDiscovery 验证NAT打洞发现的配置
func validateForNatHoleDiscovery(cfg *v1.ClientCommonConfig) error {
	if cfg.NatHoleSTUNServer == "" {
		return fmt.Errorf("nat_hole_stun_server不能为空")
	}
	return nil
}
