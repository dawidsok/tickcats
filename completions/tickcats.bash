_tickcats_complete() {
    local cur cmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    cmd="${COMP_WORDS[1]}"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "init new list move pick-next ids tui help" -- "${cur}") )
        return 0
    fi

    case "${cmd}" in
        new)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "feat task bug" -- "${cur}") )
            fi
            ;;
        move)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(tickcats __complete tickets 2>/dev/null)" -- "${cur}") )
            elif [[ ${COMP_CWORD} -eq 3 || ${COMP_CWORD} -eq 4 ]]; then
                COMPREPLY=( $(compgen -W "$(tickcats __complete columns 2>/dev/null)" -- "${cur}") )
            fi
            ;;
        ids)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "migrate" -- "${cur}") )
            fi
            ;;
        pick-next)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "--path" -- "${cur}") )
            fi
            ;;
    esac
}

complete -F _tickcats_complete tickcats
