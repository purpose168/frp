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
)

const (
	activeHelpMarker = "_activeHelp_ "
	// 以下值不应更改：程序将在其用户文档中显式使用它们，用户也将显式使用它们
	activeHelpEnvVarSuffix  = "ACTIVE_HELP"
	activeHelpGlobalEnvVar  = configEnvVarGlobalPrefix + "_" + activeHelpEnvVarSuffix
	activeHelpGlobalDisable = "0"
)

// AppendActiveHelp 将指定的字符串添加到指定的数组中用作 ActiveHelp。
// 这些字符串将由补全脚本处理，并作为 ActiveHelp 显示给用户。
// array 参数应该是包含补全结果的数组。
// 此函数可以在向数组添加补全项之前和/或之后多次调用。
// 每次使用相同数组调用此函数时，触发补全时新的 ActiveHelp 行将显示在之前行的下方。
func AppendActiveHelp(compArray []Completion, activeHelpStr string) []Completion {
	return append(compArray, fmt.Sprintf("%s%s", activeHelpMarker, activeHelpStr))
}

// GetActiveHelpConfig 返回 ActiveHelp 环境变量 <PROGRAM>_ACTIVE_HELP 的值，
// 其中 <PROGRAM> 是根命令名称的大写形式，所有非 ASCII 字母数字字符替换为 `_`。
// 如果全局环境变量 COBRA_ACTIVE_HELP 设置为 "0"，则始终返回 "0"。
func GetActiveHelpConfig(cmd *Command) string {
	activeHelpCfg := os.Getenv(activeHelpGlobalEnvVar)
	if activeHelpCfg != activeHelpGlobalDisable {
		activeHelpCfg = os.Getenv(activeHelpEnvVar(cmd.Root().Name()))
	}
	return activeHelpCfg
}

// activeHelpEnvVar 返回程序特定的 ActiveHelp 环境变量名称。
// 格式为 <PROGRAM>_ACTIVE_HELP，其中 <PROGRAM> 是根命令名称的大写形式，
// 所有非 ASCII 字母数字字符替换为 `_`。
func activeHelpEnvVar(name string) string {
	return configEnvVar(name, activeHelpEnvVarSuffix)
}
