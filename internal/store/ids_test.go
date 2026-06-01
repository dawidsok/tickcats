package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateIDsAddsIDsAndRenamesFiles(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateReady, "old-name.md", "Task: Old Name")

	result, err := MigrateIDs(root)
	if err != nil {
		t.Fatalf("MigrateIDs() error = %v", err)
	}
	if len(result.Migrated) != 1 {
		t.Fatalf("migrated count = %d, want 1", len(result.Migrated))
	}
	migration := result.Migrated[0]
	if migration.OldPath != filepath.Join(root, string(StateReady), "old-name.md") {
		t.Fatalf("OldPath = %q", migration.OldPath)
	}
	if !strings.HasPrefix(filepath.Base(migration.NewPath), strings.ToLower(migration.ID)+"-") {
		t.Fatalf("NewPath = %q, want lowercase ID prefix %q", migration.NewPath, strings.ToLower(migration.ID))
	}
	if _, err := os.Stat(migration.OldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists or stat error = %v", err)
	}
	data, err := os.ReadFile(migration.NewPath)
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	if !strings.Contains(string(data), "id: "+migration.ID) {
		t.Fatalf("migrated content missing id %q:\n%s", migration.ID, data)
	}

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	if got := board.Columns[StateReady][0].Ticket.ID; got != migration.ID {
		t.Fatalf("loaded ID = %q, want %q", got, migration.ID)
	}
}

func TestMigrateIDsIsIdempotentForTicketsWithIDs(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWithID(t, root, StateReady, "tc-a7k9q2-has-id.md", "Task: Has ID", "TC-A7K9Q2")

	result, err := MigrateIDs(root)
	if err != nil {
		t.Fatalf("MigrateIDs() error = %v", err)
	}
	if len(result.Migrated) != 0 {
		t.Fatalf("migrated count = %d, want 0", len(result.Migrated))
	}
	if _, err := os.Stat(filepath.Join(root, string(StateReady), "tc-a7k9q2-has-id.md")); err != nil {
		t.Fatalf("ticket renamed unexpectedly: %v", err)
	}
}

func TestMigrateIDsPreservesWorkflowFolder(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateDoing, "doing-ticket.md", "Task: Doing Ticket")

	result, err := MigrateIDs(root)
	if err != nil {
		t.Fatalf("MigrateIDs() error = %v", err)
	}
	if len(result.Migrated) != 1 {
		t.Fatalf("migrated count = %d, want 1", len(result.Migrated))
	}
	if filepath.Dir(result.Migrated[0].NewPath) != filepath.Join(root, string(StateDoing)) {
		t.Fatalf("NewPath = %q, want doing folder", result.Migrated[0].NewPath)
	}
}

func TestMigrateIDsBlocksDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWithID(t, root, StateReady, "a.md", "Task: a", "TC-A7K9Q2")
	writeTicketWithID(t, root, StateReady, "b.md", "Task: b", "TC-A7K9Q2")

	_, err := MigrateIDs(root)
	if err == nil || !strings.Contains(err.Error(), "duplicate ticket ids") {
		t.Fatalf("err = %v, want duplicate ticket ids", err)
	}
}
