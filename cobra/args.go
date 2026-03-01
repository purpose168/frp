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
)

// PositionalArgs 定义位置参数的验证函数类型
type PositionalArgs func(cmd *Command, args []string) error

// legacyArgs 验证具有以下行为：
// - 没有子命令的根命令可以接受任意参数
// - 有子命令的根命令将进行子命令有效性检查
// - 子命令始终可以接受任意参数
func legacyArgs(cmd *Command, args []string) error {
	// 没有子命令，始终接受参数
	if !cmd.HasSubCommands() {
		return nil
	}

	// 有子命令的根命令，执行子命令检查
	if !cmd.HasParent() && len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q%s", args[0], cmd.CommandPath(), cmd.findSuggestions(args[0]))
	}
	return nil
}

// NoArgs 如果包含任何参数则返回错误
func NoArgs(cmd *Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return nil
}

// OnlyValidArgs 如果存在任何不在 Command 的 ValidArgs 字段中的位置参数，则返回错误
func OnlyValidArgs(cmd *Command, args []string) error {
	if len(cmd.ValidArgs) > 0 {
		// 移除可能包含在 ValidArgs 中的任何描述
		// 描述跟在制表符后面
		validArgs := make([]string, 0, len(cmd.ValidArgs))
		for _, v := range cmd.ValidArgs {
			validArgs = append(validArgs, strings.SplitN(v, "\t", 2)[0])
		}
		for _, v := range args {
			if !stringInSlice(v, validArgs) {
				return fmt.Errorf("invalid argument %q for %q%s", v, cmd.CommandPath(), cmd.findSuggestions(args[0]))
			}
		}
	}
	return nil
}

// ArbitraryArgs 从不返回错误
func ArbitraryArgs(cmd *Command, args []string) error {
	return nil
}

// MinimumNArgs 如果参数少于 N 个则返回错误
func MinimumNArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) < n {
			return fmt.Errorf("requires at least %d arg(s), only received %d", n, len(args))
		}
		return nil
	}
}

// MaximumNArgs 如果参数超过 N 个则返回错误
func MaximumNArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) > n {
			return fmt.Errorf("accepts at most %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// ExactArgs 如果参数不等于 n 个则返回错误
func ExactArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

// RangeArgs 如果参数数量不在预期范围内则返回错误
func RangeArgs(min int, max int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) < min || len(args) > max {
			return fmt.Errorf("accepts between %d and %d arg(s), received %d", min, max, len(args))
		}
		return nil
	}
}

// MatchAll 允许将多个 PositionalArgs 组合在一起工作
func MatchAll(pargs ...PositionalArgs) PositionalArgs {
	return func(cmd *Command, args []string) error {
		for _, parg := range pargs {
			if err := parg(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

// ExactValidArgs 如果位置参数不等于 N 个，或者存在任何不在 Command 的 ValidArgs 字段中的位置参数，则返回错误
//
// 已弃用：请改用 MatchAll(ExactArgs(n), OnlyValidArgs)
func ExactValidArgs(n int) PositionalArgs {
	return MatchAll(ExactArgs(n), OnlyValidArgs)
}
