complete -c tickcats -n "__fish_use_subcommand" -f -a init -d "Create board folders"
complete -c tickcats -n "__fish_use_subcommand" -f -a new -d "Create a ticket"
complete -c tickcats -n "__fish_use_subcommand" -f -a list -d "List tickets grouped by column"
complete -c tickcats -n "__fish_use_subcommand" -f -a move -d "Move a ticket between columns"
complete -c tickcats -n "__fish_use_subcommand" -f -a pick-next -d "Print next ready ticket"
complete -c tickcats -n "__fish_use_subcommand" -f -a ids -d "Migrate ticket IDs"
complete -c tickcats -n "__fish_use_subcommand" -f -a tui -d "Open terminal board"
complete -c tickcats -n "__fish_use_subcommand" -f -a help -d "Show help"

complete -c tickcats -n "__fish_seen_subcommand_from new" -f -a "feat task bug"
complete -c tickcats -n "__fish_seen_subcommand_from move" -f -a "(tickcats __complete tickets 2>/dev/null)"
complete -c tickcats -n "__fish_seen_subcommand_from move" -f -a "(tickcats __complete columns 2>/dev/null)"
complete -c tickcats -n "__fish_seen_subcommand_from ids" -f -a migrate
complete -c tickcats -n "__fish_seen_subcommand_from pick-next" -f -a --path
