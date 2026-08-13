use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use serde_json::Value;
use tickcats::store::config::Config;

static NEXT_TEMP: AtomicU64 = AtomicU64::new(0);

struct TempDir(PathBuf);

impl TempDir {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "tickcats-rust-config-{}-{}",
            std::process::id(),
            NEXT_TEMP.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&path).unwrap();
        Self(path)
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

fn compatibility_board() -> TempDir {
    let temp = TempDir::new();
    let fixtures = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures/boards/compat");
    fs::copy(fixtures.join("config.json"), temp.0.join("config.json")).unwrap();
    fs::copy(fixtures.join("sort.json"), temp.0.join("sort.json")).unwrap();
    temp
}

#[test]
fn loading_config_and_ignored_sort_are_read_only() {
    let board = compatibility_board();
    let config_before = fs::read(board.0.join("config.json")).unwrap();
    let sort_before = fs::read(board.0.join("sort.json")).unwrap();

    let config = Config::load(&board.0).unwrap();
    assert!(config.matrix_enabled());
    assert_eq!(
        fs::read(board.0.join("config.json")).unwrap(),
        config_before
    );
    assert_eq!(fs::read(board.0.join("sort.json")).unwrap(), sort_before);
}

#[test]
fn matrix_toggle_preserves_every_known_legacy_field() {
    let board = compatibility_board();
    let sort_before = fs::read(board.0.join("sort.json")).unwrap();
    let before: Value =
        serde_json::from_slice(&fs::read(board.0.join("config.json")).unwrap()).unwrap();

    let mut config = Config::load(&board.0).unwrap();
    assert!(!config.toggle_matrix().unwrap());
    config.save().unwrap();

    let after: Value =
        serde_json::from_slice(&fs::read(board.0.join("config.json")).unwrap()).unwrap();
    assert_eq!(after["disable_matrix_prioritisation"], true);
    for field in ["editor", "theme", "skip_editor_prompt", "columns"] {
        assert_eq!(after[field], before[field], "changed legacy field {field}");
    }
    assert_eq!(after["columns"][0]["color"], "#88a");
    assert_eq!(fs::read(board.0.join("sort.json")).unwrap(), sort_before);

    let mut reloaded = Config::load(&board.0).unwrap();
    assert!(!reloaded.matrix_enabled());
    assert!(reloaded.toggle_matrix().unwrap());
    reloaded.save().unwrap();
    let enabled: Value =
        serde_json::from_slice(&fs::read(board.0.join("config.json")).unwrap()).unwrap();
    assert!(enabled.get("disable_matrix_prioritisation").is_none());
    assert_eq!(enabled["columns"], before["columns"]);
}

#[test]
fn missing_config_defaults_to_matrix_enabled_without_writing() {
    let board = TempDir::new();
    let config = Config::load(&board.0).unwrap();
    assert!(config.matrix_enabled());
    assert!(!board.0.join("config.json").exists());
}

#[test]
fn current_sample_board_config_loads_without_writes_when_available() {
    let board = Path::new(env!("CARGO_MANIFEST_DIR")).join(".tickcats-test");
    if !board.join("config.json").is_file() {
        return;
    }
    let before = fs::read(board.join("config.json")).unwrap();
    Config::load(&board).unwrap();
    assert_eq!(fs::read(board.join("config.json")).unwrap(), before);
}

#[test]
fn legacy_null_and_zero_value_columns_remain_go_compatible() {
    let board = TempDir::new();
    let raw = br#"{
  "editor": null,
  "theme": null,
  "skip_editor_prompt": null,
  "disable_matrix_prioritisation": null,
  "columns": [null, {}, {"id": null, "name": null, "color": null}]
}"#;
    fs::write(board.0.join("config.json"), raw).unwrap();

    let mut config = Config::load(&board.0).unwrap();
    assert!(config.matrix_enabled());
    assert!(!config.toggle_matrix().unwrap());
    config.save().unwrap();
    let saved: Value =
        serde_json::from_slice(&fs::read(board.0.join("config.json")).unwrap()).unwrap();
    assert_eq!(saved["editor"], Value::Null);
    assert_eq!(saved["columns"][0], Value::Null);
    assert_eq!(saved["columns"][1], serde_json::json!({}));
    assert_eq!(saved["columns"][2]["color"], Value::Null);

    fs::write(board.0.join("config.json"), br#"{"columns":null}"#).unwrap();
    let mut null_columns = Config::load(&board.0).unwrap();
    assert!(!null_columns.toggle_matrix().unwrap());
    null_columns.save().unwrap();
    let saved: Value =
        serde_json::from_slice(&fs::read(board.0.join("config.json")).unwrap()).unwrap();
    assert_eq!(saved["columns"], Value::Null);
}

#[test]
fn malformed_or_wrongly_typed_known_config_is_rejected() {
    let board = TempDir::new();
    fs::write(board.0.join("config.json"), br#"{"theme":"fire"}"#).unwrap();
    assert!(
        Config::load(&board.0)
            .unwrap_err()
            .to_string()
            .contains("theme")
    );

    fs::write(board.0.join("config.json"), b"[]").unwrap();
    assert!(
        Config::load(&board.0)
            .unwrap_err()
            .to_string()
            .contains("object")
    );
}
