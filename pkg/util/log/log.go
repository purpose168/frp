// Copyright 2016 fatedier, fatedier@gmail.com
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

package log

import (
	"bytes"
	"os"

	"github.com/fatedier/golib/log"
)

var (
	// TraceLevel 是追踪日志级别
	TraceLevel = log.TraceLevel
	// DebugLevel 是调试日志级别
	DebugLevel = log.DebugLevel
	// InfoLevel 是信息日志级别
	InfoLevel = log.InfoLevel
	// WarnLevel 是警告日志级别
	WarnLevel = log.WarnLevel
	// ErrorLevel 是错误日志级别
	ErrorLevel = log.ErrorLevel
)

// Logger 是全局日志记录器
var Logger *log.Logger

// init 初始化日志记录器
func init() {
	Logger = log.New(
		log.WithCaller(true),
		log.AddCallerSkip(1),
		log.WithLevel(log.InfoLevel),
	)
}

// InitLogger 初始化日志记录器
// 参数 logPath 是日志文件路径，"console" 表示输出到控制台
// 参数 levelStr 是日志级别字符串
// 参数 maxDays 是日志文件保留天数
// 参数 disableLogColor 是否禁用日志颜色
func InitLogger(logPath string, levelStr string, maxDays int, disableLogColor bool) {
	options := []log.Option{}
	if logPath == "console" {
		if !disableLogColor {
			options = append(options,
				log.WithOutput(log.NewConsoleWriter(log.ConsoleConfig{
					Colorful: true,
				}, os.Stdout)),
			)
		}
	} else {
		writer := log.NewRotateFileWriter(log.RotateFileConfig{
			FileName: logPath,
			Mode:     log.RotateFileModeDaily,
			MaxDays:  maxDays,
		})
		writer.Init()
		options = append(options, log.WithOutput(writer))
	}

	level, err := log.ParseLevel(levelStr)
	if err != nil {
		level = log.InfoLevel
	}
	options = append(options, log.WithLevel(level))
	Logger = Logger.WithOptions(options...)
}

// Errorf 输出错误级别日志
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Errorf(format string, v ...any) {
	Logger.Errorf(format, v...)
}

// Warnf 输出警告级别日志
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Warnf(format string, v ...any) {
	Logger.Warnf(format, v...)
}

// Infof 输出信息级别日志
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Infof(format string, v ...any) {
	Logger.Infof(format, v...)
}

// Debugf 输出调试级别日志
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Debugf(format string, v ...any) {
	Logger.Debugf(format, v...)
}

// Tracef 输出追踪级别日志
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Tracef(format string, v ...any) {
	Logger.Tracef(format, v...)
}

// Logf 输出指定级别的日志
// 参数 level 是日志级别
// 参数 offset 是调用栈偏移量
// 参数 format 是格式化字符串
// 参数 v 是格式化参数
func Logf(level log.Level, offset int, format string, v ...any) {
	Logger.Logf(level, offset, format, v...)
}

// WriteLogger 是写入日志记录器，实现了 io.Writer 接口
type WriteLogger struct {
	// level 是日志级别
	level log.Level
	// offset 是调用栈偏移量
	offset int
}

// NewWriteLogger 创建新的写入日志记录器
// 参数 level 是日志级别
// 参数 offset 是调用栈偏移量
// 返回写入日志记录器实例
func NewWriteLogger(level log.Level, offset int) *WriteLogger {
	return &WriteLogger{
		level:  level,
		offset: offset,
	}
}

// Write 实现 io.Writer 接口，将数据写入日志
// 参数 p 是要写入的数据
// 返回写入的字节数和可能的错误
func (w *WriteLogger) Write(p []byte) (n int, err error) {
	Logger.Log(w.level, w.offset, string(bytes.TrimRight(p, "\n")))
	return len(p), nil
}
