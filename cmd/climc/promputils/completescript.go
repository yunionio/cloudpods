// Copyright 2019 Yunion
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

package promputils

const BASH_COMPLETE_SCRIPT_1 = `arr=()
%s
_climc()
{
    local cur second
    local cmd
    local allopts subopts


    COMPREPLY=()

    cur="${COMP_WORDS[COMP_CWORD]}"

    for ((i = 0; i < ${#arr[@]}; i++))
    do
        line="${arr[$i]}"
        allopts[i]=${line%s`

const BASH_COMPLETE_SCRIPT_2 = `%#*}
        subopts[i]=${line#*#}
    done

    # Add hints for help subcommand.
    allopts[i]="help"
    subopts[i]="${allopts[*]}"

    # Generate hints for subcommand.
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${allopts[*]}" -- ${cur}) )
        return 0
    fi

    # Generate hints for options of subcommands.
    i=0
    second="${COMP_WORDS[1]}"

    for cmd in ${allopts[*]}
    do
        if [[ "$cmd" = "$second" ]]; then
            COMPREPLY=( $(compgen -W "${subopts[i]}" -- ${cur}) )
            break
        fi

        ((i++))
    done
}

complete -F _climc climc

`

const ZSH_COMPLETE_SCRIPT_1 = `arr=()
%s
_climc(){
    local curcontext="$curcontext" state line
    typeset -A opt_args
    count=1
    for ((i = 0; i < ${#arr[@]}; i++))
    do
        line="${arr[$i]}"
        allopts[count]=${line%s`
