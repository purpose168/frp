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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// checkOmit 检查是否包含不应出现的字符串
func checkOmit(t *testing.T, found, unexpected string) {
	if strings.Contains(found, unexpected) {
		t.Errorf("得到: %q\n但不应该包含!\n", unexpected)
	}
}

// check 检查是否包含预期的字符串
func check(t *testing.T, found, expected string) {
	if !strings.Contains(found, expected) {
		t.Errorf("预期包含: \n %q\n得到:\n %q\n", expected, found)
	}
}

// checkNumOccurrences 检查出现的次数
func checkNumOccurrences(t *testing.T, found, expected string, expectedOccurrences int) {
	numOccurrences := strings.Count(found, expected)
	if numOccurrences != expectedOccurrences {
		t.Errorf("预期包含 %d 次: \n %q\n得到 %d:\n %q\n", expectedOccurrences, expected, numOccurrences, found)
	}
}

// checkRegex 检查正则表达式匹配
func checkRegex(t *testing.T, found, pattern string) {
	matched, err := regexp.MatchString(pattern, found)
	if err != nil {
		t.Errorf("执行 MatchString 时出错: \n %s\n", err)
	}
	if !matched {
		t.Errorf("预期匹配: \n %q\n得到:\n %q\n", pattern, found)
	}
}

// runShellCheck 运行 shellcheck 检查脚本
func runShellCheck(s string) error {
	cmd := exec.Command("shellcheck", "-s", "bash", "-", "-e",
		"SC2034", // PREFIX appears unused. Verify it or export it.
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		_, err := stdin.Write([]byte(s))
		CheckErr(err)

		stdin.Close()
	}()

	return cmd.Run()
}

// 世界上最差的自定义函数，只是不断告诉你输入 hello!
const bashCompletionFunc = `__root_custom_func() {
	COMPREPLY=( "hello" )
}
`

// TestBashCompletions 测试 Bash 补全功能
func TestBashCompletions(t *testing.T) {
	rootCmd := &Command{
		Use:                    "root",
		ArgAliases:             []string{"pods", "nodes", "services", "replicationcontrollers", "po", "no", "svc", "rc"},
		ValidArgs:              []string{"pod", "node", "service", "replicationcontroller"},
		BashCompletionFunction: bashCompletionFunc,
		Run:                    emptyRun,
	}
	rootCmd.Flags().IntP("introot", "i", -1, "help message for flag introot")
	assertNoErr(t, rootCmd.MarkFlagRequired("introot"))

	// 文件名
	rootCmd.Flags().String("filename", "", "Enter a filename")
	assertNoErr(t, rootCmd.MarkFlagFilename("filename", "json", "yaml", "yml"))

	// 持久化文件名
	rootCmd.PersistentFlags().String("persistent-filename", "", "Enter a filename")
	assertNoErr(t, rootCmd.MarkPersistentFlagFilename("persistent-filename"))
	assertNoErr(t, rootCmd.MarkPersistentFlagRequired("persistent-filename"))

	// 文件扩展名
	rootCmd.Flags().String("filename-ext", "", "Enter a filename (extension limited)")
	assertNoErr(t, rootCmd.MarkFlagFilename("filename-ext"))
	rootCmd.Flags().String("custom", "", "Enter a filename (extension limited)")
	assertNoErr(t, rootCmd.MarkFlagCustom("custom", "__complete_custom"))

	// 给定目录中的子目录
	rootCmd.Flags().String("theme", "", "theme to use (located in /themes/THEMENAME/)")
	assertNoErr(t, rootCmd.Flags().SetAnnotation("theme", BashCompSubdirsInDir, []string{"themes"}))

	// 检查双字标志
	rootCmd.Flags().StringP("two", "t", "", "this is two word flags")
	rootCmd.Flags().BoolP("two-w-default", "T", false, "this is not two word flags")

	echoCmd := &Command{
		Use:     "echo [string to echo]",
		Aliases: []string{"say"},
		Short:   "Echo anything to the screen",
		Long:    "an utterly useless command for testing.",
		Example: "Just run cobra-test echo",
		Run:     emptyRun,
	}

	echoCmd.Flags().String("filename", "", "Enter a filename")
	assertNoErr(t, echoCmd.MarkFlagFilename("filename", "json", "yaml", "yml"))
	echoCmd.Flags().String("config", "", "config to use (located in /config/PROFILE/)")
	assertNoErr(t, echoCmd.Flags().SetAnnotation("config", BashCompSubdirsInDir, []string{"config"}))

	printCmd := &Command{
		Use:   "print [string to print]",
		Args:  MinimumNArgs(1),
		Short: "Print anything to the screen",
		Long:  "an absolutely utterly useless command for testing.",
		Run:   emptyRun,
	}

	deprecatedCmd := &Command{
		Use:        "deprecated [can't do anything here]",
		Args:       NoArgs,
		Short:      "A command which is deprecated",
		Long:       "an absolutely utterly useless command for testing deprecation!.",
		Deprecated: "Please use echo instead",
		Run:        emptyRun,
	}

	colonCmd := &Command{
		Use: "cmd:colon",
		Run: emptyRun,
	}

	timesCmd := &Command{
		Use:        "times [# times] [string to echo]",
		SuggestFor: []string{"counts"},
		Args:       OnlyValidArgs,
		ValidArgs:  []string{"one", "two", "three", "four"},
		Short:      "Echo anything to the screen more times",
		Long:       "a slightly useless command for testing.",
		Run:        emptyRun,
	}

	echoCmd.AddCommand(timesCmd)
	rootCmd.AddCommand(echoCmd, printCmd, deprecatedCmd, colonCmd)

	buf := new(bytes.Buffer)
	assertNoErr(t, rootCmd.GenBashCompletion(buf))
	output := buf.String()

	check(t, output, "_root")
	check(t, output, "_root_echo")
	check(t, output, "_root_echo_times")
	check(t, output, "_root_print")
	check(t, output, "_root_cmd__colon")

	// 检查必需的标志
	check(t, output, `must_have_one_flag+=("--introot=")`)
	check(t, output, `must_have_one_flag+=("--persistent-filename=")`)
	// 检查具有限定和非限定名称的自定义补全函数
	checkNumOccurrences(t, output, `__custom_func`, 2)      // 1. 检查存在, 2. 调用
	checkNumOccurrences(t, output, `__root_custom_func`, 3) // 1. 检查存在, 2. 调用, 3. 实际定义
	// 检查自定义补全函数体
	check(t, output, `COMPREPLY=( "hello" )`)
	// 检查必需的名词
	check(t, output, `must_have_one_noun+=("pod")`)
	// 检查名词别名
	check(t, output, `noun_aliases+=("pods")`)
	check(t, output, `noun_aliases+=("rc")`)
	checkOmit(t, output, `must_have_one_noun+=("pods")`)
	// 检查文件扩展名标志
	check(t, output, `flags_completion+=("_filedir")`)
	// 检查文件扩展名标志
	check(t, output, `must_have_one_noun+=("three")`)
	// 检查文件扩展名标志
	check(t, output, fmt.Sprintf(`flags_completion+=("__%s_handle_filename_extension_flag json|yaml|yml")`, rootCmd.Name()))
	// 检查子命令中的文件扩展名标志
	checkRegex(t, output, fmt.Sprintf(`_root_echo\(\)\n{[^}]*flags_completion\+=\("__%s_handle_filename_extension_flag json\|yaml\|yml"\)`, rootCmd.Name()))
	// 检查自定义标志
	check(t, output, `flags_completion+=("__complete_custom")`)
	// 检查 subdirs_in_dir 标志
	check(t, output, fmt.Sprintf(`flags_completion+=("__%s_handle_subdirs_in_dir_flag themes")`, rootCmd.Name()))
	// 检查子命令中的 subdirs_in_dir 标志
	checkRegex(t, output, fmt.Sprintf(`_root_echo\(\)\n{[^}]*flags_completion\+=\("__%s_handle_subdirs_in_dir_flag config"\)`, rootCmd.Name()))

	// 检查双字标志
	check(t, output, `two_word_flags+=("--two")`)
	check(t, output, `two_word_flags+=("-t")`)
	checkOmit(t, output, `two_word_flags+=("--two-w-default")`)
	checkOmit(t, output, `two_word_flags+=("-T")`)

	// 检查本地非持久标志
	check(t, output, `local_nonpersistent_flags+=("--two")`)
	check(t, output, `local_nonpersistent_flags+=("--two=")`)
	check(t, output, `local_nonpersistent_flags+=("-t")`)
	check(t, output, `local_nonpersistent_flags+=("--two-w-default")`)
	check(t, output, `local_nonpersistent_flags+=("-T")`)

	checkOmit(t, output, deprecatedCmd.Name())

	// 如果可用，对脚本运行 shellcheck
	if err := exec.Command("which", "shellcheck").Run(); err != nil {
		return
	}
	if err := runShellCheck(output); err != nil {
		t.Fatalf("shellcheck 失败: %v", err)
	}
}

