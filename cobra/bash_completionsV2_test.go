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

package cobra

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBashCompletionV2WithActiveHelp(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenBashCompletionV2(buf, true))
	output := buf.String()

	// 检查 active help 是否未被禁用
	activeHelpVar := activeHelpEnvVar(c.Name())
	checkOmit(t, output, fmt.Sprintf("%s=0", activeHelpVar))
}
