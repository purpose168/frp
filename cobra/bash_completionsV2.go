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
	"io"
	"os"
)

func (c *Command) genBashCompletion(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genBashComp(buf, c.Name(), includeDesc)
	_, err := buf.WriteTo(w)
	return err
}

func genBashComp(buf io.StringWriter, name string, includeDesc bool) {
	compCmd := ShellCompRequestCmd
	if !includeDesc {
		compCmd = ShellCompNoDescRequestCmd
	}

	WriteStringAndCheck(buf, fmt.Sprintf(`# %-36[1]s 的 bash 补全 V2 -*- shell-script -*-

__%[1]s_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Mac 系统使用 bash3，bash-completion 包不包含
# _init_completion 函数。这是该函数的最小版本。
__%[1]s_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

# 此函数调用 %[1]s 程序来获取补全结果和指令。
# 它填充 'out' 和 'directive' 变量。
__%[1]s_get_completion_results() {
    local requestComp lastParam lastChar args

    # 准备请求程序补全的命令。
    # 调用 ${words[0]} 而不是直接调用 %[1]s 可以处理别名
    args=("${words[@]:1}")
    requestComp="${words[0]} %[2]s ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __%[1]s_debug "lastParam ${lastParam}, lastChar ${lastChar}"

    if [[ -z ${cur} && ${lastChar} != = ]]; then
        # 如果最后一个参数已完整（后面有空格）
        # 我们添加一个额外的空参数以便向 go 方法指示这一点。
        __%[1]s_debug "添加额外的空参数"
        requestComp="${requestComp} ''"
    fi

    # 当补全带 = 的标志时（例如 %[1]s -n=<TAB>）
    # bash 聚焦于 = 后的部分，所以我们需要
    # 从 $cur 中移除标志部分
    if [[ ${cur} == -*=* ]]; then
        cur="${cur#*=}"
    fi

    __%[1]s_debug "调用 ${requestComp}"
    # 使用 eval 处理任何环境变量等
    out=$(eval "${requestComp}" 2>/dev/null)

    # 提取输出末尾紧跟在冒号(:)后面的指令整数
    directive=${out##*:}
    # 移除指令
    out=${out%%:*}
    if [[ ${directive} == "${out}" ]]; then
        # 未指定指令
        directive=0
    fi
    __%[1]s_debug "补全指令为: ${directive}"
    __%[1]s_debug "补全内容为: ${out}"
}

__%[1]s_process_completion_results() {
    local shellCompDirectiveError=%[3]d
    local shellCompDirectiveNoSpace=%[4]d
    local shellCompDirectiveNoFileComp=%[5]d
    local shellCompDirectiveFilterFileExt=%[6]d
    local shellCompDirectiveFilterDirs=%[7]d
    local shellCompDirectiveKeepOrder=%[8]d

    if (((directive & shellCompDirectiveError) != 0)); then
        # 错误代码。无补全。
        __%[1]s_debug "从自定义补全 go 代码接收到错误"
        return
    else
        if (((directive & shellCompDirectiveNoSpace) != 0)); then
            if [[ $(type -t compopt) == builtin ]]; then
                __%[1]s_debug "启用无空格"
                compopt -o nospace
            else
                __%[1]s_debug "此版本 bash 不支持无空格指令"
            fi
        fi
        if (((directive & shellCompDirectiveKeepOrder) != 0)); then
            if [[ $(type -t compopt) == builtin ]]; then
                # bash < 4.4 不支持无排序
                if [[ ${BASH_VERSINFO[0]} -lt 4 || ( ${BASH_VERSINFO[0]} -eq 4 && ${BASH_VERSINFO[1]} -lt 4 ) ]]; then
                    __%[1]s_debug "此版本 bash 不支持无排序指令"
                else
                    __%[1]s_debug "启用保持顺序"
                    compopt -o nosort
                fi
            else
                __%[1]s_debug "此版本 bash 不支持无排序指令"
            fi
        fi
        if (((directive & shellCompDirectiveNoFileComp) != 0)); then
            if [[ $(type -t compopt) == builtin ]]; then
                __%[1]s_debug "启用无文件补全"
                compopt +o default
            else
                __%[1]s_debug "此版本 bash 不支持无文件补全指令"
            fi
        fi
    fi

    # 将 activeHelp 与普通补全分开
    local completions=()
    local activeHelp=()
    __%[1]s_extract_activeHelp

    if (((directive & shellCompDirectiveFilterFileExt) != 0)); then
        # 文件扩展名过滤
        local fullFilter="" filter filteringCmd

        # 不要在 $completions 变量周围使用引号，否则换行符
        # 会被保留。
        for filter in ${completions[*]}; do
            fullFilter+="$filter|"
        done

        filteringCmd="_filedir $fullFilter"
        __%[1]s_debug "文件过滤命令: $filteringCmd"
        $filteringCmd
    elif (((directive & shellCompDirectiveFilterDirs) != 0)); then
        # 仅目录的文件补全

        local subdir
        subdir=${completions[0]}
        if [[ -n $subdir ]]; then
            __%[1]s_debug "列出 $subdir 中的目录"
            pushd "$subdir" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
        else
            __%[1]s_debug "列出当前目录中的目录"
            _filedir -d
        fi
    else
        __%[1]s_handle_completion_types
    fi

    __%[1]s_handle_special_char "$cur" :
    __%[1]s_handle_special_char "$cur" =

    # 在完成前打印 activeHelp 语句
    __%[1]s_handle_activeHelp
}

__%[1]s_handle_activeHelp() {
    # 打印 activeHelp 语句
    if ((${#activeHelp[*]} != 0)); then
        if [ -z $COMP_TYPE ]; then
            # Bash v3 不设置 COMP_TYPE 变量。
            printf "\n";
            printf "%%s\n" "${activeHelp[@]}"
            printf "\n"
            __%[1]s_reprint_commandLine
            return
        fi

        # 仅在第二次按 TAB 时打印 ActiveHelp
        if [ $COMP_TYPE -eq 63 ]; then
            printf "\n"
            printf "%%s\n" "${activeHelp[@]}"

            if ((${#COMPREPLY[*]} == 0)); then
                # 当程序没有提供补全选项时，文件补全
                # 可能会启动；如果程序没有禁用它；我们想知道
                # 是否有文件会匹配用户输入的内容，以便我们知道
                # 是否会有补全呈现，所以我们知道如何处理 ActiveHelp。
                # 为了找出答案，我们自己触发文件补全；
                # 调用 _filedir 会在文件匹配时填充 COMPREPLY。
                if (((directive & shellCompDirectiveNoFileComp) == 0)); then
                    __%[1]s_debug "列出文件"
                    _filedir
                fi
            fi

            if ((${#COMPREPLY[*]} != 0)); then
                # 如果有补全选项要显示，打印分隔符。
                # 重新打印命令行将自动由
                # shell 在打印补全选项时完成。
                printf -- "--"
            else
                # 当根本没有补全选项时，我们需要
                # 重新打印命令行，因为 shell 不会
                # 自己完成它。
                __%[1]s_reprint_commandLine
            fi
        elif [ $COMP_TYPE -eq 37 ] || [ $COMP_TYPE -eq 42 ]; then
            # 对于补全类型：menu-complete/menu-complete-backward 和 insert-completions
            # 补全会立即插入命令行，所以我们先
            # 打印 activeHelp 消息并重新打印命令行，因为 shell 不会。
            printf "\n"
            printf "%%s\n" "${activeHelp[@]}"

            __%[1]s_reprint_commandLine
        fi
    fi
}

__%[1]s_reprint_commandLine() {
    # 提示符格式仅在 bash 4.4 及以上版本可用。
    # 我们在使用前测试它是否可用。
    if (x=${PS1@P}) 2> /dev/null; then
        printf "%%s" "${PS1@P}${COMP_LINE[@]}"
    else
        # 无法打印提示符。只是打印
        # 用户输入的文本，足够用了。
        printf "%%s" "${COMP_LINE[@]}"
    fi
}

# 将 activeHelp 行与真实补全分开。
# 填充 $activeHelp 和 $completions 数组。
__%[1]s_extract_activeHelp() {
    local activeHelpMarker="%[9]s"
    local endIndex=${#activeHelpMarker}

    while IFS='' read -r comp; do
        [[ -z $comp ]] && continue

        if [[ ${comp:0:endIndex} == $activeHelpMarker ]]; then
            comp=${comp:endIndex}
            __%[1]s_debug "找到 ActiveHelp: $comp"
            if [[ -n $comp ]]; then
                activeHelp+=("$comp")
            fi
        else
            # 不是 activeHelp 行，而是普通补全
            completions+=("$comp")
        fi
    done <<<"${out}"
}

__%[1]s_handle_completion_types() {
    __%[1]s_debug "__%[1]s_handle_completion_types: COMP_TYPE 为 $COMP_TYPE"

    case $COMP_TYPE in
    37|42)
        # 类型：menu-complete/menu-complete-backward 和 insert-completions
        # 如果用户请求一次插入一个补全，或一次在命令行上插入所有
        # 补全，我们必须移除描述。
        # https://github.com/spf13/cobra/issues/1508

        # 如果没有补全，我们不需要做任何事
        (( ${#completions[@]} == 0 )) && return 0

        local tab=$'\t'

        # 剥离任何描述并转义补全以处理特殊字符
        IFS=$'\n' read -ra completions -d '' < <(printf "%%q\n" "${completions[@]%%%%$tab*}")

        # 只考虑与用户输入匹配的补全
        IFS=$'\n' read -ra COMPREPLY -d '' < <(IFS=$'\n'; compgen -W "${completions[*]}" -- "${cur}")

        # compgen 丢失了转义，所以我们需要再次转义所有补全，因为它们将
        # 被插入到命令行中。
        IFS=$'\n' read -ra COMPREPLY -d '' < <(printf "%%q\n" "${COMPREPLY[@]}")
        ;;

    *)
        # 类型：complete（普通补全）
        __%[1]s_handle_standard_completion_case
        ;;
    esac
}

__%[1]s_handle_standard_completion_case() {
    local tab=$'\t'

    # 如果没有补全，我们不需要做任何事
    (( ${#completions[@]} == 0 )) && return 0

    # 短路优化，如果我们没有描述
    if [[ "${completions[*]}" != *$tab* ]]; then
        # 首先，转义补全以处理特殊字符
        IFS=$'\n' read -ra completions -d '' < <(printf "%%q\n" "${completions[@]}")
        # 只考虑与用户输入匹配的补全
        IFS=$'\n' read -ra COMPREPLY -d '' < <(IFS=$'\n'; compgen -W "${completions[*]}" -- "${cur}")

        # compgen 丢失了转义，所以，如果只有一个补全，我们需要
        # 再次转义它，因为它将被插入到命令行中。如果有多个
        # 补全，我们不希望转义它们，因为它们将在列表中打印，
        # 我们不想在该列表中显示转义字符。
        if (( ${#COMPREPLY[@]} == 1 )); then
            COMPREPLY[0]=$(printf "%%q" "${COMPREPLY[0]}")
        fi
        return 0
    fi

    local longest=0
    local compline
    # 寻找最长补全以便我们可以格式化它们
    for compline in "${completions[@]}"; do
        [[ -z $compline ]] && continue

        # 在检查补全是否与用户输入匹配之前，
        # 我们需要剥离任何描述并转义补全以处理特殊
        # 字符，因为这些转义字符是用户输入的一部分。
        # 不要在子 shell 中调用 "printf"，因为它会慢得多
        # 因为我们在循环中。
        printf -v comp "%%q" "${compline%%%%$tab*}" &>/dev/null || comp=$(printf "%%q" "${compline%%%%$tab*}")

        # 只考虑与用户输入匹配的补全
        [[ $comp == "$cur"* ]] || continue

        # 补全匹配。将其添加到完整补全列表中，包括
        # 其描述。我们不转义补全，因为它可能在有多个时
        # 在列表中打印，我们不想在该列表中显示转义字符。
        COMPREPLY+=("$compline")

        # 在检查长度之前剥离任何描述，同样，不要转义
        # 补全，因为此长度仅在列表中打印补全时使用，
        # 我们不想在该列表中显示转义字符。
        comp=${compline%%%%$tab*}
        if ((${#comp}>longest)); then
            longest=${#comp}
        fi
    done

    # 如果只剩下一个补全，移除描述文本并转义任何特殊字符
    if ((${#COMPREPLY[*]} == 1)); then
        __%[1]s_debug "COMPREPLY[0]: ${COMPREPLY[0]}"
        COMPREPLY[0]=$(printf "%%q" "${COMPREPLY[0]%%%%$tab*}")
        __%[1]s_debug "从单个补全中移除了描述，现在为: ${COMPREPLY[0]}"
    else
        # 格式化描述
        __%[1]s_format_comp_descriptions $longest
    fi
}

__%[1]s_handle_special_char()
{
    local comp="$1"
    local char=$2
    if [[ "$comp" == *${char}* && "$COMP_WORDBREAKS" == *${char}* ]]; then
        local word=${comp%%"${comp##*${char}}"}
        local idx=${#COMPREPLY[*]}
        while ((--idx >= 0)); do
            COMPREPLY[idx]=${COMPREPLY[idx]#"$word"}
        done
    fi
}

__%[1]s_format_comp_descriptions()
{
    local tab=$'\t'
    local comp desc maxdesclength
    local longest=$1

    local i ci
    for ci in ${!COMPREPLY[*]}; do
        comp=${COMPREPLY[ci]}
        # 正确格式化在制表符后面的描述字符串（如果有）
        if [[ "$comp" == *$tab* ]]; then
            __%[1]s_debug "原始补全: $comp"
            desc=${comp#*$tab}
            comp=${comp%%%%$tab*}

            # $COLUMNS 存储当前 shell 宽度。
            # 再移除 4 个字符，因为我们要加 2 个空格和 2 个括号。
            maxdesclength=$(( COLUMNS - longest - 4 ))

            # 确保如果我们要对齐描述，至少可以容纳 8 个字符
            if ((maxdesclength > 8)); then
                # 添加适当数量的空格以对齐描述
                for ((i = ${#comp} ; i < longest ; i++)); do
                    comp+=" "
                done
            else
                # 不填充描述，以便在补全后容纳更多文本
                maxdesclength=$(( COLUMNS - ${#comp} - 4 ))
            fi

            # 如果有足够的空间显示任何描述文本，
            # 截断太长的描述以适应 shell 宽度
            if ((maxdesclength > 0)); then
                if ((${#desc} > maxdesclength)); then
                    desc=${desc:0:$(( maxdesclength - 1 ))}
                    desc+="…"
                fi
                comp+="  ($desc)"
            fi
            COMPREPLY[ci]=$comp
            __%[1]s_debug "最终补全: $comp"
        fi
    done
}

__start_%[1]s()
{
    local cur prev words cword split

    COMPREPLY=()

    # 从 bash-completion 包调用 _init_completion
    # 以正确准备参数
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n =: || return
    else
        __%[1]s_init_completion -n =: || return
    fi

    __%[1]s_debug
    __%[1]s_debug "========= 开始补全逻辑 =========="
    __%[1]s_debug "cur 为 ${cur}, words[*] 为 ${words[*]}, #words[@] 为 ${#words[@]}, cword 为 $cword"

    # 用户可能将光标在命令行上向后移动了。
    # 我们需要从 $cword 位置触发补全，所以我们需要
    # 将命令行（$words）截断到 $cword 位置。
    words=("${words[@]:0:$cword+1}")
    __%[1]s_debug "截断后的 words[*]: ${words[*]},"

    local out directive
    __%[1]s_get_completion_results
    __%[1]s_process_completion_results
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_%[1]s %[1]s
else
    complete -o default -o nospace -F __start_%[1]s %[1]s
fi

# 例如：ts=4 sw=4 et filetype=sh
`, name, compCmd,
		ShellCompDirectiveError, ShellCompDirectiveNoSpace, ShellCompDirectiveNoFileComp,
		ShellCompDirectiveFilterFileExt, ShellCompDirectiveFilterDirs, ShellCompDirectiveKeepOrder,
		activeHelpMarker))
}

// GenBashCompletionFileV2 生成 Bash 补全版本 2。
func (c *Command) GenBashCompletionFileV2(filename string, includeDesc bool) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.GenBashCompletionV2(outFile, includeDesc)
}

// GenBashCompletionV2 生成 Bash 补全文件版本 2
// 并将其写入传递的 writer。
func (c *Command) GenBashCompletionV2(w io.Writer, includeDesc bool) error {
	return c.genBashCompletion(w, includeDesc)
}
