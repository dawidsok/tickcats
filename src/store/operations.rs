use std::collections::HashSet;
use std::ffi::{OsStr, OsString};
use std::fs;
use std::path::{Path, PathBuf};

use chrono::{DateTime, Utc};
use rand::Rng;

use crate::ticket::{Kind, ParsedTitle, Priority, new_markdown, parse_markdown};

use super::board::{State, StoreError, load};

const ID_ALPHABET: &[u8] = b"ABCDEFGHJKMNPQRSTUVWXYZ23456789";

pub fn init(root: &Path, intro: bool) -> Result<bool, StoreError> {
    validate_gitignore_basename(root)?;
    let was_new = is_uninitialized(root)?;
    for state in super::board::STATES {
        fs::create_dir_all(root.join(state.folder()))
            .map_err(|error| StoreError::new(format!("create state directory: {error}")))?;
    }
    ensure_gitignore(root)?;
    if was_new && intro {
        write_intro(root)?;
    }
    Ok(was_new)
}

fn is_uninitialized(root: &Path) -> Result<bool, StoreError> {
    let entries = match fs::read_dir(root) {
        Ok(entries) => entries,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(true),
        Err(error) => return Err(StoreError::new(error.to_string())),
    };
    for entry in entries {
        let entry = entry.map_err(|error| StoreError::new(error.to_string()))?;
        if entry.file_name() == "config.json" || entry.file_type().is_ok_and(|kind| kind.is_dir()) {
            return Ok(false);
        }
    }
    Ok(true)
}

fn validate_gitignore_basename(root: &Path) -> Result<&str, StoreError> {
    root.file_name()
        .unwrap_or_else(|| OsStr::new(".tickcats"))
        .to_str()
        .ok_or_else(|| {
            StoreError::new("board directory name must be valid UTF-8 to add it to .gitignore")
        })
}

fn ensure_gitignore(root: &Path) -> Result<(), StoreError> {
    let parent = root.parent().unwrap_or_else(|| Path::new("."));
    let path = parent.join(".gitignore");
    let basename = validate_gitignore_basename(root)?;
    let entry = format!("{basename}/");
    let existing = match fs::read_to_string(&path) {
        Ok(existing) => existing,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => String::new(),
        Err(error) => return Err(StoreError::new(format!("ensure .gitignore entry: {error}"))),
    };
    if existing.lines().any(|line| line.trim() == entry) {
        return Ok(());
    }
    let mut updated = existing;
    if !updated.is_empty() && !updated.ends_with('\n') {
        updated.push('\n');
    }
    updated.push_str(&entry);
    updated.push('\n');
    fs::write(&path, updated)
        .map_err(|error| StoreError::new(format!("ensure .gitignore entry: {error}")))
}

fn write_intro(root: &Path) -> Result<(), StoreError> {
    let id = generate_id(&HashSet::new())?;
    let title = "[to refine] Task: Learn TickCats";
    let content = format!(
        "---\ntitle: {title}\nid: {id}\npriority: P2\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n## Context\n\nWelcome to TickCats. This ordinary ticket teaches the local workflow.\n\n1. Press ? for help.\n2. Press e to edit this Markdown and replace the placeholder Acceptance Criteria.\n3. Move it Backlog -> Ready, run tickcats pick-next, then progress it Ready -> WIP -> Done with p.\n\n## Acceptance Criteria\n\n-\n"
    );
    let path = root
        .join("backlog")
        .join(ticket_filename(&id, "Learn TickCats"));
    fs::write(path, content)
        .map_err(|error| StoreError::new(format!("write intro ticket: {error}")))
}

