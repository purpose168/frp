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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteNoDesCmdInFishScript(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: NoArgs, Run: emptyRun}
	child := &Command{
		Use:               "child",
		ValidArgsFunction: validArgsFunc,
		Run:               emptyRun,
	}
	rootCmd.AddCommand(child)

	buf := new(bytes.Buffer)
	assertNoErr(t, rootCmd.GenFishCompletion(buf, false))
	output := buf.String()

	check(t, output, ShellCompNoDescRequestCmd)
}

func TestCompleteCmdInFishScript(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: NoArgs, Run: emptyRun}
	child := &Command{
		Use:               "child",
		ValidArgsFunction: validArgsFunc,
		Run:               emptyRun,
	}
	rootCmd.AddCommand(child)

	buf := new(bytes.Buffer)
	assertNoErr(t, rootCmd.GenFishCompletion(buf, true))
	output := buf.String()

	check(t, output, ShellCompRequestCmd)
	checkOmit(t, output, ShellCompNoDescRequestCmd)
}

func TestProgWithDash(t *testing.T) {
	rootCmd := &Command{Use: "root-dash", Args: NoArgs, Run: emptyRun}
	buf := new(bytes.Buffer)
	assertNoErr(t, rootCmd.GenFishCompletion(buf, false))
	output := buf.String()

	// Functions name should have replace the '-'
	check(t, output, "__root_dash_perform_completion")
	checkOmit(t, output, "__root-dash_perform_completion")

	// The command name should not have replaced the '-'
	check(t, output, "-c root-dash")
	checkOmit(t, output, "-c root_dash")
}

func TestProgWithColon(t *testing.T) {
	rootCmd := &Command{Use: "root:colon", Args: NoArgs, Run: emptyRun}
	buf := new(bytes.Buffer)
	assertNoErr(t, rootCmd.GenFishCompletion(buf, false))
	output := buf.String()

	// Functions name should have replace the ':'
	check(t, output, "__root_colon_perform_completion")
	checkOmit(t, output, "__root:colon_perform_completion")

	// The command name should not have replaced the ':'
	check(t, output, "-c root:colon")
	checkOmit(t, output, "-c root_colon")
}

func TestFishCompletionNoActiveHelp(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	buf := new(bytes.Buffer)
	assertNoErr(t, c.GenFishCompletion(buf, true))
	output := buf.String()

	// check that active help is being disabled
	activeHelpVar := activeHelpEnvVar(c.Name())
	check(t, output, fmt.Sprintf("%s=0", activeHelpVar))
}

func TestGenFishCompletionFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "cobra-test")
	if err != nil {
		t.Fatal(err.Error())
	}

	defer os.Remove(tmpFile.Name())

	rootCmd := &Command{Use: "root", Args: NoArgs, Run: emptyRun}
	child := &Command{
		Use:               "child",
		ValidArgsFunction: validArgsFunc,
		Run:               emptyRun,
	}
	rootCmd.AddCommand(child)

	assertNoErr(t, rootCmd.GenFishCompletionFile(tmpFile.Name(), false))
}

func TestFailGenFishCompletionFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cobra-test")
	if err != nil {
		t.Fatal(err.Error())
	}

	defer os.RemoveAll(tmpDir)

	f, _ := os.OpenFile(filepath.Join(tmpDir, "test"), os.O_CREATE, 0400)
	defer f.Close()

	rootCmd := &Command{Use: "root", Args: NoArgs, Run: emptyRun}
	child := &Command{
		Use:               "child",
		ValidArgsFunction: validArgsFunc,
		Run:               emptyRun,
	}
	rootCmd.AddCommand(child)

	got := rootCmd.GenFishCompletionFile(f.Name(), false)
	if !errors.Is(got, os.ErrPermission) {
		t.Errorf("got: %s, want: %s", got.Error(), os.ErrPermission.Error())
	}
}
