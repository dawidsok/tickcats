package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddColumnCreatesFolderAndConfigEntry(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)

	if err := AddColumn(root, "Code Review"); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}

	if info, err := os.Stat(filepath.Join(root, "code-review")); err != nil || !info.IsDir() {
		t.Fatalf("code-review folder missing or not directory: info=%v err=%v", info, err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !hasColumn(cfg.Columns, "code-review", "Code Review") {
		t.Fatalf("added column missing from config: %#v", cfg.Columns)
	}
}

func TestAddColumnRejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)

	if err := AddColumn(root, "Ready"); err == nil {
		t.Fatal("AddColumn() expected duplicate error")
	}
}

func TestRenameColumnRenamesFolderAndConfigEntry(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	if err := AddColumn(root, "Code Review"); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "code-review", "a.md"), []byte("ticket"), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	if err := RenameColumn(root, "code-review", "QA Testing"); err != nil {
		t.Fatalf("RenameColumn() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "code-review")); !os.IsNotExist(err) {
		t.Fatalf("old folder still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "qa-testing", "a.md")); err != nil {
		t.Fatalf("renamed folder/ticket missing: %v", err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if hasColumn(cfg.Columns, "code-review", "") || !hasColumn(cfg.Columns, "qa-testing", "QA Testing") {
		t.Fatalf("config not renamed: %#v", cfg.Columns)
	}
}

func TestRenameColumnRejectsDuplicateTarget(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	if err := AddColumn(root, "Code Review"); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}

	if err := RenameColumn(root, "code-review", "Ready"); err == nil {
		t.Fatal("RenameColumn() expected duplicate target error")
	}
}

func TestReorderColumnsPersistsOrder(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	if err := AddColumn(root, "Code Review"); err != nil {
		t.Fatalf("AddColumn() error = %v", err)
	}

	newOrder := []string{"backlog", "code-review", "ready", "doing", "done", "wont-do"}
	if err := ReorderColumns(root, newOrder); err != nil {
		t.Fatalf("ReorderColumns() error = %v", err)
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	for i, want := range newOrder {
		if got := cfg.Columns[i].ID; got != want {
			t.Fatalf("column[%d] = %q, want %q; columns=%#v", i, got, want, cfg.Columns)
		}
	}
}

func TestReorderColumnsRejectsMissingDuplicateAndUnknownIDs(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)

	tests := []struct {
		name  string
		order []string
	}{
		{name: "missing", order: []string{"backlog"}},
		{name: "duplicate", order: []string{"backlog", "backlog", "doing", "done", "wont-do"}},
		{name: "unknown", order: []string{"backlog", "ready", "doing", "done", "elsewhere"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReorderColumns(root, tt.order); err == nil {
				t.Fatal("ReorderColumns() expected error")
			}
		})
	}
}

func TestDeleteColumnMigratesTicketsAndRemovesFolderAndConfigEntry(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	if err := os.WriteFile(filepath.Join(root, "ready", "a.md"), []byte("ticket"), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}

	if err := DeleteColumn(root, "ready"); err != nil {
		t.Fatalf("DeleteColumn() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "ready")); !os.IsNotExist(err) {
		t.Fatalf("ready folder still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "backlog", "a.md")); err != nil {
		t.Fatalf("migrated ticket missing from first column: %v", err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if hasColumn(cfg.Columns, "ready", "") {
		t.Fatalf("deleted column still in config: %#v", cfg.Columns)
	}
}

func TestDeleteColumnRejectsFirstColumn(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)

	if err := DeleteColumn(root, "backlog"); err == nil {
		t.Fatal("DeleteColumn() expected first-column error")
	}
	if _, err := os.Stat(filepath.Join(root, "backlog")); err != nil {
		t.Fatalf("backlog folder should remain: %v", err)
	}
}
