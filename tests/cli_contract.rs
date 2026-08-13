use std::collections::BTreeSet;
use std::fs;
use std::path::Path;
use std::process::Command;

fn expected_ids() -> BTreeSet<String> {
    let mut ids = BTreeSet::new();
    for (prefix, end) in [
        ("CORE", 14),
        ("CLI", 14),
        ("TUI", 29),
        ("OPS", 12),
        ("DATA", 16),
        ("DEF", 7),
    ] {
        for number in 1..=end {
            ids.insert(format!("{prefix}-{number:02}"));
        }
    }
    ids
}

#[test]
fn manifest_covers_every_approved_contract() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let manifest = fs::read_to_string(root.join("tests/contracts/manifest.tsv"))
        .expect("read contract manifest");
    let mut actual = BTreeSet::new();

    for line in manifest
        .lines()
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
    {
        let fields: Vec<_> = line.split('\t').collect();
        assert_eq!(fields.len(), 4, "invalid manifest row: {line}");
        assert!(
            actual.insert(fields[0].to_owned()),
            "duplicate ID: {}",
            fields[0]
        );
        let phase: u8 = fields[1].parse().expect("numeric phase");
        assert!((2..=6).contains(&phase), "invalid phase in row: {line}");
        assert!(
            root.join("tests/fixtures").join(fields[2]).is_dir(),
            "missing fixture: {}",
            fields[2]
        );
        assert!(
            !fields[3].trim().is_empty(),
            "missing contract text: {line}"
        );
    }

    assert_eq!(actual, expected_ids());
}

#[test]
fn fixture_corpus_covers_the_frozen_edge_cases() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let cases = fs::read_to_string(root.join("tests/contracts/fixture-cases.tsv"))
        .expect("read fixture case manifest");
    let mut tags = BTreeSet::new();

    for line in cases
        .lines()
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
    {
        let fields: Vec<_> = line.split('\t').collect();
        assert_eq!(fields.len(), 3, "invalid fixture case row: {line}");
        assert!(
            root.join("tests/fixtures").join(fields[1]).exists(),
            "missing fixture: {}",
            fields[1]
        );
        for tag in fields[2].split(',') {
            assert!(tags.insert(tag), "duplicate fixture coverage tag: {tag}");
        }
    }

    let required: BTreeSet<_> = [
        "malformed-frontmatter",
        "crlf",
        "labels",
        "priority-p0",
        "priority-p1",
        "priority-p2",
        "priority-p3",
        "valid-id",
        "missing-id",
        "invalid-id",
        "duplicate-id",
        "deadline",
        "important",
        "custom-column",
        "column-color",
        "config-preferences",
        "manual-sort",
        "move-collision",
        "pick-priority",
        "pick-oldest",
        "pick-filename",
        "pick-exact-tie",
        "pick-matrix-disagreement",
        "missing-title",
        "blank-title",
        "missing-priority",
        "blank-priority",
        "invalid-priority",
        "missing-created",
        "blank-created",
        "invalid-created",
        "missing-updated",
        "blank-updated",
        "invalid-updated",
        "lowercase-priority",
        "padded-required",
        "duplicate-last-wins",
        "unknown-frontmatter",
        "title-feature-alias",
        "title-fix-alias",
        "title-fallback",
        "trash",
        "legacy-folder",
        "gitignore-existing-newline",
        "gitignore-existing-entry",
        "gitignore-custom-basename",
    ]
    .into_iter()
    .collect();
    assert_eq!(tags, required);

    let crlf = fs::read(root.join("tests/fixtures/boards/compat/done/tc-w4x5y6-crlf.md"))
        .expect("read CRLF fixture");
    assert!(crlf.windows(2).any(|pair| pair == b"\r\n"));
    assert!(
        !crlf
            .iter()
            .enumerate()
            .any(|(index, byte)| *byte == b'\n' && (index == 0 || crlf[index - 1] != b'\r'))
    );

    let duplicate = fs::read_to_string(root.join("tests/fixtures/tickets/duplicate-last-wins.md"))
        .expect("read duplicate-key fixture");
    assert_eq!(duplicate.matches("title:").count(), 2);
    assert_eq!(duplicate.matches("priority:").count(), 2);
}

#[test]
fn scenario_manifest_references_existing_contract_assets() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let scenarios = fs::read_to_string(root.join("tests/contracts/scenarios.tsv"))
        .expect("read scenario manifest");
    let mut scenario_names = BTreeSet::new();

    for line in scenarios
        .lines()
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
    {
        let fields: Vec<_> = line.split('\t').collect();
        assert!(fields.len() >= 5, "invalid scenario row: {line}");
        assert!(
            scenario_names.insert(fields[0]),
            "duplicate scenario: {}",
            fields[0]
        );
        assert!(
            fields[1] == "-" || root.join("tests/fixtures").join(fields[1]).is_dir(),
            "missing fixture: {}",
            fields[1]
        );
        assert!(
            matches!(fields[2], "parity" | "intent"),
            "invalid policy: {line}"
        );
        assert!(
            matches!(fields[3], "readonly" | "mutation"),
            "invalid filesystem policy: {line}"
        );
        if fields[2] == "intent" {
            let expected = root.join("tests/contracts/expected").join(fields[0]);
            for artifact in ["stdout", "stderr", "status"] {
                assert!(
                    expected.join(artifact).is_file(),
                    "missing {artifact} for {}",
                    fields[0]
                );
            }
            let assertions = expected.join("filesystem.tsv");
            if assertions.is_file() {
                let contents = fs::read_to_string(assertions).expect("read filesystem assertions");
                for assertion in contents
                    .lines()
                    .filter(|line| !line.is_empty() && !line.starts_with('#'))
                {
                    let parts: Vec<_> = assertion.split('\t').collect();
                    assert_eq!(parts.len(), 3, "invalid filesystem assertion: {assertion}");
                    assert!(
                        matches!(
                            parts[0],
                            "dir"
                                | "file"
                                | "not-exists"
                                | "contains"
                                | "glob-count"
                                | "glob-contains"
                        ),
                        "invalid filesystem assertion: {assertion}"
                    );
                }
            }
        }
    }

    let expected_gaps = fs::read_to_string(root.join("tests/contracts/expected-gaps.txt"))
        .expect("read expected-gap allowlist");
    let mut gap_names = BTreeSet::new();
    for name in expected_gaps
        .lines()
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
    {
        assert!(gap_names.insert(name), "duplicate expected gap: {name}");
        assert!(
            scenario_names.contains(name),
            "unknown expected gap: {name}"
        );
    }
}

#[cfg(unix)]
#[test]
fn process_harness_captures_streams_status_and_tree_effects() {
    let status = Command::new("bash")
        .arg("scripts/compare-go-rust.sh")
        .arg("--self-test")
        .current_dir(env!("CARGO_MANIFEST_DIR"))
        .status()
        .expect("run process harness self-test");
    assert!(status.success());
}
