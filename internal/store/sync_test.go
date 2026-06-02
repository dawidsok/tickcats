package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncConfigColumnsRemovesMissingFolders(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"backlog", "ready", "doing"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	cfg := Config{Columns: DefaultColumns()}
	synced, updated, err := SyncConfigColumns(root, cfg)
	if err != nil {
		t.Fatalf("SyncConfigColumns() error = %v", err)
	}
	if !updated {
		t.Fatal("updated = false, want true")
	}
	if got, want := len(synced.Columns), 3; got != want {
		t.Fatalf("column count = %d, want %d", got, want)
	}
	for _, col := range synced.Columns {
		if col.ID == "done" || col.ID == "wont-do" {
			t.Fatalf("missing disk folder %q was kept in config", col.ID)
		}
	}
}

func TestSyncConfigColumnsAddsDiscoveredFolders(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"backlog", "ready", "doing", "done", "wont-do", "code-review", ".trash"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	synced, updated, err := SyncConfigColumns(root, Config{})
	if err != nil {
		t.Fatalf("SyncConfigColumns() error = %v", err)
	}
	if !updated {
		t.Fatal("updated = false, want true")
	}
	if got, want := len(synced.Columns), 6; got != want {
		t.Fatalf("column count = %d, want %d", got, want)
	}
	if !hasColumn(synced.Columns, "code-review", "Code Review") {
		t.Fatalf("discovered folder was not added: %#v", synced.Columns)
	}
	if hasColumn(synced.Columns, ".trash", "") {
		t.Fatalf("ignored .trash folder was added: %#v", synced.Columns)
	}
}

func TestLoadBoardScansConfiguredColumns(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"backlog", "triage", "done"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	if err := SaveConfig(root, Config{Columns: []Column{
		{ID: "backlog", DisplayName: "Backlog"},
		{ID: "triage", DisplayName: "Triage"},
		{ID: "done", DisplayName: "Done"},
	}}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	writeTicket(t, root, State("triage"), "triage.md", "Task: custom column")

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	assertColumnTitles(t, board, State("triage"), []string{"Task: custom column"})
	if _, ok := board.Columns[StateReady]; ok {
		t.Fatalf("ready column loaded even though it is absent from config: %#v", board.Columns)
	}
}

func hasColumn(columns []Column, id string, displayName string) bool {
	for _, col := range columns {
		if col.ID != id {
			continue
		}
		return displayName == "" || col.DisplayName == displayName
	}
	return false
}
