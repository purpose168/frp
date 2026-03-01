// Copyright 2013-2023 The Cobra Authors
//
// 根据 Apache 许可证 2.0 版本（"许可证"）授权；
// 除非遵守许可证，否则您不得使用此文件。
// 您可以在以下位置获取许可证副本：
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// 除非适用法律要求或书面同意，否则根据许可证分发的软件
// 是按"原样"基础分发的，不附带任何明示或暗示的担保或条件。
// 请参阅许可证中有关管理权限和
// 限制的特定语言。

package cobra

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	activeHelpMessage  = "This is an activeHelp message"
	activeHelpMessage2 = "This is the rest of the activeHelp message"
)

func TestActiveHelpAlone(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}

	activeHelpFunc := func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := AppendActiveHelp(nil, activeHelpMessage)
		return comps, ShellCompDirectiveDefault
	}

	// 测试 activeHelp 可以添加到根命令
	rootCmd.ValidArgsFunction = activeHelpFunc

	output, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected := strings.Join([]string{
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	rootCmd.ValidArgsFunction = nil

	// 测试 activeHelp 可以添加到子命令
	childCmd := &Command{
		Use:   "thechild",
		Short: "The child command",
		Run:   emptyRun,
	}
	rootCmd.AddCommand(childCmd)

	childCmd.ValidArgsFunction = activeHelpFunc

	output, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected = strings.Join([]string{
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}
}

func TestActiveHelpWithComps(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}

	childCmd := &Command{
		Use:   "thechild",
		Short: "The child command",
		Run:   emptyRun,
	}
	rootCmd.AddCommand(childCmd)

	// 测试 activeHelp 可以在其他补全之后添加
	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := []string{"first", "second"}
		comps = AppendActiveHelp(comps, activeHelpMessage)
		return comps, ShellCompDirectiveDefault
	}

	output, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected := strings.Join([]string{
		"first",
		"second",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	// 测试 activeHelp 可以在其他补全之前添加
	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		var comps []string
		comps = AppendActiveHelp(comps, activeHelpMessage)
		comps = append(comps, []string{"first", "second"}...)
		return comps, ShellCompDirectiveDefault
	}

	output, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected = strings.Join([]string{
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		"first",
		"second",
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	// 测试 activeHelp 可以与其他补全交错添加
	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := []string{"first"}
		comps = AppendActiveHelp(comps, activeHelpMessage)
		comps = append(comps, "second")
		return comps, ShellCompDirectiveDefault
	}

	output, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected = strings.Join([]string{
		"first",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		"second",
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}
}

func TestMultiActiveHelp(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}

	childCmd := &Command{
		Use:   "thechild",
		Short: "The child command",
		Run:   emptyRun,
	}
	rootCmd.AddCommand(childCmd)

	// 测试可以添加多条 activeHelp 消息
	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := AppendActiveHelp(nil, activeHelpMessage)
		comps = AppendActiveHelp(comps, activeHelpMessage2)
		return comps, ShellCompDirectiveNoFileComp
	}

	output, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected := strings.Join([]string{
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage2),
		":4",
		"Completion ended with directive: ShellCompDirectiveNoFileComp", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	// 测试多条 activeHelp 消息可以与补全一起使用
	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := []string{"first"}
		comps = AppendActiveHelp(comps, activeHelpMessage)
		comps = append(comps, "second")
		comps = AppendActiveHelp(comps, activeHelpMessage2)
		return comps, ShellCompDirectiveNoFileComp
	}

	output, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected = strings.Join([]string{
		"first",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		"second",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage2),
		":4",
		"Completion ended with directive: ShellCompDirectiveNoFileComp", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}
}

func TestActiveHelpForFlag(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}
	flagname := "flag"
	rootCmd.Flags().String(flagname, "", "A flag")

	// 测试可以添加多条 activeHelp 消息
	_ = rootCmd.RegisterFlagCompletionFunc(flagname, func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := []string{"first"}
		comps = AppendActiveHelp(comps, activeHelpMessage)
		comps = append(comps, "second")
		comps = AppendActiveHelp(comps, activeHelpMessage2)
		return comps, ShellCompDirectiveNoFileComp
	})

	output, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "--flag", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	expected := strings.Join([]string{
		"first",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage),
		"second",
		fmt.Sprintf("%s%s", activeHelpMarker, activeHelpMessage2),
		":4",
		"Completion ended with directive: ShellCompDirectiveNoFileComp", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}
}

