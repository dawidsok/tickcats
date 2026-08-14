use std::collections::{HashMap, HashSet};
use std::ffi::OsString;
use std::fmt;
use std::fs;
use std::path::{Path, PathBuf};

use chrono::{DateTime, Utc};

use crate::ticket::{Ticket, parse_markdown, valid_id};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum State {
    Backlog,
    Ready,
    Wip,
    Done,
}

pub const STATES: [State; 4] = [State::Backlog, State::Ready, State::Wip, State::Done];

impl State {
    pub const fn folder(self) -> &'static str {
        match self {
            Self::Backlog => "backlog",
            Self::Ready => "ready",
            Self::Wip => "doing",
            Self::Done => "done",
        }
    }

    pub const fn display(self) -> &'static str {
        match self {
            Self::Backlog => "Backlog",
            Self::Ready => "Ready",
            Self::Wip => "WIP",
            Self::Done => "Done",
        }
    }

    pub fn parse_cli(raw: &str) -> Result<Self, StoreError> {
        match raw.trim().to_ascii_lowercase().replace('_', "-").as_str() {
            "backlog" => Ok(Self::Backlog),
            "ready" => Ok(Self::Ready),
            "wip" | "doing" => Ok(Self::Wip),
            "done" => Ok(Self::Done),
            _ => Err(StoreError::new(format!("invalid column {raw:?}"))),
        }
    }

    pub const fn index(self) -> usize {
        match self {
            Self::Backlog => 0,
            Self::Ready => 1,
            Self::Wip => 2,
            Self::Done => 3,
        }
    }

    pub fn adjacent(self, other: Self) -> bool {
        self.index().abs_diff(other.index()) == 1
    }
}

#[derive(Debug, Clone)]
pub struct StoredTicket {
    pub path: PathBuf,
    pub name: OsString,
    pub state: State,
    pub ticket: Ticket,
}

#[derive(Debug, Clone)]
pub struct Warning {
    pub path: PathBuf,
    pub message: String,
}

#[derive(Debug, Default)]
pub struct Board {
    pub columns: HashMap<State, Vec<StoredTicket>>,
    pub warnings: Vec<Warning>,
}

impl Board {
    pub fn tickets(&self, state: State) -> &[StoredTicket] {
        self.columns.get(&state).map_or(&[], Vec::as_slice)
    }

    pub fn existing_ids(&self) -> HashSet<String> {
        self.columns
            .values()
            .flatten()
            .filter(|stored| valid_id(&stored.ticket.id))
            .map(|stored| stored.ticket.id.clone())
            .collect()
    }
}

#[derive(Debug, Clone)]
pub struct StoreError(String);

impl StoreError {
    pub fn new(message: impl Into<String>) -> Self {
        Self(message.into())
    }
}

impl fmt::Display for StoreError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for StoreError {}

pub fn load(root: &Path) -> Result<Board, StoreError> {
    let mut board = Board::default();
    let mut ids: HashMap<String, PathBuf> = HashMap::new();
    for state in STATES {
        let mut tickets = Vec::new();
        let directory = root.join(state.folder());
        let entries = match fs::read_dir(&directory) {
            Ok(entries) => entries,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                board.columns.insert(state, tickets);
                continue;
            }
            Err(error) => {
                return Err(StoreError::new(format!(
                    "read state directory {:?}: {error}",
                    directory
                )));
            }
        };
        for entry in entries {
            let entry = entry.map_err(|error| StoreError::new(error.to_string()))?;
            let path = entry.path();
            if !entry
                .file_type()
                .map_err(|error| StoreError::new(error.to_string()))?
                .is_file()
                || path.extension().is_none_or(|extension| extension != "md")
            {
                continue;
            }
            let data = match fs::read(&path) {
                Ok(data) => data,
                Err(error) => {
                    board.warnings.push(Warning {
                        path,
                        message: error.to_string(),
                    });
                    continue;
                }
            };
            let ticket = match parse_markdown(&data) {
                Ok(ticket) => ticket,
                Err(error) => {
                    board.warnings.push(Warning {
                        path,
                        message: error.to_string(),
                    });
                    continue;
                }
            };
            let name = entry.file_name();
            if ticket.id.is_empty() {
                board.warnings.push(Warning {
                    path: path.clone(),
                    message: "missing ticket id".to_owned(),
                });
            } else {
                if !valid_id(&ticket.id) {
                    board.warnings.push(Warning {
                        path: path.clone(),
                        message: format!("invalid ticket id {:?}: expected TC-XXXXXX", ticket.id),
                    });
                } else if let Some(first) = ids.get(&ticket.id) {
                    board.warnings.push(Warning {
                        path: path.clone(),
                        message: format!(
                            "duplicate ticket id {:?} also used by {}",
                            ticket.id,
                            first.display()
                        ),
                    });
                } else {
                    ids.insert(ticket.id.clone(), path.clone());
                }
            }
            tickets.push(StoredTicket {
                path,
                name,
                state,
                ticket,
            });
        }
        board.columns.insert(state, tickets);
    }
    legacy_warnings(root, &mut board.warnings)?;
    board
        .warnings
        .sort_by(|left, right| left.path.cmp(&right.path));
    let matrix = super::config::Config::load(root)
        .map_err(|error| StoreError::new(error.to_string()))?
        .matrix_enabled();
    let now = Utc::now();
    for tickets in board.columns.values_mut() {
        tickets.sort_by(|left, right| fixed_order_compare(left, right, matrix, now));
    }
    Ok(board)
}

