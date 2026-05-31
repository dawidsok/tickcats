package tui

import (
	"fmt"
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

func TestColumnWrapsLongTicketsAndSeparatesTickets(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		storedTicket("a.md", store.StateBacklog, "Task: a very long ticket title that should wrap instead of overflowing horizontally"),
		storedTicket("b.md", store.StateBacklog, "Task: second ticket"),
	}
	model := NewModel(board)
	model.Width = 80
	view := model.View()
	if !strings.Contains(view, "overflowing") || !strings.Contains(view, "horizontally") {
		t.Fatalf("View() missing wrapped title text:\n%s", view)
	}
	if !strings.Contains(view, "────") {
		t.Fatalf("View() missing ticket separator:\n%s", view)
	}
}

func TestColumnScrollIndicators(t *testing.T) {
	board := emptyBoard()
	for i := 0; i < 8; i++ {
		board.Columns[store.StateBacklog] = append(board.Columns[store.StateBacklog], storedTicket(fmt.Sprintf("%d.md", i), store.StateBacklog, fmt.Sprintf("Task: %d", i)))
	}
	model := NewModel(board)
	model.Height = 12
	view := model.View()
	if !strings.Contains(view, "↓") || !strings.Contains(view, "below") {
		t.Fatalf("View() missing below indicator:\n%s", view)
	}
	for range 7 {
		model.moveRow(1)
	}
	view = model.View()
	if !strings.Contains(view, "↑") || !strings.Contains(view, "above") {
		t.Fatalf("View() missing above indicator:\n%s", view)
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

func TestWindowSizeUpdatesModel(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := updated.(Model)
	if got.Width != 120 || got.Height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", got.Width, got.Height)
	}
	if got.columnWidth() != 28 {
		t.Fatalf("columnWidth = %d, want 28", got.columnWidth())
	}
}

func TestBoardRendersColumnBorders(t *testing.T) {
	model := NewModel(emptyBoard())
	view := model.View()
	if !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Fatalf("View() missing borders:\n%s", view)
	}
}

func TestDetailViewRendersContentAndMetadataColumns(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)

	view := model.View()
	if !strings.Contains(view, "Long body line 1") {
		t.Fatalf("detail missing body:\n%s", view)
	}
	if !strings.Contains(view, "Metadata") || !strings.Contains(view, "Title: Task: a") || !strings.Contains(view, "State: backlog") || !strings.Contains(view, "File: a.md") {
		t.Fatalf("detail missing metadata:\n%s", view)
	}
	if !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Fatalf("detail missing borders:\n%s", view)
	}
}

func TestDetailWidthsSplitTwoThirdsOneThird(t *testing.T) {
	model := NewModel(emptyBoard())
	model.Width = 120
	content, metadata := model.detailWidths()
	if content != 77 || metadata != 40 {
		t.Fatalf("widths = %d/%d, want 77/40", content, metadata)
	}
}

func TestPickNextBannerHasBorder(t *testing.T) {
	model := NewModel(emptyBoard())
	view := model.View()
	if !strings.Contains(view, "┌") || !strings.Contains(view, "Next: none") {
		t.Fatalf("View() missing bordered pick-next:\n%s", view)
	}
}