func TestConfigActiveHelp(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}

	childCmd := &Command{
		Use:   "thechild",
		Short: "The child command",
		Run:   emptyRun,
	}
	rootCmd.AddCommand(childCmd)

	activeHelpCfg := "someconfig,anotherconfig"
	// 设置用户将要设置的变量
	os.Setenv(activeHelpEnvVar(rootCmd.Name()), activeHelpCfg)

	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		receivedActiveHelpCfg := GetActiveHelpConfig(cmd)
		if receivedActiveHelpCfg != activeHelpCfg {
			t.Errorf("预期 activeHelpConfig: %q, 但得到: %q", activeHelpCfg, receivedActiveHelpCfg)
		}
		return nil, ShellCompDirectiveDefault
	}

	_, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	// 测试标志的 active help 配置
	activeHelpCfg = "a config for a flag"
	// 设置补全脚本将要设置的变量
	os.Setenv(activeHelpEnvVar(rootCmd.Name()), activeHelpCfg)

	flagname := "flag"
	childCmd.Flags().String(flagname, "", "A flag")

	// 测试可以添加多条 activeHelp 消息
	_ = childCmd.RegisterFlagCompletionFunc(flagname, func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		receivedActiveHelpCfg := GetActiveHelpConfig(cmd)
		if receivedActiveHelpCfg != activeHelpCfg {
			t.Errorf("预期 activeHelpConfig: %q, 但得到: %q", activeHelpCfg, receivedActiveHelpCfg)
		}
		return nil, ShellCompDirectiveDefault
	})

	_, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "--flag", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}
}

func TestDisableActiveHelp(t *testing.T) {
	rootCmd := &Command{
		Use: "root",
		Run: emptyRun,
	}

	childCmd := &Command{
		Use:   "thechild",
		Short: "The child command",
		Run:   emptyRun,
	}
	rootCmd.AddCommand(childCmd)

	// 测试使用补全脚本将要设置的特定程序环境变量来禁用 activeHelp
	// 通过在测试中硬编码禁用值 "0" 来确保它；这是为了向后兼容，因为程序将使用此值
	os.Setenv(activeHelpEnvVar(rootCmd.Name()), "0")

	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		comps := []string{"first"}
		comps = AppendActiveHelp(comps, activeHelpMessage)
		return comps, ShellCompDirectiveDefault
	}

	output, err := executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}
	os.Unsetenv(activeHelpEnvVar(rootCmd.Name()))

	// 确保输出中没有 ActiveHelp
	expected := strings.Join([]string{
		"first",
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	// 现在测试全局禁用 ActiveHelp
	os.Setenv(activeHelpGlobalEnvVar, "0")
	// 设置特定变量，以确保当全局环境变量正确设置时被忽略
	os.Setenv(activeHelpEnvVar(rootCmd.Name()), "1")

	output, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}

	// 确保输出中没有 ActiveHelp
	expected = strings.Join([]string{
		"first",
		":0",
		"Completion ended with directive: ShellCompDirectiveDefault", ""}, "\n")

	if output != expected {
		t.Errorf("预期: %q, 得到: %q", expected, output)
	}

	// 确保如果全局环境变量设置为禁用值以外的其他值，它将被忽略
	os.Setenv(activeHelpGlobalEnvVar, "on")
	// 设置特定变量，以确保它被使用（同时忽略全局环境变量）
	activeHelpCfg := "1"
	os.Setenv(activeHelpEnvVar(rootCmd.Name()), activeHelpCfg)

	childCmd.ValidArgsFunction = func(cmd *Command, args []string, toComplete string) ([]string, ShellCompDirective) {
		receivedActiveHelpCfg := GetActiveHelpConfig(cmd)
		if receivedActiveHelpCfg != activeHelpCfg {
			t.Errorf("预期 activeHelpConfig: %q, 但得到: %q", activeHelpCfg, receivedActiveHelpCfg)
		}
		return nil, ShellCompDirectiveDefault
	}

	_, err = executeCommand(rootCmd, ShellCompNoDescRequestCmd, "thechild", "")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}
}
