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
	"strings"
	"testing"
)

// getCommand 创建一个带有指定位置参数验证的测试命令
func getCommand(args PositionalArgs, withValid bool) *Command {
	c := &Command{
		Use:  "c",
		Args: args,
		Run:  emptyRun,
	}
	if withValid {
		c.ValidArgs = []string{"one", "two", "three"}
	}
	return c
}

// expectSuccess 验证命令执行成功
func expectSuccess(output string, err error, t *testing.T) {
	if output != "" {
		t.Errorf("意外的输出: %v", output)
	}
	if err != nil {
		t.Fatalf("意外的错误: %v", err)
	}
}

// validOnlyWithInvalidArgs 验证带有无效参数的错误情况
func validOnlyWithInvalidArgs(err error, t *testing.T) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := `invalid argument "a" for "c"`
	if got != expected {
		t.Errorf("预期: %q, 得到: %q", expected, got)
	}
}

// noArgsWithArgs 验证不允许参数时传入参数的错误情况
func noArgsWithArgs(err error, t *testing.T, arg string) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := `unknown command "` + arg + `" for "c"`
	if got != expected {
		t.Errorf("预期: %q, 得到: %q", expected, got)
	}
}

// minimumNArgsWithLessArgs 验证参数数量少于最小要求的错误情况
func minimumNArgsWithLessArgs(err error, t *testing.T) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := "requires at least 2 arg(s), only received 1"
	if got != expected {
		t.Fatalf("预期 %q, 得到 %q", expected, got)
	}
}

// maximumNArgsWithMoreArgs 验证参数数量超过最大限制的错误情况
func maximumNArgsWithMoreArgs(err error, t *testing.T) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := "accepts at most 2 arg(s), received 3"
	if got != expected {
		t.Fatalf("预期 %q, 得到 %q", expected, got)
	}
}

// exactArgsWithInvalidCount 验证参数数量不等于指定数量的错误情况
func exactArgsWithInvalidCount(err error, t *testing.T) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := "accepts 2 arg(s), received 3"
	if got != expected {
		t.Fatalf("预期 %q, 得到 %q", expected, got)
	}
}

// rangeArgsWithInvalidCount 验证参数数量不在指定范围内的错误情况
func rangeArgsWithInvalidCount(err error, t *testing.T) {
	if err == nil {
		t.Fatal("预期一个错误")
	}
	got := err.Error()
	expected := "accepts between 2 and 4 arg(s), received 1"
	if got != expected {
		t.Fatalf("预期 %q, 得到 %q", expected, got)
	}
}

// NoArgs 测试

func TestNoArgs(t *testing.T) {
	c := getCommand(NoArgs, false)
	output, err := executeCommand(c)
	expectSuccess(output, err, t)
}

func TestNoArgs_WithArgs(t *testing.T) {
	c := getCommand(NoArgs, false)
	_, err := executeCommand(c, "one")
	noArgsWithArgs(err, t, "one")
}

func TestNoArgs_WithValid_WithArgs(t *testing.T) {
	c := getCommand(NoArgs, true)
	_, err := executeCommand(c, "one")
	noArgsWithArgs(err, t, "one")
}

func TestNoArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(NoArgs, true)
	_, err := executeCommand(c, "a")
	noArgsWithArgs(err, t, "a")
}

func TestNoArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, NoArgs), true)
	_, err := executeCommand(c, "a")
	validOnlyWithInvalidArgs(err, t)
}

// OnlyValidArgs 测试

func TestOnlyValidArgs(t *testing.T) {
	c := getCommand(OnlyValidArgs, true)
	output, err := executeCommand(c, "one", "two")
	expectSuccess(output, err, t)
}

func TestOnlyValidArgs_WithInvalidArgs(t *testing.T) {
	c := getCommand(OnlyValidArgs, true)
	_, err := executeCommand(c, "a")
	validOnlyWithInvalidArgs(err, t)
}

// ArbitraryArgs 测试

func TestArbitraryArgs(t *testing.T) {
	c := getCommand(ArbitraryArgs, false)
	output, err := executeCommand(c, "a", "b")
	expectSuccess(output, err, t)
}

func TestArbitraryArgs_WithValid(t *testing.T) {
	c := getCommand(ArbitraryArgs, true)
	output, err := executeCommand(c, "one", "two")
	expectSuccess(output, err, t)
}

func TestArbitraryArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(ArbitraryArgs, true)
	output, err := executeCommand(c, "a")
	expectSuccess(output, err, t)
}

func TestArbitraryArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, ArbitraryArgs), true)
	_, err := executeCommand(c, "a")
	validOnlyWithInvalidArgs(err, t)
}

// MinimumNArgs 测试

func TestMinimumNArgs(t *testing.T) {
	c := getCommand(MinimumNArgs(2), false)
	output, err := executeCommand(c, "a", "b", "c")
	expectSuccess(output, err, t)
}

func TestMinimumNArgs_WithValid(t *testing.T) {
	c := getCommand(MinimumNArgs(2), true)
	output, err := executeCommand(c, "one", "three")
	expectSuccess(output, err, t)
}

