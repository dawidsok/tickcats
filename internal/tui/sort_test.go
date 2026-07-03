package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

func TestPrioritySortUsesMatrixWhenEnabled(t *testing.T) {
	now := time.Now().UTC()
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		matrixTicket("neither.md", false, nil, ticket.PriorityP0, now),
		matrixTicket("urgent.md", false, ptrTime(now.AddDate(0, 0, 1)), ticket.PriorityP2, now),
		matrixTicket("important.md", true, nil, ticket.PriorityP2, now),
		matrixTicket("both.md", true, ptrTime(now.AddDate(0, 0, 3)), ticket.PriorityP3, now),
	}

	m := newModelForSort(t, board)
	assertTicketOrder(t, m.Board.Columns[store.StateBacklog], "both.md", "important.md", "urgent.md", "neither.md")
}

func TestPrioritySortCanDisableMatrix(t *testing.T) {
	now := time.Now().UTC()
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		matrixTicket("important-p3.md", true, ptrTime(now.AddDate(0, 0, 1)), ticket.PriorityP3, now),
		matrixTicket("plain-p0.md", false, nil, ticket.PriorityP0, now),
	}
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := store.SaveConfig(root, store.Config{DisableMatrixPrioritisation: true}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := NewModelWithRoot(root, board)
	assertTicketOrder(t, m.Board.Columns[store.StateBacklog], "plain-p0.md", "important-p3.md")
}

func TestDeadlineSortOrdersDeadlinePriorityCreated(t *testing.T) {
	now := time.Now().UTC()
	sameDeadline := now.AddDate(0, 0, 2)
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		matrixTicket("no-deadline.md", false, nil, ticket.PriorityP0, now),
		matrixTicket("future.md", false, ptrTime(now.AddDate(0, 0, 5)), ticket.PriorityP0, now),
		matrixTicket("overdue.md", false, ptrTime(now.AddDate(0, 0, -1)), ticket.PriorityP3, now),
		matrixTicketWithCreated("p2-old.md", false, &sameDeadline, ticket.PriorityP2, now.AddDate(0, 0, -10)),
		matrixTicketWithCreated("p0-new.md", false, &sameDeadline, ticket.PriorityP0, now.AddDate(0, 0, -1)),
		matrixTicketWithCreated("p0-old.md", false, &sameDeadline, ticket.PriorityP0, now.AddDate(0, 0, -2)),
	}

	m := newModelForSort(t, board)
	m.SortMode = store.SortDeadline
	m.applySortToBoard()

	assertTicketOrder(t, m.Board.Columns[store.StateBacklog], "overdue.md", "p0-old.md", "p0-new.md", "p2-old.md", "future.md", "no-deadline.md")
}

func TestSortCycleIncludesDeadline(t *testing.T) {
	m := newModelForSort(t, emptyBoard())
	for _, want := range []store.SortMode{store.SortTitle, store.SortDate, store.SortDeadline} {
		got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		m = got.(Model)
		if m.SortMode != want {
			t.Fatalf("SortMode = %v, want %v", m.SortMode, want)
		}
	}
}

func matrixTicket(name string, important bool, deadline *time.Time, priority ticket.Priority, now time.Time) store.StoredTicket {
	return matrixTicketWithCreated(name, important, deadline, priority, now)
}

func matrixTicketWithCreated(name string, important bool, deadline *time.Time, priority ticket.Priority, created time.Time) store.StoredTicket {
	stored := storedTicket(name, store.StateBacklog, "Task: "+name)
	stored.Ticket.Important = important
	stored.Ticket.Deadline = deadline
	stored.Ticket.Priority = priority
	stored.Ticket.Created = created
	return stored
}

func ptrTime(t time.Time) *time.Time { return &t }

func assertTicketOrder(t *testing.T, tickets []store.StoredTicket, want ...string) {
	t.Helper()
	if len(tickets) < len(want) {
		t.Fatalf("ticket count = %d, want at least %d", len(tickets), len(want))
	}
	for i, name := range want {
		if tickets[i].Name != name {
			t.Fatalf("ticket[%d] = %q, want %q; order=%v", i, tickets[i].Name, name, ticketNames(tickets))
		}
	}
}

func ticketNames(tickets []store.StoredTicket) []string {
	names := make([]string, len(tickets))
	for i, t := range tickets {
		names[i] = t.Name
	}
	return names
}
