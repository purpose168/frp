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

package xlog

import "strings"

// LogWriter 将写入操作转发到 frp 的日志记录器，支持配置日志级别。
// 只要底层的 Logger 是线程安全的，它就可以安全地并发使用。
type LogWriter struct {
	xl      *Logger      // 日志记录器实例
	logFunc func(string) // 日志写入函数，根据不同级别调用不同的日志方法
}

// Write 实现 io.Writer 接口，将字节切片转换为字符串并写入日志
// p: 要写入的字节切片
// 返回写入的字节数和可能的错误
func (w LogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	w.logFunc(msg)
	return len(p), nil
}

// NewTraceWriter 创建一个 Trace 级别的日志写入器
// xl: 日志记录器实例
// 返回 Trace 级别的 LogWriter
func NewTraceWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Tracef("%s", msg) },
	}
}

// NewDebugWriter 创建一个 Debug 级别的日志写入器
// xl: 日志记录器实例
// 返回 Debug 级别的 LogWriter
func NewDebugWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Debugf("%s", msg) },
	}
}

// NewInfoWriter 创建一个 Info 级别的日志写入器
// xl: 日志记录器实例
// 返回 Info 级别的 LogWriter
func NewInfoWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Infof("%s", msg) },
	}
}

// NewWarnWriter 创建一个 Warn 级别的日志写入器
// xl: 日志记录器实例
// 返回 Warn 级别的 LogWriter
func NewWarnWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Warnf("%s", msg) },
	}
}

// NewErrorWriter 创建一个 Error 级别的日志写入器
// xl: 日志记录器实例
// 返回 Error 级别的 LogWriter
func NewErrorWriter(xl *Logger) LogWriter {
	return LogWriter{
		xl:      xl,
		logFunc: func(msg string) { xl.Errorf("%s", msg) },
	}
}
