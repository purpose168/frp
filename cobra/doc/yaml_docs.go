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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.yaml.in/yaml/v3"
)

type cmdOption struct {
	Name         string
	Shorthand    string `yaml:",omitempty"`
	DefaultValue string `yaml:"default_value,omitempty"`
	Usage        string `yaml:",omitempty"`
}

type cmdDoc struct {
	Name             string
	Synopsis         string      `yaml:",omitempty"`
	Description      string      `yaml:",omitempty"`
	Usage            string      `yaml:",omitempty"`
	Options          []cmdOption `yaml:",omitempty"`
	InheritedOptions []cmdOption `yaml:"inherited_options,omitempty"`
	Example          string      `yaml:",omitempty"`
	SeeAlso          []string    `yaml:"see_also,omitempty"`
}

// GenYamlTree 为此命令及所有后代在给定目录中创建 yaml 结构的引用文件。
// 如果您的命令名称中包含 `-`，此函数可能无法正常工作。
// 如果您有一个 `cmd` 和两个子命令 `sub` 和 `sub-third`，而 `sub` 有一个名为 `third` 的子命令，
// 则哪个帮助输出将位于文件 `cmd-sub-third.1` 中是未定义的。
func GenYamlTree(cmd *cobra.Command, dir string) error {
	identity := func(s string) string { return s }
	emptyStr := func(s string) string { return "" }
	return GenYamlTreeCustom(cmd, dir, emptyStr, identity)
}

// GenYamlTreeCustom 创建 yaml 结构的引用文件。
func GenYamlTreeCustom(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenYamlTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.ReplaceAll(cmd.CommandPath(), " ", "_") + ".yaml"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	if err := GenYamlCustom(cmd, f, linkHandler); err != nil {
		return err
	}
	return nil
}

// GenYaml 创建 yaml 输出。
func GenYaml(cmd *cobra.Command, w io.Writer) error {
	return GenYamlCustom(cmd, w, func(s string) string { return s })
}

// GenYamlCustom 创建自定义 yaml 输出。
func GenYamlCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	yamlDoc := cmdDoc{}
	yamlDoc.Name = cmd.CommandPath()

	yamlDoc.Synopsis = forceMultiLine(cmd.Short)
	yamlDoc.Description = forceMultiLine(cmd.Long)

	if cmd.Runnable() {
		yamlDoc.Usage = cmd.UseLine()
	}

	if len(cmd.Example) > 0 {
		yamlDoc.Example = cmd.Example
	}

	flags := cmd.NonInheritedFlags()
	if flags.HasFlags() {
		yamlDoc.Options = genFlagResult(flags)
	}
	flags = cmd.InheritedFlags()
	if flags.HasFlags() {
		yamlDoc.InheritedOptions = genFlagResult(flags)
	}

	if hasSeeAlso(cmd) {
		result := []string{}
		if cmd.HasParent() {
			parent := cmd.Parent()
			result = append(result, parent.CommandPath()+" - "+parent.Short)
		}
		children := cmd.Commands()
		sort.Sort(byName(children))
		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			result = append(result, child.CommandPath()+" - "+child.Short)
		}
		yamlDoc.SeeAlso = result
	}

	final, err := yaml.Marshal(&yamlDoc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if _, err := w.Write(final); err != nil {
		return err
	}
	return nil
}

func genFlagResult(flags *pflag.FlagSet) []cmdOption {
	var result []cmdOption

	flags.VisitAll(func(flag *pflag.Flag) {
		// Todo, when we mark a shorthand is deprecated, but specify an empty message.
		// The flag.ShorthandDeprecated is empty as the shorthand is deprecated.
		// Using len(flag.ShorthandDeprecated) > 0 can't handle this, others are ok.
		if len(flag.ShorthandDeprecated) == 0 && len(flag.Shorthand) > 0 {
			opt := cmdOption{
				flag.Name,
				flag.Shorthand,
				flag.DefValue,
				forceMultiLine(flag.Usage),
			}
			result = append(result, opt)
		} else {
			opt := cmdOption{
				Name:         flag.Name,
				DefaultValue: forceMultiLine(flag.DefValue),
				Usage:        forceMultiLine(flag.Usage),
			}
			result = append(result, opt)
		}
	})

	return result
}