func TestFooterHasSeparator(t *testing.T) {
	model := NewModel(emptyBoard())
	view := model.View()
	if !strings.Contains(view, "────") || !strings.Contains(view, "BOARD MODE") {
		t.Fatalf("View() missing footer separator:\n%s", view)
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

func TestOOpensDetailForSelectedTicket(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	got := updated.(Model)
	if got.Mode != ViewDetail {
		t.Fatalf("Mode = %v, want ViewDetail", got.Mode)
	}
}

func TestDetailModeEKeyOpensEditor(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Enter detail mode
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := got.(Model)
	if m2.Mode != ViewDetail {
		t.Fatalf("Mode = %v, want ViewDetail", m2.Mode)
	}

	// Press e — should return an editor cmd
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("e in detail mode returned nil cmd, want editor command")
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
	if _, err := os.Stat(filepath.Join(root, string(store.StateReady), "a.md")); err != nil {
		t.Fatalf("ready ticket missing: %v", err)
	}
	if !strings.Contains(got.Status, "Moved a.md to ready") {
		t.Fatalf("Status = %q", got.Status)
	}
	if got.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v, want move after move", got.InteractionMode)
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
	if _, err := os.Stat(filepath.Join(root, string(store.StateBacklog), "a.md")); err != nil {
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

func TestMoveModeJKInNonManualPromptsSortSwitch(t *testing.T) {
	model := enterMoveMode(t, newModelForSort(t, emptyBoard()))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.InteractionMode != InteractionSortPrompt {
		t.Fatalf("InteractionMode = %v, want InteractionSortPrompt", got.InteractionMode)
	}
	if !strings.Contains(got.Status, "manual sort") {
		t.Fatalf("Status = %q, want sort prompt message", got.Status)
	}
}

func TestEditKeyNoSelectionShowsMessage(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	got := updated.(Model)
	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if got.Status != "No ticket selected" {
		t.Fatalf("Status = %q, want no selection", got.Status)
	}
}

func TestEditorFinishedReloadsBoard(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	model := NewModelWithRoot(root, board)
	writeTUITestTicket(t, root, store.StateReady, "b.md", "Task: b")

	updated, _ := model.Update(editorFinishedMsg{})
	got := updated.(Model)
	if got.Status != "Edited ticket" {
		t.Fatalf("Status = %q, want edited", got.Status)
	}
	if len(got.Board.Columns[store.StateReady]) != 1 {
		t.Fatalf("ready count = %d, want 1", len(got.Board.Columns[store.StateReady]))
	}
}

func TestEditorFinishedError(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, _ := model.Update(editorFinishedMsg{err: errFake("boom")})
	got := updated.(Model)
	if got.Status != "Edit failed: boom" {
		t.Fatalf("Status = %q", got.Status)
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }

func TestDeleteConfirmCancel(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	if model.InteractionMode != InteractionDeleteConfirm {
		t.Fatalf("InteractionMode = %v, want delete confirm", model.InteractionMode)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = updated.(Model)
	if model.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want board", model.InteractionMode)
	}
	if model.Status != "Delete cancelled" {
		t.Fatalf("Status = %q, want cancelled", model.Status)
	}
}

func TestDeleteConfirmMovesTicketToTrash(t *testing.T) {
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

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(Model)

	if model.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want board", model.InteractionMode)
	}
	if len(model.Board.Columns[store.StateReady]) != 0 {
		t.Fatalf("ready count = %d, want 0", len(model.Board.Columns[store.StateReady]))
	}
	if _, err := os.Stat(filepath.Join(root, store.TrashDir, "a.md")); err != nil {
		t.Fatalf("trash file missing: %v", err)
	}
	if model.Status != "Deleted a.md" {
		t.Fatalf("Status = %q, want deleted", model.Status)
	}
}

func TestDeleteEmptyColumn(t *testing.T) {
	model := NewModel(emptyBoard())
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := updated.(Model)
	if got.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want board", got.InteractionMode)
	}
	if got.Status != "No ticket selected" {
		t.Fatalf("Status = %q, want no selection", got.Status)
	}
}

func TestDetailScrollIndicators(t *testing.T) {
	board := emptyBoard()
	bodyLines := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		bodyLines = append(bodyLines, fmt.Sprintf("line %d", i))
	}
	stored := storedTicket("a.md", store.StateBacklog, "Task: a")
	stored.Ticket.Body = strings.Join(bodyLines, "\n")
	board.Columns[store.StateBacklog] = []store.StoredTicket{stored}
	model := NewModel(board)
	model.Height = 12
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)

	view := model.View()
	if !strings.Contains(view, "↓") || !strings.Contains(view, "lines below") {
		t.Fatalf("detail missing below indicator:\n%s", view)
	}
	for range 10 {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		model = updated.(Model)
	}
	view = model.View()
	if !strings.Contains(view, "↑") || !strings.Contains(view, "lines above") {
		t.Fatalf("detail missing above indicator:\n%s", view)
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
	path := filepath.Join(root, string(state), name)
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

func TestPromoteToReadyMovesTicket(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	model := NewModelWithRoot(root, board)

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := got.(Model)

	if len(m.Board.Columns[store.StateReady]) != 1 {
		t.Fatalf("ready count = %d, want 1", len(m.Board.Columns[store.StateReady]))
	}
	if len(m.Board.Columns[store.StateBacklog]) != 0 {
		t.Fatalf("backlog count = %d, want 0", len(m.Board.Columns[store.StateBacklog]))
	}
	if m.SelectedCol != 1 {
		t.Fatalf("SelectedCol = %d, want 1 (ready)", m.SelectedCol)
	}
	if !strings.Contains(m.Status, "Moved a.md to ready") {
		t.Fatalf("Status = %q", m.Status)
	}
}

func TestPromoteToReadyAlreadyReadyIsNoOp(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateReady, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	model := NewModelWithRoot(root, board)
	model.SelectedCol = 1

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := got.(Model)

	if len(m.Board.Columns[store.StateReady]) != 1 {
		t.Fatalf("ready count changed unexpectedly")
	}
	if !strings.Contains(m.Status, "already in ready") {
		t.Fatalf("Status = %q, want 'already in ready'", m.Status)
	}
}

func TestEnterCreateModeOnN(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)
	if m.Mode != ViewCreate {
		t.Fatalf("Mode = %v, want ViewCreate", m.Mode)
	}
	if m.createKind != ticket.KindFeature {
		t.Fatalf("createKind = %v, want KindFeature", m.createKind)
	}
	if m.createPriority != ticket.PriorityP2 {
		t.Fatalf("createPriority = %v, want P2", m.createPriority)
	}
	if m.createField != 1 {
		t.Fatalf("createField = %d, want 1 (title)", m.createField)
	}
}

func TestCreateCancelReturnsToBoard(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)
	got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := got2.(Model)
	if m2.Mode != ViewBoard {
		t.Fatalf("Mode = %v, want ViewBoard after esc", m2.Mode)
	}
	if m2.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want InteractionBoard", m2.InteractionMode)
	}
}