pub fn create(
    root: &Path,
    kind: Kind,
    title: &str,
    priority: Priority,
    to_refine: bool,
    acceptance: Option<&str>,
    now: DateTime<Utc>,
) -> Result<PathBuf, StoreError> {
    init(root, false)?;
    let board = load(root)?;
    let id = generate_id(&board.existing_ids())?;
    let parsed = ParsedTitle {
        raw: String::new(),
        labels: if to_refine {
            vec!["to refine".to_owned()]
        } else {
            Vec::new()
        },
        kind,
        text: title.trim().to_owned(),
        had_prefix: true,
    };
    let content = new_markdown(&id, &parsed.normalized(), priority, now, acceptance);
    let path = root.join("backlog").join(ticket_filename(&id, title));
    fs::write(&path, content)
        .map_err(|error| StoreError::new(format!("write ticket {:?}: {error}", path)))?;
    Ok(path)
}

fn generate_id(existing: &HashSet<String>) -> Result<String, StoreError> {
    let mut rng = rand::rng();
    for _ in 0..100 {
        let suffix: String = (0..6)
            .map(|_| ID_ALPHABET[rng.random_range(0..ID_ALPHABET.len())] as char)
            .collect();
        let id = format!("TC-{suffix}");
        if !existing.contains(&id) {
            return Ok(id);
        }
    }
    Err(StoreError::new(
        "could not generate unique ticket id after 100 attempts",
    ))
}

pub fn ticket_filename(id: &str, title: &str) -> String {
    let s = slug(title);
    // Cap slug so id prefix (~10) + slug + ".md" stays within the 255-byte filename limit.
    let s = s[..s.len().min(200)].trim_end_matches('-');
    format!("{}-{s}.md", id.to_ascii_lowercase())
}

fn slug(raw: &str) -> String {
    let mut out = String::new();
    let mut separator = false;
    for character in raw.trim().to_ascii_lowercase().chars() {
        if character.is_ascii_alphanumeric() {
            if separator && !out.is_empty() {
                out.push('-');
            }
            out.push(character);
            separator = false;
        } else {
            separator = true;
        }
    }
    if out.is_empty() {
        "ticket".to_owned()
    } else {
        out
    }
}

pub fn resolve_ticket_name(
    root: &Path,
    state: State,
    reference: &OsStr,
) -> Result<OsString, StoreError> {
    if Path::new(reference)
        .extension()
        .is_some_and(|extension| extension == "md")
    {
        validate_filename(reference)?;
        return Ok(reference.to_owned());
    }
    let reference_text = reference
        .to_str()
        .ok_or_else(|| StoreError::new("ticket ID must be valid UTF-8"))?;
    let wanted = reference_text.trim().to_ascii_uppercase();
    if !crate::ticket::valid_id(&wanted) {
        return Err(StoreError::new(format!(
            "ticket reference must be a filename or TC-XXXXXX id, got {reference:?}"
        )));
    }
    let board = load(root)?;
    let mut matches = board
        .tickets(state)
        .iter()
        .filter(|stored| stored.ticket.id == wanted)
        .map(|stored| stored.name.clone());
    let name = matches.next().ok_or_else(|| {
        StoreError::new(format!(
            "ticket id {wanted:?} not found in {}",
            state.display()
        ))
    })?;
    if matches.next().is_some() {
        return Err(StoreError::new(format!(
            "ticket id {wanted:?} is duplicated"
        )));
    }
    Ok(name)
}

fn validate_filename(name: &OsStr) -> Result<(), StoreError> {
    if Path::new(name)
        .file_name()
        .is_none_or(|candidate| candidate != name)
    {
        return Err(StoreError::new(format!(
            "ticket name must be a file name, got {name:?}"
        )));
    }
    if Path::new(name)
        .extension()
        .is_none_or(|extension| extension != "md")
    {
        return Err(StoreError::new(format!(
            "ticket name must end with .md, got {name:?}"
        )));
    }
    Ok(())
}

