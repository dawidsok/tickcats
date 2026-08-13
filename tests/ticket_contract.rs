use std::fs;
use std::path::Path;

use tickcats::ticket::{Kind, ParsedTitle, Priority, parse_markdown, valid_id};

fn fixture(path: &str) -> Vec<u8> {
    fs::read(
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("tests/fixtures")
            .join(path),
    )
    .expect("read fixture")
}

fn markdown_files(root: &Path, files: &mut Vec<std::path::PathBuf>) {
    let Ok(entries) = fs::read_dir(root) else {
        return;
    };
    for entry in entries {
        let path = entry.unwrap().path();
        if path.is_dir() {
            markdown_files(&path, files);
        } else if path.extension().is_some_and(|extension| extension == "md") {
            files.push(path);
        }
    }
}

fn error(path: &str) -> String {
    parse_markdown(&fixture(path)).expect_err(path).to_string()
}

#[test]
fn parses_compatibility_ticket_fields_and_crlf() {
    let ticket = parse_markdown(&fixture(
        "boards/compat/doing/tc-s4t5v6-important-deadline.md",
    ))
    .unwrap();
    assert_eq!(ticket.id, "TC-S4T5V6");
    assert_eq!(ticket.priority, Priority::P3);
    assert!(ticket.important);
    assert_eq!(ticket.deadline.unwrap().to_string(), "2026-08-10");
    assert!(ticket.has_acceptance_criteria);

    let crlf = parse_markdown(&fixture("boards/compat/done/tc-w4x5y6-crlf.md")).unwrap();
    assert_eq!(crlf.title, "Task: CRLF ticket");
    assert_eq!(
        crlf.body,
        b"\n## Acceptance Criteria\n\n- CRLF input parses\n"
    );
}

#[test]
fn optional_id_and_acceptance_placeholder_match_go() {
    let missing_id = parse_markdown(&fixture("tickets/missing-id.md")).unwrap();
    assert!(missing_id.id.is_empty());

    let placeholder = parse_markdown(&fixture(
        "boards/compat/backlog/tc-d4e5f6-labeled-ticket.md",
    ))
    .unwrap();
    assert!(!placeholder.has_acceptance_criteria);
    assert!(placeholder.parsed_title.to_refine());
    assert!(placeholder.parsed_title.has_label("IDEA"));
}

#[test]
fn quoted_required_values_trim_inner_padding_like_go() {
    let ticket = parse_markdown(&fixture("tickets/padded-required.md")).unwrap();
    assert_eq!(ticket.title, "Task: Padded title");
    assert_eq!(ticket.priority, Priority::P2);
    assert_eq!(ticket.created.to_rfc3339(), "2026-05-12T09:00:00+00:00");
    assert_eq!(ticket.updated.to_rfc3339(), "2026-05-12T10:00:00+00:00");
}

#[test]
fn body_preserves_invalid_utf8_bytes() {
    let mut raw = fixture("tickets/missing-id.md");
    raw.extend_from_slice(&[0xff, b'\n']);
    let ticket = parse_markdown(&raw).unwrap();
    assert!(ticket.body.ends_with(&[0xff, b'\n']));
}

#[test]
fn duplicate_frontmatter_values_use_the_last_value() {
    let ticket = parse_markdown(&fixture("tickets/duplicate-last-wins.md")).unwrap();
    assert_eq!(ticket.title, "Bug: Final title");
    assert_eq!(ticket.priority, Priority::P1);
}

