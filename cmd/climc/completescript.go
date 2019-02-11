package main

const AUTO_COMPLETE_SCRIPT = `# Cloud climc completation

_climc()
{
    local cur second
    local i cmd
    local allopts subopts
    local CLIMC_OPTIONS_FILE="${CLIMC_OPTIONS_FILE:-/etc/bash_completion.d/helpers/climc.options}"

    COMPREPLY=()

    cur="${COMP_WORDS[COMP_CWORD]}"

    i=0
    while read line
    do
        allopts[i]=${line%#*}
        subopts[i]=${line#*#}
        ((i++))
    done <${CLIMC_OPTIONS_FILE}

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