pub fn move_ticket(
    root: &Path,
    name: &OsStr,
    from: State,
    to: State,
) -> Result<PathBuf, StoreError> {
    validate_filename(name)?;
    if !from.adjacent(to) {
        return Err(StoreError::new(format!(
            "columns {} and {} are not adjacent",
            from.display(),
            to.display()
        )));
    }
    let source = root.join(from.folder()).join(name);
    let target = root.join(to.folder()).join(name);
    validate_source(&source)?;
    if fs::symlink_metadata(&target).is_ok() {
        return Err(StoreError::new(format!(
            "target ticket already exists {:?}",
            target
        )));
    }
    fs::create_dir_all(root.join(to.folder()))
        .map_err(|error| StoreError::new(error.to_string()))?;
    rename_no_replace(&source, &target).map_err(|error| {
        StoreError::new(format!("move ticket {:?} to {:?}: {error}", source, target))
    })?;
    Ok(target)
}

pub fn trash(root: &Path, name: &OsStr, from: State) -> Result<PathBuf, StoreError> {
    validate_filename(name)?;
    let source = root.join(from.folder()).join(name);
    validate_source(&source)?;
    let target_dir = root.join(".trash");
    let target = target_dir.join(name);
    if fs::symlink_metadata(&target).is_ok() {
        return Err(StoreError::new(format!(
            "trash ticket already exists {:?}",
            target
        )));
    }
    fs::create_dir_all(&target_dir).map_err(|error| StoreError::new(error.to_string()))?;
    rename_no_replace(&source, &target)
        .map_err(|error| StoreError::new(format!("trash ticket {:?}: {error}", source)))?;
    Ok(target)
}

fn validate_source(path: &Path) -> Result<Vec<u8>, StoreError> {
    let metadata = fs::symlink_metadata(path)
        .map_err(|error| StoreError::new(format!("read source ticket {:?}: {error}", path)))?;
    if metadata.file_type().is_symlink() || !metadata.is_file() {
        return Err(StoreError::new(format!(
            "source ticket must be a regular file, got {:?}",
            path
        )));
    }
    let data = fs::read(path)
        .map_err(|error| StoreError::new(format!("read source ticket {:?}: {error}", path)))?;
    parse_markdown(&data)
        .map_err(|error| StoreError::new(format!("parse source ticket {:?}: {error}", path)))?;
    Ok(data)
}

pub fn set_important(
    root: &Path,
    name: &OsStr,
    state: State,
    important: bool,
    now: DateTime<Utc>,
) -> Result<(), StoreError> {
    validate_filename(name)?;
    let path = root.join(state.folder()).join(name);
    let data = validate_source(&path)?;
    let normalized = normalize_crlf(&data);
    let text = String::from_utf8(normalized)
        .map_err(|error| StoreError::new(format!("rewrite important: {error}")))?;
    let marker = text
        .find("\n---\n")
        .ok_or_else(|| StoreError::new("missing frontmatter closing fence"))?;
    let mut lines = text[4..marker].lines();
    let mut output = Vec::new();
    for line in lines.by_ref() {
        let key = line.split_once(':').map(|(key, _)| key.trim());
        if key == Some("important") {
            continue;
        }
        if key == Some("updated") {
            output.push(format!(
                "updated: {}",
                now.to_rfc3339_opts(chrono::SecondsFormat::Secs, true)
            ));
            if important {
                output.push("important: true".to_owned());
            }
        } else {
            output.push(line.to_owned());
        }
    }
    let rewritten = format!("---\n{}\n---\n{}", output.join("\n"), &text[marker + 5..]);
    let temporary = path.with_extension("md.tickcats-important");
    fs::write(&temporary, rewritten)
        .map_err(|error| StoreError::new(format!("stage ticket {:?}: {error}", path)))?;
    replace_file(&temporary, &path).map_err(|error| {
        let _ = fs::remove_file(&temporary);
        StoreError::new(format!("write ticket {:?}: {error}", path))
    })
}

fn normalize_crlf(data: &[u8]) -> Vec<u8> {
    let mut normalized = Vec::with_capacity(data.len());
    let mut index = 0;
    while index < data.len() {
        if data[index..].starts_with(b"\r\n") {
            normalized.push(b'\n');
            index += 2;
        } else {
            normalized.push(data[index]);
            index += 1;
        }
    }
    normalized
}

#[derive(Debug)]
pub struct MigrationResult {
    pub migrated: Vec<Migration>,
    pub skipped_legacy: Vec<(String, usize)>,
}

