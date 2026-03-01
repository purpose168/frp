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
	rootCmd.PersistentFlags().StringP("strtwo", "t", "two", "help message for parent flag strtwo")

	echoCmd.PersistentFlags().StringP("strone", "s", "one", "help message for flag strone")
	echoCmd.PersistentFlags().BoolP("persistentbool", "p", false, "help message for flag persistentbool")
	echoCmd.Flags().IntP("intone", "i", 123, "help message for flag intone")
	echoCmd.Flags().BoolP("boolone", "b", true, "help message for flag boolone")

	timesCmd.PersistentFlags().StringP("strtwo", "t", "2", "help message for child flag strtwo")
	timesCmd.Flags().IntP("inttwo", "j", 234, "help message for flag inttwo")
	timesCmd.Flags().BoolP("booltwo", "c", false, "help message for flag booltwo")

	printCmd.PersistentFlags().StringP("strthree", "s", "three", "help message for flag strthree")
	printCmd.Flags().IntP("intthree", "i", 345, "help message for flag intthree")
	printCmd.Flags().BoolP("boolthree", "b", true, "help message for flag boolthree")

	echoCmd.AddCommand(timesCmd, echoSubCmd, deprecatedCmd)
	rootCmd.AddCommand(printCmd, echoCmd, dummyCmd)
}

var rootCmd = &cobra.Command{
	Use:   "root",
	Short: "Root short description",
	Long:  "Root long description",
	Run:   emptyRun,
}

var echoCmd = &cobra.Command{
	Use:     "echo [string to echo]",
	Aliases: []string{"say"},
	Short:   "Echo anything to the screen",
	Long:    "an utterly useless command for testing",
	Example: "Just run cobra-test echo",
}

var echoSubCmd = &cobra.Command{
	Use:   "echosub [string to print]",
	Short: "second sub command for echo",
	Long:  "an absolutely utterly useless command for testing gendocs!.",
	Run:   emptyRun,
}

var timesCmd = &cobra.Command{
	Use:        "times [# times] [string to echo]",
	SuggestFor: []string{"counts"},
	Short:      "Echo anything to the screen more times",
	Long:       `a slightly useless command for testing.`,
	Run:        emptyRun,
}

var deprecatedCmd = &cobra.Command{
	Use:        "deprecated [can't do anything here]",
	Short:      "A command which is deprecated",
	Long:       `an absolutely utterly useless command for testing deprecation!.`,
	Deprecated: "Please use echo instead",
}

var printCmd = &cobra.Command{
	Use:   "print [string to print]",
	Short: "Print anything to the screen",
	Long:  `an absolutely utterly useless command for testing.`,
}

var dummyCmd = &cobra.Command{
	Use:   "dummy [action]",
	Short: "Performs a dummy action",
}

func checkStringContains(t *testing.T, got, expected string) {
	if !strings.Contains(got, expected) {
		t.Errorf("Expected to contain: \n %v\nGot:\n %v\n", expected, got)
	}
}

func checkStringOmits(t *testing.T, got, expected string) {
	if strings.Contains(got, expected) {
		t.Errorf("Expected to not contain: \n %v\nGot: %v", expected, got)
	}
}
