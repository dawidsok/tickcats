package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

func TestNavigationClampsColumns(t *testing.T) {
	model := NewModel(emptyBoard())
	model.moveColumn(-1)
	if model.SelectedCol != 0 {
		t.Fatalf("SelectedCol = %d, want 0", model.SelectedCol)
	}
	for range 10 {
		model.moveColumn(1)
	}
	if model.SelectedCol != len(columnOrder)-1 {
		t.Fatalf("SelectedCol = %d, want %d", model.SelectedCol, len(columnOrder)-1)
	}
}

func TestNavigationClampsRows(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		storedTicket("a.md", store.StateBacklog, "Task: a"),
		storedTicket("b.md", store.StateBacklog, "Task: b"),
	}
	model := NewModel(board)
	model.moveRow(-1)
	if model.SelectedRows[store.StateBacklog] != 0 {
		t.Fatalf("row = %d, want 0", model.SelectedRows[store.StateBacklog])
	}
	for range 10 {
		model.moveRow(1)
	}
	if model.SelectedRows[store.StateBacklog] != 1 {
		t.Fatalf("row = %d, want 1", model.SelectedRows[store.StateBacklog])
	}
}

func TestEmptyColumnDoesNotPanic(t *testing.T) {
	model := NewModel(emptyBoard())
	model.moveRow(1)
	view := model.View()
	if !strings.Contains(view, "empty") {
		t.Fatalf("View() = %q, want empty marker", view)
	}
}

func TestPickNextBanner(t *testing.T) {
	board := emptyBoard()
	model := NewModel(board)
	if !strings.Contains(model.View(), "Next: none") {
		t.Fatalf("View() missing no-pick banner")
	}

	board.Columns[store.StateReady] = []store.StoredTicket{storedTicket("a.md", store.StateReady, "Task: a")}
	model = NewModel(board)
	if !strings.Contains(model.View(), "Next: [P2] Task: a") {
		t.Fatalf("View() missing pick-next banner:\n%s", model.View())
	}
}

func TestUpdateQuit(t *testing.T) {
	model := NewModel(emptyBoard())
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("quit command nil")
	}
}

func TestEnterOpensDetailForSelectedTicket(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.Mode != ViewDetail {
		t.Fatalf("Mode = %v, want ViewDetail", got.Mode)
	}
	if !strings.Contains(got.View(), "Task: a") {
		t.Fatalf("detail view missing title:\n%s", got.View())
	}
}

func TestEnterOnEmptyColumnStaysOnBoard(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.Mode != ViewBoard {
		t.Fatalf("Mode = %v, want ViewBoard", got.Mode)
	}
}

func TestEscReturnsToBoard(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.Mode != ViewBoard {
		t.Fatalf("Mode = %v, want ViewBoard", model.Mode)
	}
}

func TestDetailScrollClamps(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(Model)
	if model.DetailScroll != 0 {
		t.Fatalf("DetailScroll = %d, want 0", model.DetailScroll)
	}
	for range 100 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		model = updated.(Model)
	}
	max := len(model.detailLines()) - 1
	if model.DetailScroll != max {
		t.Fatalf("DetailScroll = %d, want %d", model.DetailScroll, max)
	}
}

func emptyBoard() store.Board {
	columns := make(map[store.State][]store.StoredTicket)
	for _, state := range columnOrder {
		columns[state] = []store.StoredTicket{}
	}
	return store.Board{Columns: columns}
}

func storedTicket(name string, state store.State, title string) store.StoredTicket {
	created := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	return store.StoredTicket{
		Name:  name,
		State: state,
		Ticket: ticket.Ticket{
			Title:                 title,
			ParsedTitle:           ticket.ParseTitle(title),
			Priority:              ticket.PriorityP2,
			Created:               created,
			Updated:               created,
			Body:                  "## Context\n\nLong body line 1\nLong body line 2\nLong body line 3\n",
			HasAcceptanceCriteria: true,
		},
	}
}