#[derive(Debug)]
pub struct Migration {
    pub id: String,
    pub old_path: PathBuf,
    pub new_path: PathBuf,
}

pub fn migrate_ids(root: &Path) -> Result<MigrationResult, StoreError> {
    recover_migration_backups(root)?;
    let board = load(root)?;
    if board
        .warnings
        .iter()
        .any(|warning| warning.message.contains("duplicate ticket id"))
    {
        return Err(StoreError::new(
            "cannot migrate ids while duplicate ticket ids exist",
        ));
    }
    if let Some(warning) = board.warnings.iter().find(|warning| {
        !warning.message.starts_with("unsupported legacy column")
            && warning.message != "missing ticket id"
    }) {
        return Err(StoreError::new(format!(
            "cannot migrate ids while board warnings exist: {}: {}",
            warning.path.display(),
            warning.message
        )));
    }
    let mut existing = board.existing_ids();
    let mut plans = Vec::new();
    for state in super::board::STATES {
        for stored in board.tickets(state) {
            let data =
                fs::read(&stored.path).map_err(|error| StoreError::new(error.to_string()))?;
            let (id, updated) = if stored.ticket.id.is_empty() {
                let id = generate_id(&existing)?;
                existing.insert(id.clone());
                let updated = add_id(&data, &id)?;
                (id, updated)
            } else {
                let expected_prefix = format!("{}-", stored.ticket.id.to_ascii_lowercase());
                if stored.name.to_string_lossy().starts_with(&expected_prefix) {
                    continue;
                }
                (stored.ticket.id.clone(), data)
            };
            let new_path = unique_path(
                root,
                state,
                &ticket_filename(&id, &stored.ticket.title),
                &plans,
            );
            plans.push((stored.path.clone(), new_path, id, updated));
        }
    }

    let mut migrated = Vec::new();
    for (old_path, new_path, id, updated) in plans {
        let temporary = old_path.with_extension("md.tickcats-migrate");
        fs::write(&temporary, updated)
            .map_err(|error| StoreError::new(format!("stage ticket {:?}: {error}", old_path)))?;
        if let Err(error) = replace_file(&temporary, &old_path) {
            let _ = fs::remove_file(&temporary);
            return Err(StoreError::new(format!(
                "rewrite ticket {:?}: {error}; completed {} ticket(s); rerun safely",
                old_path,
                migrated.len()
            )));
        }
        if old_path != new_path
            && let Err(error) = rename_no_replace(&old_path, &new_path)
        {
            return Err(StoreError::new(format!(
                "rename ticket {:?} to {:?}: {error}; completed {} ticket(s); rerun safely",
                old_path,
                new_path,
                migrated.len()
            )));
        }
        migrated.push(Migration {
            id,
            old_path,
            new_path,
        });
    }

    let skipped_legacy = board
        .warnings
        .iter()
        .filter_map(|warning| {
            warning
                .message
                .strip_prefix("unsupported legacy column contains ")
                .and_then(|rest| rest.split_whitespace().next())
                .and_then(|count| count.parse().ok())
                .map(|count| {
                    (
                        warning
                            .path
                            .file_name()
                            .unwrap_or_default()
                            .to_string_lossy()
                            .into_owned(),
                        count,
                    )
                })
        })
        .collect();
    Ok(MigrationResult {
        migrated,
        skipped_legacy,
    })
}