fn legacy_warnings(root: &Path, warnings: &mut Vec<Warning>) -> Result<(), StoreError> {
    let entries = match fs::read_dir(root) {
        Ok(entries) => entries,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(error) => return Err(StoreError::new(error.to_string())),
    };
    let fixed: HashSet<_> = STATES.into_iter().map(State::folder).collect();
    for entry in entries {
        let entry = entry.map_err(|error| StoreError::new(error.to_string()))?;
        let name = entry.file_name().to_string_lossy().into_owned();
        if !entry
            .file_type()
            .map_err(|error| StoreError::new(error.to_string()))?
            .is_dir()
            || name.starts_with('.')
            || fixed.contains(name.as_str())
        {
            continue;
        }
        let count = fs::read_dir(entry.path())
            .map_err(|error| StoreError::new(error.to_string()))?
            .filter_map(Result::ok)
            .filter(|file| {
                file.file_type().is_ok_and(|kind| kind.is_file())
                    && file
                        .path()
                        .extension()
                        .is_some_and(|extension| extension == "md")
            })
            .count();
        if count > 0 {
            warnings.push(Warning {
                path: entry.path(),
                message: format!(
                    "unsupported legacy column contains {count} ticket{}",
                    if count == 1 { "" } else { "s" }
                ),
            });
        }
    }
    Ok(())
}

pub fn pick_next(board: &Board) -> PickResult {
    let mut candidates: Vec<_> = board
        .tickets(State::Ready)
        .iter()
        .filter(|stored| {
            !stored.ticket.title.is_empty()
                && stored.ticket.has_acceptance_criteria
                && !stored.ticket.parsed_title.blocked()
                && !stored.ticket.parsed_title.to_refine()
        })
        .cloned()
        .collect();
    candidates.sort_by(|left, right| {
        left.ticket
            .priority
            .cmp(&right.ticket.priority)
            .then_with(|| left.ticket.created.cmp(&right.ticket.created))
            .then_with(|| left.name.cmp(&right.name))
    });
    let Some(best) = candidates.first().cloned() else {
        return PickResult::None;
    };
    let tied: Vec<_> = candidates
        .into_iter()
        .take_while(|candidate| {
            candidate.ticket.priority == best.ticket.priority
                && candidate.ticket.created == best.ticket.created
        })
        .collect();
    if tied.len() > 1 {
        PickResult::Tie(tied)
    } else {
        PickResult::One(Box::new(best))
    }
}

#[derive(Debug, Clone)]
pub enum PickResult {
    None,
    One(Box<StoredTicket>),
    Tie(Vec<StoredTicket>),
}

pub fn fixed_order_less(
    left: &StoredTicket,
    right: &StoredTicket,
    matrix: bool,
    now: DateTime<Utc>,
) -> bool {
    fixed_order_compare(left, right, matrix, now).is_lt()
}

fn fixed_order_compare(
    left: &StoredTicket,
    right: &StoredTicket,
    matrix: bool,
    now: DateTime<Utc>,
) -> std::cmp::Ordering {
    let common = || {
        left.ticket
            .priority
            .cmp(&right.ticket.priority)
            .then_with(|| left.ticket.created.cmp(&right.ticket.created))
            .then_with(|| left.name.cmp(&right.name))
    };
    if !matrix {
        return common();
    }
    let bucket = |stored: &StoredTicket| {
        let urgent = stored.ticket.deadline.is_some_and(|deadline| {
            deadline.signed_duration_since(now.date_naive()).num_days() <= 7
        });
        match (urgent, stored.ticket.important) {
            (true, true) => 0,
            (false, true) => 1,
            (true, false) => 2,
            (false, false) => 3,
        }
    };
    bucket(left).cmp(&bucket(right)).then_with(common)
}