func TestCreateEmptyTitleShowsError(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)
	got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := got2.(Model)
	if m2.Mode != ViewCreate {
		t.Fatalf("Mode = %v, want ViewCreate after empty submit", m2.Mode)
	}
	if !strings.Contains(m2.Status, "Title required") {
		t.Fatalf("Status = %q, want 'Title required'", m2.Status)
	}
}

func TestCreateTicketWritesFile(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	board, err := store.LoadBoard(boardRoot)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}

	model := NewModelWithRoot(boardRoot, board)
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)

	// Type "My Ticket" into the title field
	for _, ch := range "My Ticket" {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = got2.(Model)
	}

	got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := got3.(Model)

	if m3.Mode != ViewBoard {
		t.Fatalf("Mode = %v, want ViewBoard after create", m3.Mode)
	}
	if m3.InteractionMode != InteractionPostCreate {
		t.Fatalf("InteractionMode = %v, want InteractionPostCreate", m3.InteractionMode)
	}

	backlogTickets := m3.Board.Columns[store.StateBacklog]
	if len(backlogTickets) == 0 {
		t.Fatalf("backlog empty after create")
	}
	if !strings.Contains(backlogTickets[0].Ticket.Title, "My Ticket") {
		t.Fatalf("ticket title = %q, want My Ticket", backlogTickets[0].Ticket.Title)
	}
}

func TestPostCreateNReturnsToBoard(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	board, _ := store.LoadBoard(boardRoot)
	model := NewModelWithRoot(boardRoot, board)

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)
	for _, ch := range "Test" {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = got2.(Model)
	}
	got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := got3.(Model)

	got4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m4 := got4.(Model)
	if m4.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want InteractionBoard after n", m4.InteractionMode)
	}
	if !strings.Contains(m4.Status, "Created") {
		t.Fatalf("Status = %q, want 'Created ...'", m4.Status)
	}
}