func TestMinimumNArgs_WithValid__WithInvalidArgs(t *testing.T) {
	c := getCommand(MinimumNArgs(2), true)
	output, err := executeCommand(c, "a", "b")
	expectSuccess(output, err, t)
}

func TestMinimumNArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, MinimumNArgs(2)), true)
	_, err := executeCommand(c, "a", "b")
	validOnlyWithInvalidArgs(err, t)
}

func TestMinimumNArgs_WithLessArgs(t *testing.T) {
	c := getCommand(MinimumNArgs(2), false)
	_, err := executeCommand(c, "a")
	minimumNArgsWithLessArgs(err, t)
}

func TestMinimumNArgs_WithLessArgs_WithValid(t *testing.T) {
	c := getCommand(MinimumNArgs(2), true)
	_, err := executeCommand(c, "one")
	minimumNArgsWithLessArgs(err, t)
}

func TestMinimumNArgs_WithLessArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(MinimumNArgs(2), true)
	_, err := executeCommand(c, "a")
	minimumNArgsWithLessArgs(err, t)
}

func TestMinimumNArgs_WithLessArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, MinimumNArgs(2)), true)
	_, err := executeCommand(c, "a")
	validOnlyWithInvalidArgs(err, t)
}

// MaximumNArgs 测试

func TestMaximumNArgs(t *testing.T) {
	c := getCommand(MaximumNArgs(3), false)
	output, err := executeCommand(c, "a", "b")
	expectSuccess(output, err, t)
}

func TestMaximumNArgs_WithValid(t *testing.T) {
	c := getCommand(MaximumNArgs(2), true)
	output, err := executeCommand(c, "one", "three")
	expectSuccess(output, err, t)
}

func TestMaximumNArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(MaximumNArgs(2), true)
	output, err := executeCommand(c, "a", "b")
	expectSuccess(output, err, t)
}

func TestMaximumNArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, MaximumNArgs(2)), true)
	_, err := executeCommand(c, "a", "b")
	validOnlyWithInvalidArgs(err, t)
}

func TestMaximumNArgs_WithMoreArgs(t *testing.T) {
	c := getCommand(MaximumNArgs(2), false)
	_, err := executeCommand(c, "a", "b", "c")
	maximumNArgsWithMoreArgs(err, t)
}

func TestMaximumNArgs_WithMoreArgs_WithValid(t *testing.T) {
	c := getCommand(MaximumNArgs(2), true)
	_, err := executeCommand(c, "one", "three", "two")
	maximumNArgsWithMoreArgs(err, t)
}

func TestMaximumNArgs_WithMoreArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(MaximumNArgs(2), true)
	_, err := executeCommand(c, "a", "b", "c")
	maximumNArgsWithMoreArgs(err, t)
}

func TestMaximumNArgs_WithMoreArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, MaximumNArgs(2)), true)
	_, err := executeCommand(c, "a", "b", "c")
	validOnlyWithInvalidArgs(err, t)
}

// ExactArgs 测试

func TestExactArgs(t *testing.T) {
	c := getCommand(ExactArgs(3), false)
	output, err := executeCommand(c, "a", "b", "c")
	expectSuccess(output, err, t)
}

func TestExactArgs_WithValid(t *testing.T) {
	c := getCommand(ExactArgs(3), true)
	output, err := executeCommand(c, "three", "one", "two")
	expectSuccess(output, err, t)
}

func TestExactArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(ExactArgs(3), true)
	output, err := executeCommand(c, "three", "a", "two")
	expectSuccess(output, err, t)
}

func TestExactArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, ExactArgs(3)), true)
	_, err := executeCommand(c, "three", "a", "two")
	validOnlyWithInvalidArgs(err, t)
}

func TestExactArgs_WithInvalidCount(t *testing.T) {
	c := getCommand(ExactArgs(2), false)
	_, err := executeCommand(c, "a", "b", "c")
	exactArgsWithInvalidCount(err, t)
}

func TestExactArgs_WithInvalidCount_WithValid(t *testing.T) {
	c := getCommand(ExactArgs(2), true)
	_, err := executeCommand(c, "three", "one", "two")
	exactArgsWithInvalidCount(err, t)
}

func TestExactArgs_WithInvalidCount_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(ExactArgs(2), true)
	_, err := executeCommand(c, "three", "a", "two")
	exactArgsWithInvalidCount(err, t)
}

func TestExactArgs_WithInvalidCount_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, ExactArgs(2)), true)
	_, err := executeCommand(c, "three", "a", "two")
	validOnlyWithInvalidArgs(err, t)
}

// RangeArgs 测试

func TestRangeArgs(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), false)
	output, err := executeCommand(c, "a", "b", "c")
	expectSuccess(output, err, t)
}

func TestRangeArgs_WithValid(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), true)
	output, err := executeCommand(c, "three", "one", "two")
	expectSuccess(output, err, t)
}

func TestRangeArgs_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), true)
	output, err := executeCommand(c, "three", "a", "two")
	expectSuccess(output, err, t)
}

func TestRangeArgs_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, RangeArgs(2, 4)), true)
	_, err := executeCommand(c, "three", "a", "two")
	validOnlyWithInvalidArgs(err, t)
}

