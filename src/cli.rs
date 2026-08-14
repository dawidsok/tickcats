use std::ffi::{OsStr, OsString};
use std::path::{Path, PathBuf};

use chrono::Utc;

use crate::store::board::{PickResult, State, StoreError, load, pick_next};
use crate::store::operations::{create, init, migrate_ids, move_ticket, resolve_ticket_name};
use crate::ticket::{Kind, Priority};

pub fn run(raw_args: impl IntoIterator<Item = OsString>) -> Result<(), StoreError> {
    let mut raw: Vec<OsString> = raw_args.into_iter().collect();
    let board = extract_board_path(&mut raw)?;
    if raw.is_empty() {
        return Err(StoreError::new("TUI is not implemented yet"));
    }
    let command = raw[0]
        .to_str()
        .ok_or_else(|| StoreError::new("command must be valid UTF-8"))?;
    if command == "move" {
        return run_move_os(&raw[1..], &board);
    }
    let args: Result<Vec<String>, _> = raw[1..]
        .iter()
        .map(|arg| {
            arg.to_str()
                .map(str::to_owned)
                .ok_or_else(|| StoreError::new("command argument must be valid UTF-8"))
        })
        .collect();
    let args = args?;

    match command {
        "init" => run_init(&args, &board),
        "new" => run_new(&args, &board),
        "list" => run_list(&args, &board),
        "pick-next" => run_pick(&args, &board),
        "ids" => run_ids(&args, &board),
        "__complete" => run_complete(&args, &board),
        "help" | "--help" | "-h" => {
            print_help();
            Ok(())
        }
        "tui" => Err(StoreError::new("TUI is not implemented yet")),
        command => Err(StoreError::new(format!("unknown command {command:?}"))),
    }
}

fn extract_board_path(args: &mut Vec<OsString>) -> Result<PathBuf, StoreError> {
    if let Some(index) = args.iter().position(|arg| arg == OsStr::new("--path")) {
        if index + 1 >= args.len() {
            return Err(StoreError::new("--path requires an argument"));
        }
        let board = PathBuf::from(args.remove(index + 1));
        args.remove(index);
        Ok(board)
    } else {
        Ok(PathBuf::from(".tickcats"))
    }
}

fn run_init(args: &[String], board: &Path) -> Result<(), StoreError> {
    let intro = match args {
        [] => true,
        [flag] if flag == "--no-intro" => false,
        _ => return Err(StoreError::new("usage: tickcats init [--no-intro]")),
    };
    init(board, intro)?;
    println!("Initialized {}", board.display());
    Ok(())
}

fn run_new(args: &[String], board: &Path) -> Result<(), StoreError> {
    if args.len() < 2 {
        return Err(StoreError::new(
            "usage: tickcats new feat|task|bug <title> [--ac text]",
        ));
    }
    let kind = Kind::parse_cli(&args[0]).map_err(|error| StoreError::new(error.to_string()))?;
    let ac_index = args.iter().position(|arg| arg == "--ac");
    if args.iter().any(|arg| arg == "--acceptance") {
        return Err(StoreError::new(
            "usage: tickcats new feat|task|bug <title> [--ac text]",
        ));
    }
    let title_end = ac_index.unwrap_or(args.len());
    let title = args[1..title_end].join(" ");
    if title.trim().is_empty() {
        return Err(StoreError::new("ticket title cannot be empty"));
    }
    let acceptance = ac_index.map(|index| args[index + 1..].join(" "));
    let path = create(
        board,
        kind,
        &title,
        Priority::P2,
        true,
        acceptance.as_deref(),
        Utc::now(),
    )?;
    println!("{}", path.display());
    Ok(())
}

fn run_list(args: &[String], board: &Path) -> Result<(), StoreError> {
    if !args.is_empty() {
        return Err(StoreError::new("usage: tickcats list"));
    }
    let board = load(board)?;
    print_warnings(&board);
    for state in crate::store::board::STATES {
        println!("{}", state.display());
        for stored in board.tickets(state) {
            println!(
                "  {}  {}  [{}] {}",
                stored.name.to_string_lossy(),
                display_id(&stored.ticket.id),
                stored.ticket.priority,
                stored.ticket.title
            );
        }
    }
    Ok(())
}

