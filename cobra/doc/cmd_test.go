// Copyright 2013-2023 The Cobra Authors
//
// 根据 Apache 许可证第 2.0 版（"许可证"）许可；
// 除非遵守许可证，否则不得使用此文件。
// 您可以在以下地址获取许可证副本：
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 按"原样"分发，不提供任何明示或暗示的保证或条件。
// 请参阅许可证了解具体的语言和权限限制。

package doc

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func emptyRun(*cobra.Command, []string) {}

func init() {
	rootCmd.PersistentFlags().StringP("rootflag", "r", "two", "")
	rootCmd.PersistentFlags().StringP("strtwo", "t", "two", "父标志 strtwo 的帮助信息")

	echoCmd.PersistentFlags().StringP("strone", "s", "one", "标志 strone 的帮助信息")
	echoCmd.PersistentFlags().BoolP("persistentbool", "p", false, "标志 persistentbool 的帮助信息")
	echoCmd.Flags().IntP("intone", "i", 123, "标志 intone 的帮助信息")
	echoCmd.Flags().BoolP("boolone", "b", true, "标志 boolone 的帮助信息")

	timesCmd.PersistentFlags().StringP("strtwo", "t", "2", "子标志 strtwo 的帮助信息")
	timesCmd.Flags().IntP("inttwo", "j", 234, "标志 inttwo 的帮助信息")
	timesCmd.Flags().BoolP("booltwo", "c", false, "标志 booltwo 的帮助信息")

	printCmd.PersistentFlags().StringP("strthree", "s", "three", "标志 strthree 的帮助信息")
	printCmd.Flags().IntP("intthree", "i", 345, "标志 intthree 的帮助信息")
	printCmd.Flags().BoolP("boolthree", "b", true, "标志 boolthree 的帮助信息")

	echoCmd.AddCommand(timesCmd, echoSubCmd, deprecatedCmd)
	rootCmd.AddCommand(printCmd, echoCmd, dummyCmd)
}

var rootCmd = &cobra.Command{
	Use:   "root",
	Short: "根命令简短描述",
	Long:  "根命令详细描述",
	Run:   emptyRun,
}

var echoCmd = &cobra.Command{
	Use:     "echo [要回显的字符串]",
	Aliases: []string{"say"},
	Short:   "向屏幕回显任何内容",
	Long:    "一个完全无用的测试命令",
	Example: "Just run cobra-test echo",
}

var echoSubCmd = &cobra.Command{
	Use:   "echosub [要打印的字符串]",
	Short: "echo 的第二个子命令",
	Long:  "一个绝对完全无用的测试 gendocs 命令",
	Run:   emptyRun,
}

var timesCmd = &cobra.Command{
	Use:        "times [# 次] [要回显的字符串]",
	SuggestFor: []string{"counts"},
	Short:      "多次向屏幕回显任何内容",
	Long:       `一个稍微有点无用的测试命令`,
	Run:        emptyRun,
}

var deprecatedCmd = &cobra.Command{
	Use:        "deprecated [这里什么都不能做]",
	Short:      "一个已废弃的命令",
	Long:       `一个绝对完全无用的测试废弃功能的命令`,
	Deprecated: "请改用 echo",
}

var printCmd = &cobra.Command{
	Use:   "print [要打印的字符串]",
	Short: "向屏幕打印任何内容",
	Long:  `一个绝对完全无用的测试命令`,
}

var dummyCmd = &cobra.Command{
	Use:   "dummy [动作]",
	Short: "执行一个虚拟动作",
}

func checkStringContains(t *testing.T, got, expected string) {
	if !strings.Contains(got, expected) {
		t.Errorf("期望包含: \n %v\n实际获取:\n %v\n", expected, got)
	}
}

func checkStringOmits(t *testing.T, got, expected string) {
	if strings.Contains(got, expected) {
		t.Errorf("期望不包含: \n %v\n实际获取: %v", expected, got)
	}
}
