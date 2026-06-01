package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadBoardGroupsTicketsByState(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateBacklog, "a.md", "Task: backlog item")
	writeTicket(t, root, StateReady, "b.md", "Feat: ready item")
	writeTicket(t, root, StateDoing, "c.md", "Bug: doing item")
	writeTicket(t, root, StateDone, "d.md", "Task: done item")
	writeTicket(t, root, StateWontDo, "e.md", "Task: rejected item")

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	if len(board.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", board.Warnings)
	}

	assertColumnTitles(t, board, StateBacklog, []string{"Task: backlog item"})
	assertColumnTitles(t, board, StateReady, []string{"Feat: ready item"})
	assertColumnTitles(t, board, StateDoing, []string{"Bug: doing item"})
	assertColumnTitles(t, board, StateDone, []string{"Task: done item"})
	assertColumnTitles(t, board, StateWontDo, []string{"Task: rejected item"})
}

func TestLoadBoardToleratesMissingWontDoFolder(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	if err := os.RemoveAll(filepath.Join(root, string(StateWontDo))); err != nil {
		t.Fatalf("remove wont-do folder: %v", err)
	}

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	assertColumnTitles(t, board, StateWontDo, []string{})
}

func TestLoadBoardSkipsMalformedTicketsWithWarning(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicket(t, root, StateReady, "valid.md", "Task: valid")
	malformedPath := filepath.Join(root, string(StateReady), "bad.md")
	if err := os.WriteFile(malformedPath, []byte("not frontmatter"), 0o644); err != nil {
		t.Fatalf("write malformed ticket: %v", err)
	}

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	assertColumnTitles(t, board, StateReady, []string{"Task: valid"})
	if len(board.Warnings) != 1 {
		t.Fatalf("Warnings count = %d, want 1", len(board.Warnings))
	}
	if board.Warnings[0].Path != malformedPath {
		t.Fatalf("Warning path = %q, want %q", board.Warnings[0].Path, malformedPath)
	}
}

func TestLoadBoardWarnsOnInvalidDeadline(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	badPath := filepath.Join(root, string(StateReady), "bad-deadline.md")
	content := strings.Replace(ticketContent("Task: bad deadline", time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC), time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)), "updated: 2026-05-30T10:00:00Z\n", "updated: 2026-05-30T10:00:00Z\ndeadline: soon\n", 1)
	if err := os.WriteFile(badPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write bad deadline ticket: %v", err)
	}

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	if len(board.Warnings) != 1 {
		t.Fatalf("Warnings count = %d, want 1", len(board.Warnings))
	}
	if !strings.Contains(board.Warnings[0].Err.Error(), "invalid deadline date") {
		t.Fatalf("warning error = %v, want invalid deadline date", board.Warnings[0].Err)
	}
}

func TestLoadBoardWarnsOnInvalidTicketIDButKeepsTicket(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	path := filepath.Join(root, string(StateReady), "bad-id.md")
	content := strings.Replace(ticketContent("Task: bad id", time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC), time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)), "title: Task: bad id\n", "title: Task: bad id\nid: nope\n", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write bad id ticket: %v", err)
	}

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	assertColumnTitles(t, board, StateReady, []string{"Task: bad id"})
	if len(board.Warnings) != 1 || !strings.Contains(board.Warnings[0].Err.Error(), "invalid ticket id") {
		t.Fatalf("Warnings = %#v, want invalid ticket id", board.Warnings)
	}
}

func TestLoadBoardWarnsOnDuplicateTicketID(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWithID(t, root, StateReady, "a.md", "Task: a", "TC-A7K9Q2")
	writeTicketWithID(t, root, StateReady, "b.md", "Task: b", "TC-A7K9Q2")

	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	assertColumnTitles(t, board, StateReady, []string{"Task: a", "Task: b"})
	if len(board.Warnings) != 1 || !strings.Contains(board.Warnings[0].Err.Error(), "duplicate ticket id") {
		t.Fatalf("Warnings = %#v, want duplicate ticket id", board.Warnings)
	}
}

func TestMoveTicketPreservesContentAndDoesNotUpdateTimestamp(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	created := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 5, 30, 11, 0, 0, 0, time.UTC)
	content := strings.Replace(ticketContent("Task: move me", created, updated), "title: Task: move me\n", "title: Task: move me\nid: TC-A7K9Q2\n", 1)
	content = strings.Replace(content, "updated: 2026-05-30T11:00:00Z\n", "updated: 2026-05-30T11:00:00Z\ndeadline: 2026-06-15\n", 1)
	source := filepath.Join(root, string(StateReady), "move-me.md")
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("write source ticket: %v", err)
	}

	target, err := Move(root, "move-me.md", StateReady, StateDoing)
	if err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if target != filepath.Join(root, string(StateDoing), "move-me.md") {
		t.Fatalf("target = %q", target)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists or stat error = %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target ticket: %v", err)
	}
	if string(got) != content {
		t.Fatalf("moved content changed:\n%s", got)
	}
}

func TestMoveInvalidStateFails(t *testing.T) {
	root := t.TempDir()
	_, err := Move(root, "x.md", StateReady, State("later"))
	if err == nil {
		t.Fatalf("Move() expected error")
	}
	if !strings.Contains(err.Error(), "invalid state") {
		t.Fatalf("error = %q, want invalid state", err)
	}
}

func TestMoveMalformedTicketFails(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	path := filepath.Join(root, string(StateReady), "bad.md")
	if err := os.WriteFile(path, []byte("not frontmatter"), 0o644); err != nil {
		t.Fatalf("write malformed ticket: %v", err)
	}

	_, err := Move(root, "bad.md", StateReady, StateDoing)
	if err == nil {
		t.Fatalf("Move() expected error")
	}
	if !strings.Contains(err.Error(), "parse source ticket") {
		t.Fatalf("error = %q, want parse source ticket", err)
	}
}

func mustInit(t *testing.T, root string) {
	t.Helper()
	if err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
}

func writeTicket(t *testing.T, root string, state State, name string, title string) {
	t.Helper()
	path := filepath.Join(root, string(state), name)
	content := ticketContent(title, time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC), time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket %q: %v", path, err)
	}
}

func writeTicketWithID(t *testing.T, root string, state State, name string, title string, id string) {
	t.Helper()
	path := filepath.Join(root, string(state), name)
	content := strings.Replace(ticketContent(title, time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC), time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)), "title: "+title+"\n", "title: "+title+"\nid: "+id+"\n", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket %q: %v", path, err)
	}
}

func ticketContent(title string, created time.Time, updated time.Time) string {
	return `---
title: ` + title + `
priority: P2
created: ` + created.Format(time.RFC3339) + `
updated: ` + updated.Format(time.RFC3339) + `
---

## Context

Context.

## Acceptance Criteria
- done
`
}

func assertColumnTitles(t *testing.T, board Board, state State, want []string) {
	t.Helper()
	gotTickets := board.Columns[state]
	if len(gotTickets) != len(want) {
		t.Fatalf("column %s length = %d, want %d", state, len(gotTickets), len(want))
	}
	for i := range want {
		if gotTickets[i].Ticket.Title != want[i] {
			t.Fatalf("column %s title[%d] = %q, want %q", state, i, gotTickets[i].Ticket.Title, want[i])
		}
	}
}