fn run_move_os(args: &[OsString], board: &Path) -> Result<(), StoreError> {
    if args.len() != 3 {
        return Err(StoreError::new(
            "usage: tickcats move <ticket.md> <from-column> <to-column>",
        ));
    }
    let from = args[1]
        .to_str()
        .ok_or_else(|| StoreError::new("column must be valid UTF-8"))?;
    let to = args[2]
        .to_str()
        .ok_or_else(|| StoreError::new("column must be valid UTF-8"))?;
    let from = State::parse_cli(from)?;
    let to = State::parse_cli(to)?;
    let name = resolve_ticket_name(board, from, &args[0])?;
    let path = move_ticket(board, &name, from, to)?;
    println!("{}", path.display());
    Ok(())
}

fn run_pick(args: &[String], board_path: &Path) -> Result<(), StoreError> {
    let print_path = match args {
        [] => false,
        [flag] if flag == "--print-path" => true,
        _ => {
            return Err(StoreError::new("usage: tickcats pick-next [--print-path]"));
        }
    };
    let board = load(board_path)?;
    print_warnings(&board);
    match (pick_next(&board), print_path) {
        (PickResult::None, false) => println!("No ready ticket found"),
        (PickResult::None, true) => return Err(StoreError::new("no ready ticket found")),
        (PickResult::One(ticket), false) => print_ticket(&ticket),
        (PickResult::One(ticket), true) => println!("{}", ticket.path.display()),
        (PickResult::Tie(tickets), false) => {
            println!("Tie candidates:");
            for ticket in tickets {
                print!("  ");
                print_ticket(&ticket);
            }
        }
        (PickResult::Tie(tickets), true) => {
            eprintln!("Tie candidates:");
            for ticket in tickets {
                eprintln!("{}", ticket.path.display());
            }
            return Err(StoreError::new("multiple ready tickets tied for next pick"));
        }
    }
    Ok(())
}

fn run_ids(args: &[String], board: &Path) -> Result<(), StoreError> {
    if args != ["migrate"] {
        return Err(StoreError::new("usage: tickcats ids migrate"));
    }
    let result = migrate_ids(board)?;
    println!("Migrated {} ticket(s)", result.migrated.len());
    for migration in result.migrated {
        println!(
            "  {}  {} -> {}",
            migration.id,
            migration.old_path.display(),
            migration.new_path.display()
        );
    }
    for (column, count) in result.skipped_legacy {
        eprintln!("Warning: skipped legacy column {column} containing {count} ticket(s)");
    }
    Ok(())
}

fn run_complete(args: &[String], board_path: &Path) -> Result<(), StoreError> {
    match args {
        [kind] if kind == "columns" => {
            for column in ["backlog", "ready", "wip", "done"] {
                println!("{column}");
            }
            Ok(())
        }
        [kind] if kind == "tickets" => {
            let board = load(board_path)?;
            for state in crate::store::board::STATES {
                for ticket in board.tickets(state) {
                    println!("{}", ticket.name.to_string_lossy());
                }
            }
            Ok(())
        }
        _ => Err(StoreError::new(
            "usage: tickcats __complete tickets|columns",
        )),
    }
}

fn print_warnings(board: &crate::store::board::Board) {
    for warning in &board.warnings {
        eprintln!("Warning: {}: {}", warning.path.display(), warning.message);
    }
}

fn print_ticket(ticket: &crate::store::board::StoredTicket) {
    println!(
        "{}  {}  [{}] {}",
        ticket.name.to_string_lossy(),
        display_id(&ticket.ticket.id),
        ticket.ticket.priority,
        ticket.ticket.title
    );
}

fn display_id(id: &str) -> &str {
    if id.trim().is_empty() { "—" } else { id }
}

fn print_help() {
    println!("TickCats");
    println!();
    println!("Usage: tickcats [--path <dir>] <command>");
    println!();
    println!("Commands:");
    println!("  init [--no-intro]");
    println!("  new feat|task|bug <title> [--ac text]");
    println!("  list");
    println!("  move <ticket> <from> <to>");
    println!("  pick-next [--print-path]");
    println!("  ids migrate");
    println!("  tui");
}