func TestCreateToRefineCheckboxTogglesWithSpace(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	board, _ := store.LoadBoard(boardRoot)
	model := NewModelWithRoot(boardRoot, board)

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)

	// Navigate to toRefine field (field 3) via tab x3 from title (field 1)
	for range 2 {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = got2.(Model)
	}
	if m.createField != 3 {
		t.Fatalf("createField = %d, want 3 (toRefine)", m.createField)
	}
	if !m.createToRefine {
		t.Fatalf("createToRefine = false initially, want true")
	}

	// Toggle with space
	got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3 := got3.(Model)
	if m3.createToRefine {
		t.Fatalf("createToRefine = true after space, want false")
	}

	// Toggle back
	got4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m4 := got4.(Model)
	if !m4.createToRefine {
		t.Fatalf("createToRefine = false after second space, want true")
	}
}

func TestEnterOnToRefineFieldDoesNotToggle(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)
	initial := m.createToRefine

	// Navigate to toRefine field (field 3)
	for range 2 {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = got2.(Model)
	}
	if m.createField != 3 {
		t.Fatalf("createField = %d, want 3", m.createField)
	}

	got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := got3.(Model)
	if m3.createToRefine != initial {
		t.Fatalf("enter on field 3 toggled createToRefine: was %v, now %v", initial, m3.createToRefine)
	}
}

func TestCreateToRefineAddsLabel(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(boardRoot); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	board, _ := store.LoadBoard(boardRoot)
	model := NewModelWithRoot(boardRoot, board)

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)

	// Type title
	for _, ch := range "Needs Design" {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = got2.(Model)
	}

	if !m.createToRefine {
		t.Fatalf("createToRefine not set by default")
	}

	// Tab x3 to get past toRefine (field 3) back to kind field, then submit via title
	for range 3 {
		got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = got3.(Model)
	}

	// Tab back to title field and submit
	got5, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got5.(Model)
	got6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m5 := got6.(Model)

	if m5.Mode != ViewBoard {
		t.Fatalf("Mode = %v after create, want ViewBoard", m5.Mode)
	}

	tickets := m5.Board.Columns[store.StateBacklog]
	if len(tickets) == 0 {
		t.Fatalf("backlog empty after create")
	}
	if !tickets[0].Ticket.ParsedTitle.ToRefine() {
		t.Fatalf("ticket does not have to-refine label: title = %q", tickets[0].Ticket.Title)
	}
}

func TestCreateTitleFieldArrowKeysMovesCursor(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)

	// Type "ab" into title
	for _, ch := range "ab" {
		got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = got2.(Model)
	}
	if m.createInput.Value() != "ab" {
		t.Fatalf("value = %q, want ab", m.createInput.Value())
	}

	// left arrow should move cursor (cursor pos goes from 2 to 1)
	posBefore := m.createInput.Position()
	got3, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m3 := got3.(Model)
	if m3.createInput.Position() >= posBefore {
		t.Fatalf("cursor did not move left: pos %d -> %d", posBefore, m3.createInput.Position())
	}
	// kind must not have changed
	if m3.createKind != m.createKind {
		t.Fatalf("left arrow changed kind on title field")
	}

	// right arrow should move cursor back
	posAfterLeft := m3.createInput.Position()
	got4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRight})
	m4 := got4.(Model)
	if m4.createInput.Position() <= posAfterLeft {
		t.Fatalf("cursor did not move right: pos %d -> %d", posAfterLeft, m4.createInput.Position())
	}
}

