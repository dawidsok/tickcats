package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPickNextSelectsHighestPriorityReadyTicket(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "p2.md", "Task: p2", "P2", atHour(10), "- done")
	writeTicketWith(t, root, StateReady, "p0.md", "Task: p0", "P0", atHour(11), "- done")
	writeTicketWith(t, root, StateReady, "p1.md", "Task: p1", "P1", atHour(9), "- done")

	result := pickFromRoot(t, root)
	if !result.HasPick {
		t.Fatalf("HasPick = false, want true")
	}
	if result.Ticket.Name != "p0.md" {
		t.Fatalf("pick = %q, want p0.md", result.Ticket.Name)
	}
}

func TestPickNextExcludesNonReadyFolder(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateBacklog, "p0.md", "Task: p0", "P0", atHour(10), "- done")
	writeTicketWith(t, root, StateReady, "p2.md", "Task: p2", "P2", atHour(11), "- done")

	result := pickFromRoot(t, root)
	if result.Ticket.Name != "p2.md" {
		t.Fatalf("pick = %q, want p2.md", result.Ticket.Name)
	}
}

func TestPickNextExcludesEmptyAcceptanceCriteria(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "empty.md", "Task: empty", "P0", atHour(10), "-")
	writeTicketWith(t, root, StateReady, "valid.md", "Task: valid", "P1", atHour(11), "- done")

	result := pickFromRoot(t, root)
	if result.Ticket.Name != "valid.md" {
		t.Fatalf("pick = %q, want valid.md", result.Ticket.Name)
	}
}

func TestPickNextExcludesBlockedAndToRefineLabels(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "blocked.md", "[blocked] Task: blocked", "P0", atHour(10), "- done")
	writeTicketWith(t, root, StateReady, "to-refine.md", "[to refine] Task: refine", "P0", atHour(11), "- done")
	writeTicketWith(t, root, StateReady, "valid.md", "Task: valid", "P1", atHour(12), "- done")

	result := pickFromRoot(t, root)
	if result.Ticket.Name != "valid.md" {
		t.Fatalf("pick = %q, want valid.md", result.Ticket.Name)
	}
}

func TestPickNextOldestCreatedWinsWithinPriority(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "newer.md", "Task: newer", "P1", atHour(12), "- done")
	writeTicketWith(t, root, StateReady, "older.md", "Task: older", "P1", atHour(9), "- done")

	result := pickFromRoot(t, root)
	if result.Ticket.Name != "older.md" {
		t.Fatalf("pick = %q, want older.md", result.Ticket.Name)
	}
}

func TestPickNextMissingKindPrefixStillEligible(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "plain.md", "write README", "P1", atHour(10), "- done")

	result := pickFromRoot(t, root)
	if result.Ticket.Name != "plain.md" {
		t.Fatalf("pick = %q, want plain.md", result.Ticket.Name)
	}
}

func TestPickNextReturnsDeterministicTieSet(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	created := atHour(10)
	writeTicketWith(t, root, StateReady, "b.md", "Task: b", "P1", created, "- done")
	writeTicketWith(t, root, StateReady, "a.md", "Task: a", "P1", created, "- done")
	writeTicketWith(t, root, StateReady, "c.md", "Task: c", "P2", created, "- done")

	result := pickFromRoot(t, root)
	if !result.NeedsChoice {
		t.Fatalf("NeedsChoice = false, want true")
	}
	if len(result.Tied) != 2 {
		t.Fatalf("tied count = %d, want 2", len(result.Tied))
	}
	if result.Tied[0].Name != "a.md" || result.Tied[1].Name != "b.md" {
		t.Fatalf("tied = %q, %q; want a.md, b.md", result.Tied[0].Name, result.Tied[1].Name)
	}
}

func TestPickNextNoEligibleTickets(t *testing.T) {
	root := t.TempDir()
	mustInit(t, root)
	writeTicketWith(t, root, StateReady, "blocked.md", "[blocked] Task: blocked", "P0", atHour(10), "- done")

	result := pickFromRoot(t, root)
	if result.HasPick {
		t.Fatalf("HasPick = true, want false")
	}
}

func pickFromRoot(t *testing.T, root string) PickResult {
	t.Helper()
	board, err := LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	if len(board.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", board.Warnings)
	}
	return PickNext(board)
}

func writeTicketWith(t *testing.T, root string, state State, name string, title string, priority string, created time.Time, acceptance string) {
	t.Helper()
	path := filepath.Join(root, StateDir(state), name)
	content := `---
title: ` + title + `
priority: ` + priority + `
created: ` + created.Format(time.RFC3339) + `
updated: ` + created.Format(time.RFC3339) + `
---

## Context

Context.

## Acceptance Criteria
` + acceptance + `
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket %q: %v", path, err)
	}
}

func atHour(hour int) time.Time {
	return time.Date(2026, 5, 30, hour, 0, 0, 0, time.UTC)
}
