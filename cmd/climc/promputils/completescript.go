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

const ZSH_COMPLETE_SCRIPT_2 = `%"#"*}
        subopts[count]=${line#*"#"}
        ((count++))
    done
    _arguments \
            '1: :->commands'\
            '*: :->options'

    case $state in 
        commands)
        _arguments '1:Comm:(${allopts[*]})'
        ;;
        *)
            item=1
            for key in ${allopts[*]}
            do
                case $words[2] in
                    $key)
                    compadd "$@"` + "`echo $subopts[item]`" + `
                    ;;
                    *)
                    ;;
                esac
                ((item++))
            done
    esac    
}
_climc "$@"
`
