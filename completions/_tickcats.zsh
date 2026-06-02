#compdef tickcats

_tickcats() {
  local context state line
  typeset -A opt_args

  _arguments -C \
    '1:command:->command' \
    '*::arg:->args'

  case $state in
    command)
      local -a commands
      commands=(
        'init:create board folders'
        'new:create a ticket'
        'list:list tickets grouped by column'
        'move:move a ticket between columns'
        'pick-next:print next ready ticket'
        'ids:migrate ticket IDs'
        'tui:open terminal board'
        'help:show help'
      )
      _describe -t commands 'tickcats commands' commands
      ;;
    args)
      case $line[1] in
        new)
          if (( CURRENT == 2 )); then
            local -a kinds
            kinds=('feat:feature' 'task:task' 'bug:bug')
            _describe -t kinds 'ticket kinds' kinds
          fi
          ;;
        move)
          if (( CURRENT == 2 )); then
            local -a tickets
            tickets=(${(f)"$(tickcats __complete tickets 2>/dev/null)"})
            _describe -t tickets 'tickets' tickets
          elif (( CURRENT == 3 || CURRENT == 4 )); then
            local -a columns
            columns=(${(f)"$(tickcats __complete columns 2>/dev/null)"})
            _describe -t columns 'columns' columns
          fi
          ;;
        ids)
          if (( CURRENT == 2 )); then
            _values 'ids command' 'migrate[migrate missing ticket IDs]'
          fi
          ;;
        pick-next)
          if (( CURRENT == 2 )); then
            _values 'pick-next option' '--path[print only the ticket path]'
          fi
          ;;
      esac
      ;;
  esac
}

_tickcats "$@"