fn recover_migration_backups(root: &Path) -> Result<(), StoreError> {
    for state in super::board::STATES {
        let directory = root.join(state.folder());
        let entries = match fs::read_dir(&directory) {
            Ok(entries) => entries,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
            Err(error) => return Err(StoreError::new(error.to_string())),
        };
        for entry in entries {
            let entry = entry.map_err(|error| StoreError::new(error.to_string()))?;
            let name = entry.file_name();
            let name = name.to_string_lossy();
            let Some(original_name) = name.strip_suffix(".tickcats-backup") else {
                continue;
            };
            let backup = entry.path();
            let original = directory.join(original_name);
            if original.exists() {
                let ticket = parse_markdown(
                    &fs::read(&original).map_err(|error| StoreError::new(error.to_string()))?,
                )
                .map_err(|error| StoreError::new(error.to_string()))?;
                if ticket.id.is_empty() {
                    return Err(StoreError::new(format!(
                        "cannot recover migration backup {:?}: target has no id",
                        backup
                    )));
                }
                fs::remove_file(&backup).map_err(|error| StoreError::new(error.to_string()))?;
            } else {
                fs::rename(&backup, &original)
                    .map_err(|error| StoreError::new(format!("recover {:?}: {error}", backup)))?;
            }
        }
    }
    Ok(())
}

fn replace_file(source: &Path, target: &Path) -> std::io::Result<()> {
    #[cfg(not(windows))]
    return fs::rename(source, target);

    #[cfg(windows)]
    {
        let backup = target.with_extension("md.tickcats-backup");
        if fs::symlink_metadata(&backup).is_ok() {
            fs::remove_file(&backup)?;
        }
        fs::rename(target, &backup)?;
        match fs::rename(source, target) {
            Ok(()) => fs::remove_file(backup),
            Err(error) => {
                let _ = fs::rename(backup, target);
                Err(error)
            }
        }
    }
}

fn rename_no_replace(source: &Path, target: &Path) -> std::io::Result<()> {
    fs::hard_link(source, target)?;
    if let Err(error) = fs::remove_file(source) {
        let _ = fs::remove_file(target);
        return Err(error);
    }
    Ok(())
}

fn add_id(data: &[u8], id: &str) -> Result<Vec<u8>, StoreError> {
    let text = String::from_utf8(data.to_vec())
        .map_err(|error| StoreError::new(format!("add id: {error}")))?
        .replace("\r\n", "\n");
    let mut lines: Vec<_> = text.lines().map(str::to_owned).collect();
    if lines.first().is_none_or(|line| line != "---") {
        return Err(StoreError::new("missing frontmatter opening fence"));
    }
    let closing = lines
        .iter()
        .enumerate()
        .skip(1)
        .find_map(|(index, line)| (line.trim() == "---").then_some(index))
        .ok_or_else(|| StoreError::new("missing frontmatter closing fence"))?;
    if let Some(index) = lines[1..closing]
        .iter()
        .rposition(|line| {
            line.trim()
                .split_once(':')
                .is_some_and(|(key, _)| key.trim() == "id")
        })
        .map(|index| index + 1)
    {
        let value = lines[index]
            .split_once(':')
            .map_or("", |(_, value)| value)
            .trim();
        if !value.is_empty() {
            return Ok(text.into_bytes());
        }
        lines[index] = format!("id: {id}");
        let mut output = lines.join("\n");
        if text.ends_with('\n') {
            output.push('\n');
        }
        return Ok(output.into_bytes());
    }
    let title = lines[1..closing]
        .iter()
        .position(|line| {
            line.trim()
                .split_once(':')
                .is_some_and(|(key, _)| key.trim() == "title")
        })
        .map(|index| index + 1)
        .ok_or_else(|| StoreError::new("frontmatter title field not found"))?;
    lines.insert(title + 1, format!("id: {id}"));
    let mut output = lines.join("\n");
    if text.ends_with('\n') {
        output.push('\n');
    }
    Ok(output.into_bytes())
}

fn unique_path(
    root: &Path,
    state: State,
    preferred: &str,
    plans: &[(PathBuf, PathBuf, String, Vec<u8>)],
) -> PathBuf {
    let directory = root.join(state.folder());
    let mut candidate = directory.join(preferred);
    let mut suffix = 2;
    while fs::symlink_metadata(&candidate).is_ok() || plans.iter().any(|plan| plan.1 == candidate) {
        let base = preferred.trim_end_matches(".md");
        candidate = directory.join(format!("{base}-{suffix}.md"));
        suffix += 1;
    }
    candidate
}
