package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dawidsok/tickcats/internal/store"
)

func TestParseNewKind(t *testing.T) {
	tests := []struct {
		raw string
		ok  bool
	}{
		{raw: "feat", ok: true},
		{raw: "feature", ok: true},
		{raw: "task", ok: true},
		{raw: "bug", ok: true},
		{raw: "fix", ok: true},
		{raw: "chore", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := parseNewKind(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("parseNewKind() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("parseNewKind() expected error")
			}
		})
	}
}

func TestSplitTitleAndAcceptance(t *testing.T) {
	title, acceptance := splitTitleAndAcceptance([]string{"write", "README", "--ac", "README", "explains", "usage"})
	if got := strings.Join(title, " "); got != "write README" {
		t.Fatalf("title = %q, want write README", got)
	}
	if acceptance != "README explains usage" {
		t.Fatalf("acceptance = %q, want README explains usage", acceptance)
	}
}

func TestParsePickNextArgs(t *testing.T) {
	pathOnly, err := parsePickNextArgs([]string{"--path"})
	if err != nil {
		t.Fatalf("parsePickNextArgs() error = %v", err)
	}
	if !pathOnly {
		t.Fatalf("pathOnly = false, want true")
	}

	if _, err := parsePickNextArgs([]string{"--json"}); err == nil {
		t.Fatalf("parsePickNextArgs() expected error")
	}
}

func TestPickNextPathPrintsOnlyPath(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateReady, "a.md", "Task: a", "2026-05-30T10:00:00Z")

		stdout, stderr, err := captureOutput(func() error { return runPickNext([]string{"--path"}, store.RootDir) })
		if err != nil {
			t.Fatalf("runPickNext() error = %v", err)
		}
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
		want := filepath.Join(store.RootDir, string(store.StateReady), "a.md") + "\n"
		if stdout != want {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	})
}

func TestPickNextPathNoEligibleErrors(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}

		stdout, _, err := captureOutput(func() error { return runPickNext([]string{"--path"}, store.RootDir) })
		if err == nil || !strings.Contains(err.Error(), "no ready ticket found") {
			t.Fatalf("err = %v, want no ready ticket", err)
		}
		if stdout != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
	})
}

func TestPickNextPathTieErrorsWithCandidatePaths(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateReady, "a.md", "Task: a", "2026-05-30T10:00:00Z")
		writeMainTestTicket(t, store.RootDir, store.StateReady, "b.md", "Task: b", "2026-05-30T10:00:00Z")

		stdout, stderr, err := captureOutput(func() error { return runPickNext([]string{"--path"}, store.RootDir) })
		if err == nil || !strings.Contains(err.Error(), "multiple ready tickets tied") {
			t.Fatalf("err = %v, want tie", err)
		}
		if stdout != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
		if !strings.Contains(stderr, filepath.Join(store.RootDir, string(store.StateReady), "a.md")) ||
			!strings.Contains(stderr, filepath.Join(store.RootDir, string(store.StateReady), "b.md")) {
			t.Fatalf("stderr = %q, want candidate paths", stderr)
		}
	})
}

func TestPickNextHumanOutputIncludesID(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicketWithID(t, store.RootDir, store.StateReady, "a.md", "Task: a", "TC-A7K9Q2", "2026-05-30T10:00:00Z")

		stdout, _, err := captureOutput(func() error { return runPickNext(nil, store.RootDir) })
		if err != nil {
			t.Fatalf("runPickNext() error = %v", err)
		}
		if stdout != "a.md  TC-A7K9Q2  [P2] Task: a\n" {
			t.Fatalf("stdout = %q", stdout)
		}
	})
}

func TestPickNextHumanOutputIncludesMissingIDPlaceholder(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateReady, "a.md", "Task: a", "2026-05-30T10:00:00Z")

		stdout, _, err := captureOutput(func() error { return runPickNext(nil, store.RootDir) })
		if err != nil {
			t.Fatalf("runPickNext() error = %v", err)
		}
		if stdout != "a.md  —  [P2] Task: a\n" {
			t.Fatalf("stdout = %q", stdout)
		}
	})
}

func TestMoveAcceptsWontDoState(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateDone, "a.md", "Task: a", "2026-05-30T10:00:00Z")

		_, _, err := captureOutput(func() error { return runMove([]string{"a.md", "done", "Won't Do"}, store.RootDir) })
		if err != nil {
			t.Fatalf("runMove() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(store.RootDir, string(store.StateWontDo), "a.md")); err != nil {
			t.Fatalf("wont-do ticket missing: %v", err)
		}
	})
}

func TestListDisplaysWontDoColumn(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateWontDo, "a.md", "Task: rejected", "2026-05-30T10:00:00Z")

		stdout, _, err := captureOutput(func() error { return runList(store.RootDir) })
		if err != nil {
			t.Fatalf("runList() error = %v", err)
		}
		if !strings.Contains(stdout, "Won't Do\n") || !strings.Contains(stdout, "a.md  —  [P2] Task: rejected") {
			t.Fatalf("stdout missing Won't Do ticket:\n%s", stdout)
		}
	})
}

func TestIDsMigrateCommand(t *testing.T) {
	root := t.TempDir()
	withCwd(t, root, func() {
		if err := store.Init(store.RootDir); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		writeMainTestTicket(t, store.RootDir, store.StateReady, "a.md", "Task: migrate", "2026-05-30T10:00:00Z")

		stdout, _, err := captureOutput(func() error { return runIDs([]string{"migrate"}, store.RootDir) })
		if err != nil {
			t.Fatalf("runIDs() error = %v", err)
		}
		if !strings.Contains(stdout, "Migrated 1 ticket(s)") || !strings.Contains(stdout, "TC-") {
			t.Fatalf("stdout missing migration result:\n%s", stdout)
		}
	})
}

func withCwd(t *testing.T, dir string, fn func()) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	fn()
}

func captureOutput(fn func() error) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	readOut, writeOut, _ := os.Pipe()
	readErr, writeErr, _ := os.Pipe()
	os.Stdout = writeOut
	os.Stderr = writeErr

	err := fn()

	_ = writeOut.Close()
	_ = writeErr.Close()
	stdoutBytes, _ := io.ReadAll(readOut)
	stderrBytes, _ := io.ReadAll(readErr)
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	return string(stdoutBytes), string(stderrBytes), err
}

func writeMainTestTicket(t *testing.T, boardPath string, state store.State, name string, title string, createdRaw string) {
	t.Helper()
	writeMainTestTicketWithID(t, boardPath, state, name, title, "", createdRaw)
}

func writeMainTestTicketWithID(t *testing.T, boardPath string, state store.State, name string, title string, id string, createdRaw string) {
	t.Helper()
	created, err := time.Parse(time.RFC3339, createdRaw)
	if err != nil {
		t.Fatalf("parse created: %v", err)
	}
	idLine := ""
	if id != "" {
		idLine = "id: " + id + "\n"
	}
	content := `---
title: ` + title + `
` + idLine + `priority: P2
created: ` + created.Format(time.RFC3339) + `
updated: ` + created.Format(time.RFC3339) + `
---

## Context

Context.

## Acceptance Criteria
- done
`
	path := filepath.Join(boardPath, string(state), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}
}
