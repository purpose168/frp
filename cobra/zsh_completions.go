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

// GenZshCompletionFile 生成带描述的 zsh 补全文件。
func (c *Command) GenZshCompletionFile(filename string) error {
	return c.genZshCompletionFile(filename, true)
}

// GenZshCompletion 生成带描述的 zsh 补全文件并写入传递的 writer。
func (c *Command) GenZshCompletion(w io.Writer) error {
	return c.genZshCompletion(w, true)
}

// GenZshCompletionFileNoDesc 生成不带描述的 zsh 补全文件。
func (c *Command) GenZshCompletionFileNoDesc(filename string) error {
	return c.genZshCompletionFile(filename, false)
}

// GenZshCompletionNoDesc 生成不带描述的 zsh 补全文件并写入传递的 writer。
func (c *Command) GenZshCompletionNoDesc(w io.Writer) error {
	return c.genZshCompletion(w, false)
}

// MarkZshCompPositionalArgumentFile 仅适用于 zsh，其行为与 Bash 补全不一致。
// 因此已被禁用。
// 默认情况下，当没有指定其他补全时，每个参数都会执行文件补全。
// 可以使用 ValidArgsFunction 和 ShellCompDirectiveNoFileComp 按参数禁用文件补全。
// 要实现文件扩展名过滤，可以使用 ValidArgsFunction 和 ShellCompDirectiveFilterFileExt。
//
// 已废弃
func (c *Command) MarkZshCompPositionalArgumentFile(argPosition int, patterns ...string) error {
	return nil
}

// MarkZshCompPositionalArgumentWords 仅适用于 zsh。因此已被禁用。
// 要在所有 shell 中实现相同的行为，可以使用
// ValidArgs（仅适用于第一个参数）或 ValidArgsFunction 适用于任何参数（包括第一个）。
//
// 已废弃
func (c *Command) MarkZshCompPositionalArgumentWords(argPosition int, words ...string) error {
	return nil
}

func (c *Command) genZshCompletionFile(filename string, includeDesc bool) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.genZshCompletion(outFile, includeDesc)
}

func (c *Command) genZshCompletion(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genZshComp(buf, c.Name(), includeDesc)
	_, err := buf.WriteTo(w)
	return err
}

