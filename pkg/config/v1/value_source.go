// Copyright 2025 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ValueSource 提供了一种从各种来源（如文件、环境变量或外部服务）动态解析配置值的方法
type ValueSource struct {
	Type string      `json:"type"`
	File *FileSource `json:"file,omitempty"`
	Exec *ExecSource `json:"exec,omitempty"`
}

// FileSource 指定如何从文件加载值
type FileSource struct {
	Path string `json:"path"`
}

// ExecSource 指定如何从作为子进程启动的其他程序获取值
type ExecSource struct {
	Command string       `json:"command"`
	Args    []string     `json:"args,omitempty"`
	Env     []ExecEnvVar `json:"env,omitempty"`
}

// ExecEnvVar 执行环境变量结构体
type ExecEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Validate 验证 ValueSource 配置
func (v *ValueSource) Validate() error {
	if v == nil {
		return errors.New("valueSource 不能为 nil")
	}

	switch v.Type {
	case "file":
		if v.File == nil {
			return errors.New("当类型为 'file' 时需要 file 配置")
		}
		return v.File.Validate()
	case "exec":
		if v.Exec == nil {
			return errors.New("当类型为 'exec' 时需要 exec 配置")
		}
		return v.Exec.Validate()
	default:
		return fmt.Errorf("不支持的值源类型: %s (仅支持 'file' 和 'exec')", v.Type)
	}
}

// Resolve 从配置的源解析值
func (v *ValueSource) Resolve(ctx context.Context) (string, error) {
	if err := v.Validate(); err != nil {
		return "", err
	}

	switch v.Type {
	case "file":
		return v.File.Resolve(ctx)
	case "exec":
		return v.Exec.Resolve(ctx)
	default:
		return "", fmt.Errorf("不支持的值源类型: %s", v.Type)
	}
}

// Validate 验证 FileSource 配置
func (f *FileSource) Validate() error {
	if f == nil {
		return errors.New("fileSource 不能为 nil")
	}

	if f.Path == "" {
		return errors.New("文件路径不能为空")
	}
	return nil
}

// Resolve 读取并返回指定文件的内容
func (f *FileSource) Resolve(_ context.Context) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(f.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %v", f.Path, err)
	}

	// 去除空白字符，这对于基于文件的令牌很重要
	return strings.TrimSpace(string(content)), nil
}

// Validate 验证 ExecSource 配置
func (e *ExecSource) Validate() error {
	if e == nil {
		return errors.New("execSource 不能为 nil")
	}

	if e.Command == "" {
		return errors.New("执行命令不能为空")
	}

	for _, env := range e.Env {
		if env.Name == "" {
			return errors.New("执行环境变量名称不能为空")
		}
		if strings.Contains(env.Name, "=") {
			return errors.New("执行环境变量名称不能包含 '='")
		}
	}
	return nil
}

// Resolve 读取并返回从启动的子进程的标准输出捕获的内容
func (e *ExecSource) Resolve(ctx context.Context) (string, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, e.Command, e.Args...)
	if len(e.Env) != 0 {
		cmd.Env = os.Environ()
		for _, env := range e.Env {
			cmd.Env = append(cmd.Env, env.Name+"="+env.Value)
		}
	}

	content, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("执行命令 %v 失败: %v", e.Command, err)
	}

	// 去除空白字符，这对于基于执行的令牌很重要
	return strings.TrimSpace(string(content)), nil
}
