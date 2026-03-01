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

// 类似于 git、go 工具和其他现代 CLI 工具的命令
// 灵感来源于 go、go-Commander、gh 和 subcommand

package cobra

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
)

var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"appendIfNotPresent":      appendIfNotPresent,
	"rpad":                    rpad,
	"gt":                      Gt,
	"eq":                      Eq,
}

var initializers []func()
var finalizers []func()

const (
	defaultPrefixMatching   = false
	defaultCommandSorting   = true
	defaultCaseInsensitive  = false
	defaultTraverseRunHooks = false
)

// EnablePrefixMatching 允许设置自动前缀匹配。自动前缀匹配在 CLI 工具中
// 自动启用可能是一件危险的事情。
// 将此设置为 true 以启用它。
var EnablePrefixMatching = defaultPrefixMatching

// EnableCommandSorting 控制命令切片的排序，默认情况下是开启的。
// 要禁用排序，请将其设置为 false。
var EnableCommandSorting = defaultCommandSorting

// EnableCaseInsensitive 允许命令名称不区分大小写。（默认区分大小写）
var EnableCaseInsensitive = defaultCaseInsensitive

// EnableTraverseRunHooks 从所有父级执行持久化的预运行和后运行钩子。
// 默认情况下这是禁用的，这意味着只执行找到的第一个运行钩子。
var EnableTraverseRunHooks = defaultTraverseRunHooks

// MousetrapHelpText 在 Windows 上启用信息启动画面
// 如果 CLI 是从 explorer.exe 启动的。
// 要禁用鼠标陷阱，只需将此变量设置为空字符串（""）。
// 仅在 Microsoft Windows 上有效。
var MousetrapHelpText = `这是一个命令行工具。

您需要打开 cmd.exe 并从那里运行它。
`

// MousetrapDisplayDuration 控制 MousetrapHelpText 消息在 Windows 上显示的时间
// 如果 CLI 是从 explorer.exe 启动的。设置为 0 以等待按下回车键。
// 要禁用鼠标陷阱，只需将 MousetrapHelpText 设置为空字符串（""）。
// 仅在 Microsoft Windows 上有效。
var MousetrapDisplayDuration = 5 * time.Second

// AddTemplateFunc 添加一个模板函数，该函数可用于用法和帮助
// 模板生成。
func AddTemplateFunc(name string, tmplFunc interface{}) {
	templateFuncs[name] = tmplFunc
}

// AddTemplateFuncs 添加多个模板函数，这些函数可用于用法和
// 帮助模板生成。
func AddTemplateFuncs(tmplFuncs template.FuncMap) {
	for k, v := range tmplFuncs {
		templateFuncs[k] = v
	}
}

// OnInitialize 设置传递的函数在每个命令的
// Execute 方法被调用时运行。
func OnInitialize(y ...func()) {
	initializers = append(initializers, y...)
}

// OnFinalize 设置传递的函数在每个命令的
// Execute 方法终止时运行。
func OnFinalize(y ...func()) {
	finalizers = append(finalizers, y...)
}

// FIXME Gt 未被 cobra 使用，应在版本 2 中移除。它的存在仅为了与 cobra 的用户兼容。

// Gt 接受两个类型并检查第一个类型是否大于第二个。对于数组、通道、
// 映射和切片类型，Gt 将比较它们的长度。整数直接比较，而字符串首先解析为
// 整数，然后进行比较。
func Gt(a interface{}, b interface{}) bool {
	var left, right int64
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = int64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = av.Int()
	case reflect.String:
		left, _ = strconv.ParseInt(av.String(), 10, 64)
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = int64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = bv.Int()
	case reflect.String:
		right, _ = strconv.ParseInt(bv.String(), 10, 64)
	}

	return left > right
}

// FIXME Eq 未被 cobra 使用，应在版本 2 中移除。它的存在仅为了与 cobra 的用户兼容。

// Eq 接受两个类型并检查它们是否相等。支持的类型有 int 和 string。不支持的类型将引发 panic。
func Eq(a interface{}, b interface{}) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		panic("Eq 在不支持的类型上调用")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() == bv.Int()
	case reflect.String:
		return av.String() == bv.String()
	}
	return false
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// FIXME appendIfNotPresent 未被 cobra 使用，应在版本 2 中移除。它的存在仅为了与 cobra 的用户兼容。

// appendIfNotPresent 将 stringToAppend 追加到 s 的末尾，但仅当它尚未存在于 s 中时。
func appendIfNotPresent(s, stringToAppend string) string {
	if strings.Contains(s, stringToAppend) {
		return s
	}
	return s + " " + stringToAppend
}

// rpad 在字符串右侧添加填充。
func rpad(s string, padding int) string {
	formattedString := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(formattedString, s)
}

func tmpl(text string) *tmplFunc {
	return &tmplFunc{
		tmpl: text,
		fn: func(w io.Writer, data interface{}) error {
			t := template.New("top")
			t.Funcs(templateFuncs)
			template.Must(t.Parse(text))
			return t.Execute(w, data)
		},
	}
}

// ld 比较两个字符串并返回它们之间的莱文斯坦距离。
func ld(s, t string, ignoreCase bool) int {
	if ignoreCase {
		s = strings.ToLower(s)
		t = strings.ToLower(t)
	}
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// CheckErr 打印带有"错误:"前缀的消息，并以错误代码 1 退出。如果消息为 nil，则不执行任何操作。
func CheckErr(msg interface{}) {
	if msg != nil {
		fmt.Fprintln(os.Stderr, "错误:", msg)
		os.Exit(1)
	}
}

// WriteStringAndCheck 将字符串写入缓冲区，并检查错误是否为 nil。
func WriteStringAndCheck(b io.StringWriter, s string) {
	_, err := b.WriteString(s)
	CheckErr(err)
}