func TestCreateKindCyclesWithHL(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got.(Model)

	// Navigate to kind field (field 0) via shift+tab from title (field 1)
	got2, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m2 := got2.(Model)
	if m2.createField != 0 {
		t.Fatalf("createField = %d, want 0 (kind)", m2.createField)
	}
	if m2.createKind != ticket.KindFeature {
		t.Fatalf("initial kind = %v, want KindFeature", m2.createKind)
	}

	got3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m3 := got3.(Model)
	if m3.createKind != ticket.KindTask {
		t.Fatalf("kind after l = %v, want KindTask", m3.createKind)
	}

	got4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m4 := got4.(Model)
	if m4.createKind != ticket.KindFeature {
		t.Fatalf("kind after h = %v, want KindFeature", m4.createKind)
	}
}

func TestRKeyReloadsBoard(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	m := NewModelWithRoot(root, board)

	// Write a second ticket directly to disk (simulating external change)
	writeTUITestTicket(t, root, store.StateBacklog, "b.md", "Task: b")

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m2 := got.(Model)
	if len(m2.Board.Columns[store.StateBacklog]) != 2 {
		t.Fatalf("backlog len = %d after r, want 2", len(m2.Board.Columns[store.StateBacklog]))
	}
}

func TestWatcherMsgReloadsBoard(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	m := NewModelWithRoot(root, board)

	writeTUITestTicket(t, root, store.StateBacklog, "b.md", "Task: b")

	got, cmd := m.Update(msgFileChanged{})
	m2 := got.(Model)
	if len(m2.Board.Columns[store.StateBacklog]) != 2 {
		t.Fatalf("backlog len = %d after watcher msg, want 2", len(m2.Board.Columns[store.StateBacklog]))
	}
	if cmd == nil {
		t.Fatal("Update(msgFileChanged) returned nil cmd — watcher not re-subscribed")
	}
}

func TestReloadPreservesFocus(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	writeTUITestTicket(t, root, store.StateBacklog, "b.md", "Task: b")
	board, err := store.LoadBoard(root)
	if err != nil {
		t.Fatalf("LoadBoard() error = %v", err)
	}
	m := NewModelWithRoot(root, board)
	m.SelectedRows[store.StateBacklog] = 1 // focus b.md

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m2 := got.(Model)
	focused := m2.Board.Columns[store.StateBacklog][m2.SelectedRows[store.StateBacklog]]
	if focused.Name != "b.md" {
		t.Fatalf("focused = %q after reload, want b.md", focused.Name)
	}
}

func newModelForSort(t *testing.T, board store.Board) Model {
	t.Helper()
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return NewModelWithRoot(root, board)
}

func TestDefaultSortIsPriority(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		storedTicket("p3.md", store.StateBacklog, "Task: low"),
		storedTicket("p0.md", store.StateBacklog, "Task: high"),
		storedTicket("p1.md", store.StateBacklog, "Task: mid"),
	}
	board.Columns[store.StateBacklog][0].Ticket.Priority = ticket.PriorityP3
	board.Columns[store.StateBacklog][1].Ticket.Priority = ticket.PriorityP0
	board.Columns[store.StateBacklog][2].Ticket.Priority = ticket.PriorityP1

	m := newModelForSort(t, board)
	tickets := m.Board.Columns[store.StateBacklog]
	if tickets[0].Name != "p0.md" || tickets[1].Name != "p1.md" || tickets[2].Name != "p3.md" {
		t.Fatalf("priority sort wrong: %v %v %v", tickets[0].Name, tickets[1].Name, tickets[2].Name)
	}
}

func TestSortByCycleTitle(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		storedTicket("c.md", store.StateBacklog, "Task: Charlie"),
		storedTicket("a.md", store.StateBacklog, "Task: Alpha"),
		storedTicket("b.md", store.StateBacklog, "Task: Bravo"),
	}
	board.Columns[store.StateBacklog][0].Ticket.Title = "Task: Charlie"
	board.Columns[store.StateBacklog][1].Ticket.Title = "Task: Alpha"
	board.Columns[store.StateBacklog][2].Ticket.Title = "Task: Bravo"

	m := newModelForSort(t, board)
	// Cycle from priority → title
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m2 := got.(Model)
	if m2.SortMode != store.SortTitle {
		t.Fatalf("SortMode = %v, want title", m2.SortMode)
	}
	tickets := m2.Board.Columns[store.StateBacklog]
	if tickets[0].Name != "a.md" {
		t.Fatalf("title sort: first = %q, want a.md", tickets[0].Name)
	}
}