// TestBashCompletionHiddenFlag 测试隐藏标志的 Bash 补全
func TestBashCompletionHiddenFlag(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	const flagName = "hiddenFlag"
	c.Flags().Bool(flagName, false, "")
	assertNoErr(t, c.Flags().MarkHidden(flagName))

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenBashCompletion(buf))
	output := buf.String()

	if strings.Contains(output, flagName) {
		t.Errorf("预期补全不包含 %q 标志: 得到 %v", flagName, output)
	}
}

// TestBashCompletionDeprecatedFlag 测试弃用标志的 Bash 补全
func TestBashCompletionDeprecatedFlag(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	const flagName = "deprecated-flag"
	c.Flags().Bool(flagName, false, "")
	assertNoErr(t, c.Flags().MarkDeprecated(flagName, "use --not-deprecated instead"))

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenBashCompletion(buf))
	output := buf.String()

	if strings.Contains(output, flagName) {
		t.Errorf("预期补全不包含 %q 标志: 得到 %v", flagName, output)
	}
}

// TestBashCompletionTraverseChildren 测试遍历子命令的 Bash 补全
func TestBashCompletionTraverseChildren(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun, TraverseChildren: true}

	c.Flags().StringP("string-flag", "s", "", "string flag")
	c.Flags().BoolP("bool-flag", "b", false, "bool flag")

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenBashCompletion(buf))
	output := buf.String()

	// 检查由于 TraverseChildren 设置为 true，本地非持久标志未设置
	checkOmit(t, output, `local_nonpersistent_flags+=("--string-flag")`)
	checkOmit(t, output, `local_nonpersistent_flags+=("--string-flag=")`)
	checkOmit(t, output, `local_nonpersistent_flags+=("-s")`)
	checkOmit(t, output, `local_nonpersistent_flags+=("--bool-flag")`)
	checkOmit(t, output, `local_nonpersistent_flags+=("-b")`)
}

// TestBashCompletionNoActiveHelp 测试禁用主动帮助的 Bash 补全
func TestBashCompletionNoActiveHelp(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenBashCompletion(buf))
	output := buf.String()

	// 检查主动帮助是否被禁用
	activeHelpVar := activeHelpEnvVar(c.Name())
	check(t, output, fmt.Sprintf("%s=0", activeHelpVar))
}