func genZshComp(buf io.StringWriter, name string, includeDesc bool) {
	compCmd := ShellCompRequestCmd
	if !includeDesc {
		compCmd = ShellCompNoDescRequestCmd
	}
	WriteStringAndCheck(buf, fmt.Sprintf(`#compdef %[1]s
compdef _%[1]s %[1]s

# zsh completion for %-36[1]s -*- shell-script -*-

__%[1]s_debug()
{
    local file="$BASH_COMP_DEBUG_FILE"
    if [[ -n ${file} ]]; then
        echo "$*" >> "${file}"
    fi
}

_%[1]s()
{
    local shellCompDirectiveError=%[3]d
    local shellCompDirectiveNoSpace=%[4]d
    local shellCompDirectiveNoFileComp=%[5]d
    local shellCompDirectiveFilterFileExt=%[6]d
    local shellCompDirectiveFilterDirs=%[7]d
    local shellCompDirectiveKeepOrder=%[8]d

    local lastParam lastChar flagPrefix requestComp out directive comp lastComp noSpace keepOrder
    local -a completions

    __%[1]s_debug "\n========= starting completion logic =========="
    __%[1]s_debug "CURRENT: ${CURRENT}, words[*]: ${words[*]}"

    # The user could have moved the cursor backwards on the command-line.
    # We need to trigger completion from the $CURRENT location, so we need
    # to truncate the command-line ($words) up to the $CURRENT location.
    # (We cannot use $CURSOR as its value does not work when a command is an alias.)
    words=("${=words[1,CURRENT]}")
    __%[1]s_debug "Truncated words[*]: ${words[*]},"

    lastParam=${words[-1]}
    lastChar=${lastParam[-1]}
    __%[1]s_debug "lastParam: ${lastParam}, lastChar: ${lastChar}"

    # For zsh, when completing a flag with an = (e.g., %[1]s -n=<TAB>)
    # completions must be prefixed with the flag
    setopt local_options BASH_REMATCH
    if [[ "${lastParam}" =~ '-.*=' ]]; then
        # We are dealing with a flag with an =
        flagPrefix="-P ${BASH_REMATCH}"
    fi

    # Prepare the command to obtain completions
    requestComp="${words[1]} %[2]s ${words[2,-1]}"
    if [ "${lastChar}" = "" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go completion code.
        __%[1]s_debug "Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __%[1]s_debug "About to call: eval ${requestComp}"

    # Use eval to handle any environment variables and such
    out=$(eval ${requestComp} 2>/dev/null)
    __%[1]s_debug "completion output: ${out}"

    # Extract the directive integer following a : from the last line
    local lastLine
    while IFS='\n' read -r line; do
        lastLine=${line}
    done < <(printf "%%s\n" "${out[@]}")
    __%[1]s_debug "last line: ${lastLine}"

    if [ "${lastLine[1]}" = : ]; then
        directive=${lastLine[2,-1]}
        # Remove the directive including the : and the newline
        local suffix
        (( suffix=${#lastLine}+2))
        out=${out[1,-$suffix]}
    else
        # There is no directive specified.  Leave $out as is.
        __%[1]s_debug "No directive found.  Setting do default"
        directive=0
    fi

    __%[1]s_debug "directive: ${directive}"
    __%[1]s_debug "completions: ${out}"
    __%[1]s_debug "flagPrefix: ${flagPrefix}"

    if [ $((directive & shellCompDirectiveError)) -ne 0 ]; then
        __%[1]s_debug "Completion received error. Ignoring completions."
        return
    fi

    local activeHelpMarker="%[9]s"
    local endIndex=${#activeHelpMarker}
    local startIndex=$((${#activeHelpMarker}+1))
    local hasActiveHelp=0
    while IFS='\n' read -r comp; do
        # Check if this is an activeHelp statement (i.e., prefixed with $activeHelpMarker)
        if [ "${comp[1,$endIndex]}" = "$activeHelpMarker" ];then
            __%[1]s_debug "ActiveHelp found: $comp"
            comp="${comp[$startIndex,-1]}"
            if [ -n "$comp" ]; then
                compadd -x "${comp}"
                __%[1]s_debug "ActiveHelp will need delimiter"
                hasActiveHelp=1
            fi

            continue
        fi

        if [ -n "$comp" ]; then
            # If requested, completions are returned with a description.
            # The description is preceded by a TAB character.
            # For zsh's _describe, we need to use a : instead of a TAB.
            # We first need to escape any : as part of the completion itself.
            comp=${comp//:/\\:}

            local tab="$(printf '\t')"
            comp=${comp//$tab/:}

            __%[1]s_debug "Adding completion: ${comp}"
            completions+=${comp}
            lastComp=$comp
        fi
    done < <(printf "%%s\n" "${out[@]}")

    # Add a delimiter after the activeHelp statements, but only if:
    # - there are completions following the activeHelp statements, or
    # - file completion will be performed (so there will be choices after the activeHelp)
    if [ $hasActiveHelp -eq 1 ]; then
        if [ ${#completions} -ne 0 ] || [ $((directive & shellCompDirectiveNoFileComp)) -eq 0 ]; then
            __%[1]s_debug "Adding activeHelp delimiter"
            compadd -x "--"
            hasActiveHelp=0
        fi
    fi

    if [ $((directive & shellCompDirectiveNoSpace)) -ne 0 ]; then
        __%[1]s_debug "Activating nospace."
        noSpace="-S ''"
    fi

    if [ $((directive & shellCompDirectiveKeepOrder)) -ne 0 ]; then
        __%[1]s_debug "Activating keep order."
        keepOrder="-V"
    fi

    if [ $((directive & shellCompDirectiveFilterFileExt)) -ne 0 ]; then
        # File extension filtering
        local filteringCmd
        filteringCmd='_files'
        for filter in ${completions[@]}; do
            if [ ${filter[1]} != '*' ]; then
                # zsh requires a glob pattern to do file filtering
                filter="\*.$filter"
            fi
            filteringCmd+=" -g $filter"
        done
        filteringCmd+=" ${flagPrefix}"

        __%[1]s_debug "File filtering command: $filteringCmd"
        _arguments '*:filename:'"$filteringCmd"
    elif [ $((directive & shellCompDirectiveFilterDirs)) -ne 0 ]; then
        # File completion for directories only
        local subdir
        subdir="${completions[1]}"
        if [ -n "$subdir" ]; then
            __%[1]s_debug "Listing directories in $subdir"
            pushd "${subdir}" >/dev/null 2>&1
        else
            __%[1]s_debug "Listing directories in ."
        fi

        local result
        _arguments '*:dirname:_files -/'" ${flagPrefix}"
        result=$?
        if [ -n "$subdir" ]; then
            popd >/dev/null 2>&1
        fi
        return $result
    else
        __%[1]s_debug "Calling _describe"
        if eval _describe $keepOrder "completions" completions $flagPrefix $noSpace; then
            __%[1]s_debug "_describe found some completions"

            # Return the success of having called _describe
            return 0
        else
            __%[1]s_debug "_describe did not find completions."
            __%[1]s_debug "Checking if we should do file completion."
            if [ $((directive & shellCompDirectiveNoFileComp)) -ne 0 ]; then
                __%[1]s_debug "deactivating file completion"

                # We must return an error code here to let zsh know that there were no
                # completions found by _describe; this is what will trigger other
                # matching algorithms to attempt to find completions.
                # For example zsh can match letters in the middle of words.
                return 1
            else
                # Perform file completion
                __%[1]s_debug "Activating file completion"

                # We must return the result of this command, so it must be the
                # last command, or else we must store its result to return it.
                _arguments '*:filename:_files'" ${flagPrefix}"
            fi
        fi
    fi
}

# don't run the completion function when being source-ed or eval-ed
if [ "$funcstack[1]" = "_%[1]s" ]; then
    _%[1]s
fi
`, name, compCmd,
		ShellCompDirectiveError, ShellCompDirectiveNoSpace, ShellCompDirectiveNoFileComp,
		ShellCompDirectiveFilterFileExt, ShellCompDirectiveFilterDirs, ShellCompDirectiveKeepOrder,
		activeHelpMarker))
}