func TestSortPromptYSwitchesToManual(t *testing.T) {
	model := enterMoveMode(t, newModelForSort(t, emptyBoard()))
	// Trigger sort prompt with j
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m2 := got.(Model)
	if m2.InteractionMode != InteractionSortPrompt {
		t.Fatalf("expected InteractionSortPrompt, got %v", m2.InteractionMode)
	}
	// Confirm with y
	got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m3 := got2.(Model)
	if m3.SortMode != store.SortManual {
		t.Fatalf("SortMode = %v after y, want manual", m3.SortMode)
	}
	if m3.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v after y, want InteractionMove", m3.InteractionMode)
	}
}

func TestSortPromptNCancels(t *testing.T) {
	model := enterMoveMode(t, newModelForSort(t, emptyBoard()))
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m2 := got.(Model)
	prevSort := m2.SortMode
	got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m3 := got2.(Model)
	if m3.SortMode != prevSort {
		t.Fatalf("SortMode changed to %v after n, want %v", m3.SortMode, prevSort)
	}
	if m3.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v after n, want InteractionMove", m3.InteractionMode)
	}
}

func TestManualMoveReordersTickets(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	writeTUITestTicket(t, root, store.StateBacklog, "b.md", "Task: b")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)
	// Switch to manual
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m2 := got.(Model)
	if m2.SortMode != store.SortManual {
		t.Fatalf("SortMode = %v, want manual", m2.SortMode)
	}
	// Enter move mode and press j to move first ticket down
	got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	got3, _ := got2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m3 := got3.(Model)
	// a.md should now be second
	tickets := m3.Board.Columns[store.StateBacklog]
	if tickets[0].Name != "b.md" || tickets[1].Name != "a.md" {
		t.Fatalf("after j in manual: order = [%s, %s], want [b.md, a.md]", tickets[0].Name, tickets[1].Name)
	}
}

func TestCKeyOpensConfigPage(t *testing.T) {
	m := newModelForSort(t, emptyBoard())
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m2 := got.(Model)
	if m2.Mode != ViewConfig {
		t.Fatalf("Mode = %v, want ViewConfig", m2.Mode)
	}
}

func TestConfigEscReturnsToBoard(t *testing.T) {
	m := newModelForSort(t, emptyBoard())
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := got2.(Model)
	if m3.Mode != ViewBoard {
		t.Fatalf("Mode = %v after esc, want ViewBoard", m3.Mode)
	}
}

func TestConfigSelectPresetAndSave(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Open config, cycle to nvim (index 1), save
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got3, _ := got2.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := got3.(Model)

	if m4.Mode != ViewBoard {
		t.Fatalf("Mode = %v after save, want ViewBoard", m4.Mode)
	}
	if m4.Config.Editor != "nvim" {
		t.Fatalf("Config.Editor = %q, want nvim", m4.Config.Editor)
	}

	// Verify persisted
	cfg, err := store.LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Editor != "nvim" {
		t.Fatalf("persisted editor = %q, want nvim", cfg.Editor)
	}
}

func TestConfigCustomEditor(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Open config, cycle to "custom" (past all presets)
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mc := got.(Model)
	for range len(editorPresets) {
		got2, _ := mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		mc = got2.(Model)
	}
	if mc.configEditorIdx != len(editorPresets) {
		t.Fatalf("configEditorIdx = %d, want %d (custom)", mc.configEditorIdx, len(editorPresets))
	}

	// Type a custom editor
	for _, ch := range "emacs" {
		got2, _ := mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		mc = got2.(Model)
	}

	// Save
	got3, _ := mc.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := got3.(Model)
	if m4.Config.Editor != "emacs" {
		t.Fatalf("Config.Editor = %q, want emacs", m4.Config.Editor)
	}
}