#[test]
fn required_fields_and_values_are_validated() {
    let cases = [
        (
            "tickets/missing-title.md",
            "missing required frontmatter field \"title\"",
        ),
        (
            "tickets/blank-title.md",
            "missing required frontmatter field \"title\"",
        ),
        (
            "tickets/missing-priority.md",
            "missing required frontmatter field \"priority\"",
        ),
        (
            "tickets/blank-priority.md",
            "missing required frontmatter field \"priority\"",
        ),
        ("tickets/invalid-priority.md", "invalid priority"),
        (
            "tickets/missing-created.md",
            "missing required frontmatter field \"created\"",
        ),
        (
            "tickets/blank-created.md",
            "missing required frontmatter field \"created\"",
        ),
        ("tickets/invalid-created.md", "invalid created timestamp"),
        (
            "tickets/missing-updated.md",
            "missing required frontmatter field \"updated\"",
        ),
        (
            "tickets/blank-updated.md",
            "missing required frontmatter field \"updated\"",
        ),
        ("tickets/invalid-updated.md", "invalid updated timestamp"),
        (
            "boards/compat/backlog/malformed.md",
            "missing frontmatter opening fence",
        ),
    ];
    for (path, wanted) in cases {
        assert!(error(path).contains(wanted), "{path}: {}", error(path));
    }
}

#[test]
fn priorities_are_case_insensitive_and_ordered() {
    let ticket = parse_markdown(&fixture("tickets/lowercase-priority.md")).unwrap();
    assert_eq!(ticket.priority, Priority::P1);
    assert_eq!(Priority::P0.rank(), 0);
    assert!(Priority::P0 < Priority::P3);
    assert_eq!(Priority::P2.to_string(), "P2");
}

#[test]
fn title_aliases_labels_and_fallback_match_go() {
    let feature = parse_markdown(&fixture("tickets/title-feature-alias.md")).unwrap();
    assert_eq!(feature.parsed_title.kind, Kind::Feature);
    assert_eq!(feature.parsed_title.normalized(), "Feat: Alias title");

    let fix = parse_markdown(&fixture("tickets/title-fix-alias.md")).unwrap();
    assert_eq!(fix.parsed_title.kind, Kind::Bug);
    assert_eq!(fix.parsed_title.normalized(), "Bug: Alias title");

    let fallback = parse_markdown(&fixture("tickets/title-fallback.md")).unwrap();
    assert_eq!(fallback.parsed_title.kind, Kind::Task);
    assert!(!fallback.parsed_title.had_prefix);
    assert_eq!(fallback.parsed_title.normalized(), "Task: Unprefixed title");

    let blocked = ParsedTitle::parse("[blocked, to refine] Fix: Crash");
    assert!(blocked.blocked());
    assert!(blocked.to_refine());
    assert_eq!(blocked.normalized(), "[blocked, to refine] Bug: Crash");
}

#[test]
fn current_sample_board_tickets_parse_when_available() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join(".tickcats-test");
    if !root.is_dir() {
        return;
    }
    let mut files = Vec::new();
    markdown_files(&root, &mut files);
    assert!(!files.is_empty());
    for path in files {
        parse_markdown(&fs::read(&path).unwrap())
            .unwrap_or_else(|error| panic!("{}: {error}", path.display()));
    }
}

#[test]
fn validates_the_restricted_ticket_id_alphabet() {
    for valid in ["TC-A7K9Q2", "TC-AAAAAA"] {
        assert!(valid_id(valid));
    }
    for invalid in ["tc-A7K9Q2", "TC-A7K9Q", "TC-A7K9O2", "XX-A7K9Q2"] {
        assert!(!valid_id(invalid));
    }
}

#[test]
fn invalid_optional_values_are_rejected() {
    let template = fixture("tickets/missing-id.md");
    let text = String::from_utf8(template).unwrap();
    for (field, wanted) in [
        ("deadline: soon\n", "invalid deadline date"),
        ("important: maybe\n", "invalid important bool"),
    ] {
        let changed = text.replace(
            "updated: 2026-05-12T09:00:00Z\n",
            &format!("updated: 2026-05-12T09:00:00Z\n{field}"),
        );
        assert!(
            parse_markdown(changed.as_bytes())
                .unwrap_err()
                .to_string()
                .contains(wanted)
        );
    }
}