func TestRangeArgs_WithInvalidCount(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), false)
	_, err := executeCommand(c, "a")
	rangeArgsWithInvalidCount(err, t)
}

func TestRangeArgs_WithInvalidCount_WithValid(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), true)
	_, err := executeCommand(c, "two")
	rangeArgsWithInvalidCount(err, t)
}

func TestRangeArgs_WithInvalidCount_WithValid_WithInvalidArgs(t *testing.T) {
	c := getCommand(RangeArgs(2, 4), true)
	_, err := executeCommand(c, "a")
	rangeArgsWithInvalidCount(err, t)
}

func TestRangeArgs_WithInvalidCount_WithValidOnly_WithInvalidArgs(t *testing.T) {
	c := getCommand(MatchAll(OnlyValidArgs, RangeArgs(2, 4)), true)
	_, err := executeCommand(c, "a")
	validOnlyWithInvalidArgs(err, t)
}

// Takes(No)Args 测试

func TestRootTakesNoArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "illegal", "args")
	if err == nil {
		t.Fatal("预期一个错误")
	}

	got := err.Error()
	expected := `unknown command "illegal" for "root"`
	if !strings.Contains(got, expected) {
		t.Errorf("预期 %q, 得到 %q", expected, got)
	}
}

func TestRootTakesArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: ArbitraryArgs, Run: emptyRun}
	childCmd := &Command{Use: "child", Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "legal", "args")
	if err != nil {
		t.Errorf("意外的错误: %v", err)
	}
}

func TestChildTakesNoArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Args: NoArgs, Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "child", "illegal", "args")
	if err == nil {
		t.Fatal("预期一个错误")
	}

	got := err.Error()
	expected := `unknown command "illegal" for "root child"`
	if !strings.Contains(got, expected) {
		t.Errorf("预期 %q, 得到 %q", expected, got)
	}
}

func TestChildTakesArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Args: ArbitraryArgs, Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "child", "legal", "args")
	if err != nil {
		t.Fatalf("意外的错误: %v", err)
	}
}

func TestMatchAll(t *testing.T) {
	// 稍微复杂的示例，确保正好有3个参数，且每个参数正好是2字节长
	pargs := MatchAll(
		ExactArgs(3),
		func(cmd *Command, args []string) error {
			for _, arg := range args {
				if len([]byte(arg)) != 2 {
					return fmt.Errorf("预期正好是2字节长")
				}
			}
			return nil
		},
	)

	testCases := map[string]struct {
		args []string
		fail bool
	}{
		"happy path": {
			[]string{"aa", "bb", "cc"},
			false,
		},
		"incorrect number of args": {
			[]string{"aa", "bb", "cc", "dd"},
			true,
		},
		"incorrect number of bytes in one arg": {
			[]string{"aa", "bb", "abc"},
			true,
		},
	}

	rootCmd := &Command{Use: "root", Args: pargs, Run: emptyRun}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := executeCommand(rootCmd, tc.args...)
			if err != nil && !tc.fail {
				t.Errorf("意外的错误: %v\n", err)
			}
			if err == nil && tc.fail {
				t.Errorf("预期错误")
			}
		})
	}
}

// 已弃用

func TestExactValidArgs(t *testing.T) {
	c := getCommand(ExactValidArgs(3), true)
	output, err := executeCommand(c, "three", "one", "two")
	expectSuccess(output, err, t)
}

func TestExactValidArgs_WithInvalidCount(t *testing.T) {
	c := getCommand(ExactValidArgs(2), false)
	_, err := executeCommand(c, "three", "one", "two")
	exactArgsWithInvalidCount(err, t)
}

func TestExactValidArgs_WithInvalidCount_WithInvalidArgs(t *testing.T) {
	c := getCommand(ExactValidArgs(2), true)
	_, err := executeCommand(c, "three", "a", "two")
	exactArgsWithInvalidCount(err, t)
}

func TestExactValidArgs_WithInvalidArgs(t *testing.T) {
	c := getCommand(ExactValidArgs(2), true)
	_, err := executeCommand(c, "three", "a")
	validOnlyWithInvalidArgs(err, t)
}

// 此测试确保我们与 legacyArgs() 函数保持向后兼容性
// 它确保根命令在没有子命令时接受参数
func TestLegacyArgsRootAcceptsArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: nil, Run: emptyRun}

	_, err := executeCommand(rootCmd, "somearg")
	if err != nil {
		t.Fatalf("意外的错误: %v", err)
	}
}

// 此测试确保我们与 legacyArgs() 函数保持向后兼容性
// 它确保子命令接受参数和进一步的子命令
func TestLegacyArgsSubcmdAcceptsArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: nil, Run: emptyRun}
	childCmd := &Command{Use: "child", Args: nil, Run: emptyRun}
	grandchildCmd := &Command{Use: "grandchild", Args: nil, Run: emptyRun}
	rootCmd.AddCommand(childCmd)
	childCmd.AddCommand(grandchildCmd)

	_, err := executeCommand(rootCmd, "child", "somearg")
	if err != nil {
		t.Fatalf("意外的错误: %v", err)
	}
}
