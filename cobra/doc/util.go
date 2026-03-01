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

	"github.com/spf13/cobra"
)

// 测试我们是否有理由在文档中打印"另见"信息
// 基本上这是对父命令或子命令的测试，该子命令既未废弃也不是自动生成的帮助命令。
func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

// yaml 库生成包含长字符串但不包含 \n 的不正确 yaml 的临时解决方案。
func forceMultiLine(s string) string {
	if len(s) > 60 && !strings.Contains(s, "\n") {
		s += "\n"
	}
	return s
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
