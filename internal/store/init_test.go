package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesBoardFolders(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for _, state := range ValidStates {
		path := filepath.Join(root, StateDir(state))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected state dir %q: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not directory", path)
		}
	}
}

func TestInitIsIdempotentAndPreservesTickets(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}

	ticketPath := filepath.Join(root, StateDir(StateBacklog), "example.md")
	content := []byte("hello")
	if err := os.WriteFile(ticketPath, content, 0o644); err != nil {
		t.Fatalf("write ticket fixture: %v", err)
	}

	if err := Init(root); err != nil {
		t.Fatalf("Init() second error = %v", err)
	}

	got, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read preserved ticket: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("ticket content = %q, want %q", got, content)
	}
}

func TestInitCreatesGitignore(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	assertGitignoreEntryCount(t, filepath.Join(root, ".gitignore"), 1)
}

func TestInitAppendsGitignoreOnce(t *testing.T) {
	root := t.TempDir()
	gitignore := filepath.Join(root, ".gitignore")
	initial := "dist/\nnode_modules/\n"
	if err := os.WriteFile(gitignore, []byte(initial), 0o644); err != nil {
		t.Fatalf("write .gitignore fixture: %v", err)
	}

	if err := Init(root); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}
	if err := Init(root); err != nil {
		t.Fatalf("Init() second error = %v", err)
	}

	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.HasPrefix(string(data), initial) {
		t.Fatalf(".gitignore = %q, want prefix %q", data, initial)
	}
	assertGitignoreEntryCount(t, gitignore, 1)
}

func assertGitignoreEntryCount(t *testing.T, path string, want int) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	got := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == gitignoreEntry {
			got++
		}
	}
	if got != want {
		t.Fatalf("%q count = %d, want %d in .gitignore:\n%s", gitignoreEntry, got, want, data)
	}
}
