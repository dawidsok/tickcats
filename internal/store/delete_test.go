package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrashMovesTicketToTrash(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateReady, "a.md", "Task: a")

	target, err := Trash(root, "a.md", StateReady)
	if err != nil {
		t.Fatalf("Trash() error = %v", err)
	}
	if target != filepath.Join(root, TrashDir, "a.md") {
		t.Fatalf("target = %q", target)
	}
	if _, err := os.Stat(filepath.Join(root, string(StateReady), "a.md")); !os.IsNotExist(err) {
		t.Fatalf("source still exists or stat error = %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("trash target missing: %v", err)
	}
}

func TestTrashInvalidStateFails(t *testing.T) {
	_, err := Trash(t.TempDir(), "a.md", State("later"))
	if err == nil || !strings.Contains(err.Error(), "invalid state") {
		t.Fatalf("err = %v, want invalid state", err)
	}
}

func TestTrashMalformedTicketFails(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	path := filepath.Join(root, string(StateReady), "bad.md")
	if err := os.WriteFile(path, []byte("not frontmatter"), 0o644); err != nil {
		t.Fatalf("write malformed ticket: %v", err)
	}

	_, err := Trash(root, "bad.md", StateReady)
	if err == nil || !strings.Contains(err.Error(), "parse source ticket") {
		t.Fatalf("err = %v, want parse source ticket", err)
	}
}

func TestTrashNameCollisionFails(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateReady, "a.md", "Task: a")
	trashDir := filepath.Join(root, TrashDir)
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		t.Fatalf("mkdir trash: %v", err)
	}
	if err := os.WriteFile(filepath.Join(trashDir, "a.md"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write existing trash: %v", err)
	}

	_, err := Trash(root, "a.md", StateReady)
	if err == nil || !strings.Contains(err.Error(), "trash ticket already exists") {
		t.Fatalf("err = %v, want collision", err)
	}
}
