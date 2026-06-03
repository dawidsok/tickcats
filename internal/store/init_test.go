package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesBoardFolders(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")

	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for _, state := range ValidStates {
		path := filepath.Join(boardRoot, string(state))
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
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}

	ticketPath := filepath.Join(boardRoot, string(StateBacklog), "example.md")
	content := []byte("hello")
	if err := os.WriteFile(ticketPath, content, 0o644); err != nil {
		t.Fatalf("write ticket fixture: %v", err)
	}

	if err := Init(boardRoot); err != nil {
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

func TestInitDoesNotRecreateRemovedColumnsOnExistingBoard(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}
	for _, state := range []State{StateDoing, StateWontDo} {
		if err := os.RemoveAll(filepath.Join(boardRoot, string(state))); err != nil {
			t.Fatalf("remove %s: %v", state, err)
		}
	}

	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() second error = %v", err)
	}

	for _, state := range []State{StateDoing, StateWontDo} {
		path := filepath.Join(boardRoot, string(state))
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Init recreated removed column %q; stat err = %v", state, err)
		}
	}
}

func TestInitCreatesGitignore(t *testing.T) {
	tempDir := t.TempDir()
	boardRoot := filepath.Join(tempDir, ".tickcats")

	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	assertGitignoreEntryCount(t, filepath.Join(tempDir, ".gitignore"), ".tickcats/", 1)
}

func TestInitAppendsGitignoreOnce(t *testing.T) {
	tempDir := t.TempDir()
	boardRoot := filepath.Join(tempDir, ".tickcats")
	gitignore := filepath.Join(tempDir, ".gitignore")
	initial := "dist/\nnode_modules/\n"
	if err := os.WriteFile(gitignore, []byte(initial), 0o644); err != nil {
		t.Fatalf("write .gitignore fixture: %v", err)
	}

	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() first error = %v", err)
	}
	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() second error = %v", err)
	}

	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.HasPrefix(string(data), initial) {
		t.Fatalf(".gitignore = %q, want prefix %q", data, initial)
	}
	assertGitignoreEntryCount(t, gitignore, ".tickcats/", 1)
}

func TestInitAlternatePath(t *testing.T) {
	tempDir := t.TempDir()
	boardRoot := filepath.Join(tempDir, ".tickcats-test")

	if err := Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for _, state := range ValidStates {
		path := filepath.Join(boardRoot, string(state))
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("expected state dir %q", path)
		}
	}

	assertGitignoreEntryCount(t, filepath.Join(tempDir, ".gitignore"), ".tickcats-test/", 1)
}

func assertGitignoreEntryCount(t *testing.T, path string, entry string, want int) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	got := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			got++
		}
	}
	if got != want {
		t.Fatalf("%q count = %d, want %d in .gitignore:\n%s", entry, got, want, data)
	}
}
