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
	"github.com/spf13/pflag"
)

// MarkFlagRequired 指示各种 shell 补全实现
// 在执行补全时优先处理指定的标志，
// 并使命令在没有该标志的情况下调用时报告错误。
func (c *Command) MarkFlagRequired(name string) error {
	return MarkFlagRequired(c.Flags(), name)
}

// MarkPersistentFlagRequired 指示各种 shell 补全实现
// 在执行补全时优先处理指定的持久化标志，
// 并使命令在没有该标志的情况下调用时报告错误。
func (c *Command) MarkPersistentFlagRequired(name string) error {
	return MarkFlagRequired(c.PersistentFlags(), name)
}

// MarkFlagRequired 指示各种 shell 补全实现
// 在执行补全时优先处理指定的标志，
// 并使命令在没有该标志的情况下调用时报告错误。
func MarkFlagRequired(flags *pflag.FlagSet, name string) error {
	return flags.SetAnnotation(name, BashCompOneRequiredFlag, []string{"true"})
}

// MarkFlagFilename 指示各种 shell 补全实现
// 将指定标志的补全限制为指定的文件扩展名。
func (c *Command) MarkFlagFilename(name string, extensions ...string) error {
	return MarkFlagFilename(c.Flags(), name, extensions...)
}

// MarkFlagCustom 为指定标志添加 BashCompCustom 注解（如果存在）。
// bash 补全脚本将为该标志调用 bash 函数 f。
//
// 这仅适用于 bash 补全。
// 建议改用 c.RegisterFlagCompletionFunc(...)，它允许
// 注册一个 Go 函数，该函数可在所有 shell 中工作。
func (c *Command) MarkFlagCustom(name string, f string) error {
	return MarkFlagCustom(c.Flags(), name, f)
}

// MarkPersistentFlagFilename 指示各种 shell 补全实现
// 将指定持久化标志的补全限制为指定的文件扩展名。
func (c *Command) MarkPersistentFlagFilename(name string, extensions ...string) error {
	return MarkFlagFilename(c.PersistentFlags(), name, extensions...)
}

// MarkFlagFilename 指示各种 shell 补全实现
// 将指定标志的补全限制为指定的文件扩展名。
func MarkFlagFilename(flags *pflag.FlagSet, name string, extensions ...string) error {
	return flags.SetAnnotation(name, BashCompFilenameExt, extensions)
}

// MarkFlagCustom 为指定标志添加 BashCompCustom 注解（如果存在）。
// bash 补全脚本将为该标志调用 bash 函数 f。
//
// 这仅适用于 bash 补全。
// 建议改用 c.RegisterFlagCompletionFunc(...)，它允许
// 注册一个 Go 函数，该函数可在所有 shell 中工作。
func MarkFlagCustom(flags *pflag.FlagSet, name string, f string) error {
	return flags.SetAnnotation(name, BashCompCustom, []string{f})
}

// MarkFlagDirname 指示各种 shell 补全实现
// 将指定标志的补全限制为目录名。
func (c *Command) MarkFlagDirname(name string) error {
	return MarkFlagDirname(c.Flags(), name)
}

// MarkPersistentFlagDirname 指示各种 shell 补全实现
// 将指定持久化标志的补全限制为目录名。
func (c *Command) MarkPersistentFlagDirname(name string) error {
	return MarkFlagDirname(c.PersistentFlags(), name)
}

// MarkFlagDirname 指示各种 shell 补全实现
// 将指定标志的补全限制为目录名。
func MarkFlagDirname(flags *pflag.FlagSet, name string) error {
	return flags.SetAnnotation(name, BashCompSubdirsInDir, []string{})
}
