use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::atomic::{AtomicU64, Ordering};

use chrono::{TimeZone, Utc};
use tickcats::store::board::{PickResult, State, fixed_order_less, load, pick_next};
use tickcats::store::operations::{create, init, migrate_ids, move_ticket, set_important, trash};
use tickcats::ticket::{Kind, Priority, parse_markdown};

static NEXT_TEMP: AtomicU64 = AtomicU64::new(0);

struct TempDir(PathBuf);

impl TempDir {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "tickcats-rust-store-{}-{}",
            std::process::id(),
            NEXT_TEMP.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&path).unwrap();
        Self(path)
    }

    fn board(&self) -> PathBuf {
        self.0.join(".tickcats")
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

fn copy_dir(source: &Path, target: &Path) {
    fs::create_dir_all(target).unwrap();
    for entry in fs::read_dir(source).unwrap() {
        let entry = entry.unwrap();
        let destination = target.join(entry.file_name());
        if entry.file_type().unwrap().is_dir() {
            copy_dir(&entry.path(), &destination);
        } else {
            fs::copy(entry.path(), destination).unwrap();
        }
    }
}

fn fixture_board(name: &str) -> TempDir {
    let temp = TempDir::new();
    copy_dir(
        &Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("tests/fixtures/boards")
            .join(name),
        &temp.board(),
    );
    temp
}

fn run(args: &[&str]) -> Output {
    Command::new(env!("CARGO_BIN_EXE_tickcats"))
        .args(args)
        .output()
        .expect("run Rust tickcats")
}

#[test]
fn init_creates_fixed_board_intro_once_and_supports_opt_out() {
    let temp = TempDir::new();
    assert!(init(&temp.board(), true).unwrap());
    for folder in ["backlog", "ready", "doing", "done"] {
        assert!(temp.board().join(folder).is_dir());
    }
    assert!(!temp.board().join("wont-do").exists());
    assert_eq!(
        fs::read_dir(temp.board().join("backlog")).unwrap().count(),
        1
    );
    assert!(!init(&temp.board(), true).unwrap());
    assert_eq!(
        fs::read_dir(temp.board().join("backlog")).unwrap().count(),
        1
    );
    assert_eq!(
        fs::read_to_string(temp.0.join(".gitignore")).unwrap(),
        ".tickcats/\n"
    );

    let other = TempDir::new();
    fs::create_dir(other.board()).unwrap();
    assert!(init(&other.board(), true).unwrap());
    assert_eq!(
        fs::read_dir(other.board().join("backlog")).unwrap().count(),
        1
    );

    let no_intro = TempDir::new();
    init(&no_intro.board(), false).unwrap();
    assert_eq!(
        fs::read_dir(no_intro.board().join("backlog"))
            .unwrap()
            .count(),
        0
    );

    #[cfg(unix)]
    {
        use std::os::unix::ffi::OsStringExt;
        let invalid = temp.0.join(std::ffi::OsString::from_vec(vec![b'.', 0xff]));
        let error = init(&invalid, false).unwrap_err();
        assert!(
            error.to_string().contains("valid UTF-8"),
            "unexpected error: {error}"
        );
    }
}

#[test]
fn create_defaults_to_refinement_and_adjacent_moves_preserve_bytes() {
    let temp = TempDir::new();
    let path = create(
        &temp.board(),
        Kind::Feature,
        "Ship Rust",
        Priority::P2,
        true,
        Some("It works"),
        Utc.with_ymd_and_hms(2026, 8, 13, 10, 0, 0).unwrap(),
    )
    .unwrap();
    let ticket = parse_markdown(&fs::read(&path).unwrap()).unwrap();
    assert_eq!(ticket.title, "[to refine] Feat: Ship Rust");
    let bytes = fs::read(&path).unwrap();
    let moved = move_ticket(
        &temp.board(),
        path.file_name().unwrap(),
        State::Backlog,
        State::Ready,
    )
    .unwrap();
    assert_eq!(fs::read(moved).unwrap(), bytes);
    assert!(
        move_ticket(
            &temp.board(),
            path.file_name().unwrap(),
            State::Ready,
            State::Done,
        )
        .unwrap_err()
        .to_string()
        .contains("not adjacent")
    );

    #[cfg(unix)]
    {
        use std::os::unix::fs::symlink;
        let outside = temp.0.join("outside.md");
        fs::write(&outside, &bytes).unwrap();
        let linked = temp.board().join("backlog/linked.md");
        symlink(&outside, &linked).unwrap();
        assert!(
            move_ticket(
                &temp.board(),
                std::ffi::OsStr::new("linked.md"),
                State::Backlog,
                State::Ready
            )
            .is_err()
        );
        assert_eq!(fs::read(outside).unwrap(), bytes);
    }
}

#[test]
fn board_warns_about_bad_and_legacy_files_without_mutation() {
    let temp = fixture_board("compat");
    let config = fs::read(temp.board().join("config.json")).unwrap();
    let sort = fs::read(temp.board().join("sort.json")).unwrap();
    let legacy = fs::read(temp.board().join("review/tc-c4d5e6-custom.md")).unwrap();
    let board = load(&temp.board()).unwrap();
    assert_eq!(board.tickets(State::Backlog).len(), 2);
    assert_eq!(board.tickets(State::Ready).len(), 4);
    assert!(
        board
            .warnings
            .iter()
            .any(|warning| warning.message.contains("invalid ticket id"))
    );
    assert_eq!(
        board
            .warnings
            .iter()
            .filter(|warning| warning.message.contains("unsupported legacy column"))
            .count(),
        2
    );
    assert_eq!(fs::read(temp.board().join("config.json")).unwrap(), config);
    assert_eq!(fs::read(temp.board().join("sort.json")).unwrap(), sort);
    assert_eq!(
        fs::read(temp.board().join("review/tc-c4d5e6-custom.md")).unwrap(),
        legacy
    );
}

#[test]
fn pick_next_is_independent_from_matrix_board_order() {
    let temp = fixture_board("matrix-disagreement");
    let board = load(&temp.board()).unwrap();
    let PickResult::One(picked) = pick_next(&board) else {
        panic!("expected unique pick")
    };
    assert_eq!(picked.ticket.priority, Priority::P0);
    let ready = board.tickets(State::Ready);
    let normal = ready
        .iter()
        .find(|ticket| ticket.ticket.priority == Priority::P0)
        .unwrap();
    let matrix = ready
        .iter()
        .find(|ticket| ticket.ticket.priority == Priority::P2)
        .unwrap();
    let now = Utc.with_ymd_and_hms(2026, 8, 13, 0, 0, 0).unwrap();
    assert!(fixed_order_less(matrix, normal, true, now));
}

#[test]
fn important_toggle_and_trash_are_safe() {
    let temp = fixture_board("pick-priority");
    let name = "tc-c3d4e5-p0-newer.md";
    set_important(
        &temp.board(),
        std::ffi::OsStr::new(name),
        State::Ready,
        true,
        Utc.with_ymd_and_hms(2026, 8, 13, 11, 0, 0).unwrap(),
    )
    .unwrap();
    let path = temp.board().join("ready").join(name);
    let mut no_newline = fs::read(&path).unwrap();
    while no_newline.last() == Some(&b'\n') {
        no_newline.pop();
    }
    fs::write(&path, no_newline).unwrap();
    set_important(
        &temp.board(),
        std::ffi::OsStr::new(name),
        State::Ready,
        true,
        Utc.with_ymd_and_hms(2026, 8, 13, 11, 0, 0).unwrap(),
    )
    .unwrap();
    let data = fs::read(&path).unwrap();
    assert!(parse_markdown(&data).unwrap().important);
    assert!(data.ends_with(b"Eligible"));
    let trashed = trash(&temp.board(), std::ffi::OsStr::new(name), State::Ready).unwrap();
    assert!(trashed.starts_with(temp.board().join(".trash")));
    assert!(trash(&temp.board(), std::ffi::OsStr::new(name), State::Ready).is_err());
}

#[test]
fn id_migration_is_preflighted_idempotent_and_skips_legacy_folders() {
    let temp = TempDir::new();
    init(&temp.board(), false).unwrap();
    let legacy = fs::read(
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures/tickets/missing-id.md"),
    )
    .unwrap();
    fs::write(temp.board().join("ready/legacy.md"), &legacy).unwrap();
    fs::create_dir(temp.board().join("review")).unwrap();
    fs::write(temp.board().join("review/legacy.md"), &legacy).unwrap();

    let first = migrate_ids(&temp.board()).unwrap();
    assert_eq!(first.migrated.len(), 1);
    assert_eq!(first.skipped_legacy, vec![("review".to_owned(), 1)]);
    assert_eq!(migrate_ids(&temp.board()).unwrap().migrated.len(), 0);

    let duplicate_blank = TempDir::new();
    init(&duplicate_blank.board(), false).unwrap();
    let raw = b"---\ntitle : Task: Duplicate blank id\nid: TC-A7K9Q2\nid : \npriority: P2\ncreated: 2026-05-12T09:00:00Z\nupdated: 2026-05-12T09:00:00Z\n---\n\n## Acceptance Criteria\n\n- Migrated\n";
    fs::write(duplicate_blank.board().join("ready/blank-id.md"), raw).unwrap();
    let migrated = migrate_ids(&duplicate_blank.board()).unwrap();
    assert_eq!(migrated.migrated.len(), 1);
    let parsed = parse_markdown(&fs::read(&migrated.migrated[0].new_path).unwrap()).unwrap();
    assert_eq!(parsed.id, migrated.migrated[0].id);
    assert_eq!(
        migrate_ids(&duplicate_blank.board())
            .unwrap()
            .migrated
            .len(),
        0
    );

    let resumable = TempDir::new();
    init(&resumable.board(), false).unwrap();
    let generated = create(
        &resumable.board(),
        Kind::Task,
        "Interrupted migration",
        Priority::P2,
        true,
        None,
        Utc.with_ymd_and_hms(2026, 8, 13, 12, 0, 0).unwrap(),
    )
    .unwrap();
    let ticket = parse_markdown(&fs::read(&generated).unwrap()).unwrap();
    let interrupted = resumable.board().join("ready/interrupted.md");
    fs::rename(&generated, &interrupted).unwrap();
    let recovered = migrate_ids(&resumable.board()).unwrap();
    assert_eq!(recovered.migrated.len(), 1);
    assert!(
        recovered.migrated[0]
            .new_path
            .file_name()
            .unwrap()
            .to_string_lossy()
            .starts_with(&ticket.id.to_ascii_lowercase())
    );
    assert_eq!(migrate_ids(&resumable.board()).unwrap().migrated.len(), 0);

    // Long-title ticket: slug is capped so migration succeeds and rerun is idempotent.
    let long_title_board = TempDir::new();
    init(&long_title_board.board(), false).unwrap();
    let long_content = format!(
        "---\ntitle: Task: {}\npriority: P2\ncreated: 2026-08-13T12:00:00Z\nupdated: 2026-08-13T12:00:00Z\n---\n\n## Acceptance Criteria\n\n- Done\n",
        "a".repeat(300)
    );
    fs::write(long_title_board.board().join("ready/long.md"), long_content).unwrap();
    let long_result = migrate_ids(&long_title_board.board()).unwrap();
    assert_eq!(long_result.migrated.len(), 1);
    let new_name = long_result.migrated[0]
        .new_path
        .file_name()
        .unwrap()
        .to_string_lossy();
    assert!(
        new_name.len() <= 255,
        "filename {new_name:?} is {} bytes, want ≤255",
        new_name.len()
    );
    assert_eq!(
        migrate_ids(&long_title_board.board())
            .unwrap()
            .migrated
            .len(),
        0
    );
}

#[test]
fn cli_enforces_new_vocabulary_and_completions() {
    let temp = TempDir::new();
    let board = temp.board();
    let board_text = board.to_str().unwrap();
    let initialized = run(&["--path", board_text, "init", "--no-intro"]);
    assert!(initialized.status.success());

    let alias = run(&["--path", board_text, "new", "feature", "No alias"]);
    assert!(!alias.status.success());
    let created = run(&[
        "--path",
        board_text,
        "new",
        "task",
        "CLI ticket",
        "--ac",
        "Done",
    ]);
    assert!(created.status.success());
    let path = String::from_utf8(created.stdout).unwrap();
    let ticket = parse_markdown(&fs::read(path.trim()).unwrap()).unwrap();
    assert!(ticket.parsed_title.to_refine());

    let columns = run(&["--path", board_text, "__complete", "columns"]);
    assert_eq!(
        String::from_utf8(columns.stdout).unwrap(),
        "backlog\nready\nwip\ndone\n"
    );

    let moved = run(&["--path", board_text, "move", &ticket.id, "backlog", "ready"]);
    assert!(moved.status.success());
}
