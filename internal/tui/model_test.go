package tui

import (
	"os"
	"path/filepath"
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

func TestDOpensDetailForSelectedTicket(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := updated.(Model)
	if got.Mode != ViewDetail {
		t.Fatalf("Mode = %v, want ViewDetail", got.Mode)
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

func TestMoveSelectedRightMovesTicketOnDisk(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	model := enterMoveMode(t, NewModelWithRoot(root, board))

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(Model)

	if got.SelectedCol != 1 {
		t.Fatalf("SelectedCol = %d, want 1", got.SelectedCol)
	}
	if len(got.Board.Columns[store.StateBacklog]) != 0 {
		t.Fatalf("backlog count = %d, want 0", len(got.Board.Columns[store.StateBacklog]))
	}
	if len(got.Board.Columns[store.StateReady]) != 1 {
		t.Fatalf("ready count = %d, want 1", len(got.Board.Columns[store.StateReady]))
	}
	if _, err := os.Stat(filepath.Join(root, store.StateDir(store.StateReady), "a.md")); err != nil {
		t.Fatalf("ready ticket missing: %v", err)
	}
	if !strings.Contains(got.Status, "Moved a.md to ready") {
		t.Fatalf("Status = %q", got.Status)
	}
	if got.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want board after move", got.InteractionMode)
	}
}

func TestMoveSelectedLeftMovesTicketOnDisk(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateReady, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	model := NewModelWithRoot(root, board)
	model.SelectedCol = 1
	model = enterMoveMode(t, model)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(Model)

	if got.SelectedCol != 0 {
		t.Fatalf("SelectedCol = %d, want 0", got.SelectedCol)
	}
	if len(got.Board.Columns[store.StateReady]) != 0 {
		t.Fatalf("ready count = %d, want 0", len(got.Board.Columns[store.StateReady]))
	}
	if len(got.Board.Columns[store.StateBacklog]) != 1 {
		t.Fatalf("backlog count = %d, want 1", len(got.Board.Columns[store.StateBacklog]))
	}
	if _, err := os.Stat(filepath.Join(root, store.StateDir(store.StateBacklog), "a.md")); err != nil {
		t.Fatalf("backlog ticket missing: %v", err)
	}
	if !strings.Contains(got.Status, "Moved a.md to backlog") {
		t.Fatalf("Status = %q", got.Status)
	}
}

func TestMoveSelectedRightEmptyColumn(t *testing.T) {
	model := enterMoveMode(t, NewModel(emptyBoard()))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(Model)
	if got.Status != "No ticket selected" {
		t.Fatalf("Status = %q, want no selection", got.Status)
	}
}

func TestMoveSelectedRightDoneNoop(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateDone] = []store.StoredTicket{storedTicket("a.md", store.StateDone, "Task: a")}
	model := NewModel(board)
	model.SelectedCol = 3
	model = enterMoveMode(t, model)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(Model)
	if got.Status != "Ticket already done" {
		t.Fatalf("Status = %q, want already done", got.Status)
	}
}

func TestMoveSelectedLeftBacklogNoop(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := enterMoveMode(t, NewModel(board))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := updated.(Model)
	if got.Status != "Ticket already in backlog" {
		t.Fatalf("Status = %q, want already in backlog", got.Status)
	}
}

func TestMoveModeEscReturnsToBoardMode(t *testing.T) {
	model := enterMoveMode(t, NewModel(emptyBoard()))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want board", got.InteractionMode)
	}
}

func TestMoveModeReorderNotImplemented(t *testing.T) {
	model := enterMoveMode(t, NewModel(emptyBoard()))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.Status != "Manual reorder not implemented yet" {
		t.Fatalf("Status = %q, want reorder message", got.Status)
	}
}

func TestEditKeyShowsMessage(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	got := updated.(Model)
	if got.Status != "Edit mode not implemented yet; later opens $EDITOR" {
		t.Fatalf("Status = %q, want edit message", got.Status)
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

func enterMoveMode(t *testing.T, model Model) Model {
	t.Helper()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	got := updated.(Model)
	if got.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v, want move", got.InteractionMode)
	}
	return got
}

func writeTUITestTicket(t *testing.T, root string, state store.State, name string, title string) {
	t.Helper()
	content := `---
title: ` + title + `
priority: P2
created: 2026-05-30T10:00:00Z
updated: 2026-05-30T10:00:00Z
---

## Context

Context.

## Acceptance Criteria
- done
`
	path := filepath.Join(root, store.StateDir(state), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
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
