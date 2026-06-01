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

func TestColumnScrollKeepsFocusedTicketVisibleAtBottom(t *testing.T) {
	board := emptyBoard()
	for i := 0; i < 8; i++ {
		board.Columns[store.StateBacklog] = append(board.Columns[store.StateBacklog], storedTicket(fmt.Sprintf("%d.md", i), store.StateBacklog, fmt.Sprintf("Task: visible-%d", i)))
	}
	model := NewModel(board)
	model.Height = 12

	for range 7 {
		model.moveRow(1)
	}

	view := model.View()
	if !strings.Contains(view, "Task: visible-7") {
		t.Fatalf("View() missing focused bottom ticket after scrolling:\n%s", view)
	}
}

func TestColumnScrollKeepsFocusedTicketVisibleAfterWrappedTicket(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{
		storedTicket("long.md", store.StateBacklog, "Task: this title is intentionally very long so it wraps over multiple board lines and consumes the line budget"),
		storedTicket("selected.md", store.StateBacklog, "Task: selected-after-wrapped-ticket"),
	}
	model := NewModel(board)
	model.Height = 12
	model.Width = 80

	model.moveRow(1)

	view := model.View()
	if !strings.Contains(view, "Task: selected-after-wrapped-ticket") {
		t.Fatalf("View() missing focused ticket after wrapped predecessor:\n%s", view)
	}
}

func TestDKeyScrollsHalfPageDown(t *testing.T) {
	board := emptyBoard()
	for i := range 10 {
		board.Columns[store.StateBacklog] = append(board.Columns[store.StateBacklog],
			storedTicket(fmt.Sprintf("%d.md", i), store.StateBacklog, fmt.Sprintf("Task: %d", i)))
	}
	m := NewModel(board)
	m.Height = 20

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := got.(Model)
	if m2.SelectedRows[store.StateBacklog] == 0 {
		t.Fatal("d did not advance row selection")
	}
	if m2.SelectedRows[store.StateBacklog] <= 1 {
		t.Fatalf("d moved only 1 row, expected half-page jump; row=%d", m2.SelectedRows[store.StateBacklog])
	}
}

func TestUKeyScrollsHalfPageUp(t *testing.T) {
	board := emptyBoard()
	for i := range 10 {
		board.Columns[store.StateBacklog] = append(board.Columns[store.StateBacklog],
			storedTicket(fmt.Sprintf("%d.md", i), store.StateBacklog, fmt.Sprintf("Task: %d", i)))
	}
	m := NewModel(board)
	m.Height = 20
	// Move to bottom first
	for range 9 {
		m.moveRow(1)
	}
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m2 := got.(Model)
	if m2.SelectedRows[store.StateBacklog] >= 9 {
		t.Fatal("u did not scroll up")
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
	if got.columnWidth() != 58 {
		t.Fatalf("columnWidth = %d, want 58", got.columnWidth())
	}
}

func TestBoardRendersColumnBorders(t *testing.T) {
	model := NewModel(emptyBoard())
	view := model.View()
	if !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Fatalf("View() missing borders:\n%s", view)
	}
}

func TestBoardRendersWontDoColumnAfterDone(t *testing.T) {
	model := NewModel(emptyBoard())
	model.Width = 300
	view := model.View()
	doneIdx := strings.Index(view, "DONE")
	wontDoIdx := strings.Index(view, "WON'T DO")
	if doneIdx < 0 || wontDoIdx < 0 {
		t.Fatalf("View() missing Done or Won't Do columns:\n%s", view)
	}
	if wontDoIdx < doneIdx {
		t.Fatalf("Won't Do column rendered before Done:\n%s", view)
	}
}

func TestThemesDefineDistinctWontDoColor(t *testing.T) {
	for _, theme := range colorThemes {
		t.Run(theme.name, func(t *testing.T) {
			if len(theme.colors) != len(columnOrder) {
				t.Fatalf("color count = %d, want %d", len(theme.colors), len(columnOrder))
			}
			wontDoColor := theme.colors[stateColIndex(store.StateWontDo)]
			doneColor := theme.colors[stateColIndex(store.StateDone)]
			if wontDoColor == "" {
				t.Fatal("Won't Do color is empty")
			}
			if wontDoColor == doneColor {
				t.Fatalf("Won't Do color = Done color %q, want distinct muted/rejected color", wontDoColor)
			}
		})
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
	if !strings.Contains(view, "CONTENT") {
		t.Fatalf("detail missing CONTENT header:\n%s", view)
	}
	if !strings.Contains(view, "Long body line 1") {
		t.Fatalf("detail missing body:\n%s", view)
	}
	if !strings.Contains(view, "METADATA") || !strings.Contains(view, "Title: Task: a") || !strings.Contains(view, "State:") || !strings.Contains(view, "backlog") || !strings.Contains(view, "File: a.md") {
		t.Fatalf("detail missing metadata:\n%s", view)
	}
	if !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Fatalf("detail missing borders:\n%s", view)
	}
}

func TestDetailMetadataHeaderAndColors(t *testing.T) {
	board := emptyBoard()
	stored := storedTicket("a.md", store.StateReady, "Task: a")
	stored.Ticket.Priority = ticket.PriorityP0
	board.Columns[store.StateReady] = []store.StoredTicket{stored}
	m := NewModel(board)
	m.SelectedCol = 1
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	meta := m.renderDetailMetadata(*m.selectedTicket())

	if !strings.Contains(meta, "METADATA") {
		t.Fatalf("missing METADATA header: %s", meta)
	}
	if !strings.Contains(meta, "Title: Task: a") {
		t.Fatalf("missing title line: %s", meta)
	}
	if !strings.Contains(meta, "State:") || !strings.Contains(meta, "ready") {
		t.Fatalf("missing state value: %s", meta)
	}
	if !strings.Contains(meta, "Priority:") || !strings.Contains(meta, "P0") {
		t.Fatalf("missing priority value: %s", meta)
	}
	if !strings.Contains(meta, "Deadline: —") {
		t.Fatalf("missing empty deadline line: %s", meta)
	}
}

func TestDetailMetadataShowsDeadlineWhenPresent(t *testing.T) {
	stored := storedTicket("a.md", store.StateReady, "Task: a")
	deadline := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	stored.Ticket.Deadline = &deadline
	m := NewModel(emptyBoard())

	meta := m.renderDetailMetadata(stored)
	if !strings.Contains(meta, "Deadline: 2026-06-15") {
		t.Fatalf("missing deadline line: %s", meta)
	}
}

func TestBoardShowsSLAIndicatorForTicketWithDeadline(t *testing.T) {
	board := emptyBoard()
	stored := storedTicket("a.md", store.StateReady, "Task: has deadline")
	deadline := time.Now().UTC().AddDate(0, 0, 2)
	stored.Ticket.Deadline = &deadline
	board.Columns[store.StateReady] = []store.StoredTicket{stored}
	m := NewModel(board)
	m.SelectedCol = stateColIndex(store.StateReady)

	view := m.View()
	if !strings.Contains(view, "SLA") || !strings.Contains(view, "|") {
		t.Fatalf("view missing SLA bar indicator:\n%s", view)
	}
	if strings.Count(view, "|") != 6 {
		t.Fatalf("SLA pipe count = %d, want 6 in view:\n%s", strings.Count(view, "|"), view)
	}
	if strings.Contains(view, "█") {
		t.Fatalf("view shows block SLA bars instead of pipe bars:\n%s", view)
	}
	if strings.Contains(view, deadline.Format(time.DateOnly)) {
		t.Fatalf("board view shows deadline date instead of visual indicator:\n%s", view)
	}
}

func TestDeadlineBarStatesUseReadyDoingDoneColors(t *testing.T) {
	want := []store.State{
		store.StateReady,
		store.StateReady,
		store.StateReady,
		store.StateDoing,
		store.StateDoing,
		store.StateDone,
	}
	got := deadlineBarStates()
	if len(got) != len(want) {
		t.Fatalf("deadlineBarStates len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("deadlineBarStates()[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestDeadlineBarCountIncreasesAsDeadlineApproaches(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		deadline time.Time
		want     int
	}{
		{name: "far", deadline: now.AddDate(0, 0, 30), want: 1},
		{name: "two weeks", deadline: now.AddDate(0, 0, 14), want: 2},
		{name: "week", deadline: now.AddDate(0, 0, 7), want: 3},
		{name: "three days", deadline: now.AddDate(0, 0, 3), want: 4},
		{name: "tomorrow", deadline: now.AddDate(0, 0, 1), want: 5},
		{name: "due today", deadline: now, want: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deadlineBarCount(tt.deadline, now); got != tt.want {
				t.Fatalf("deadlineBarCount() = %d, want %d", got, tt.want)
			}
		})
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
	if !strings.Contains(view, "────") || !strings.Contains(view, "BOARD:") {
		t.Fatalf("View() missing footer separator:\n%s", view)
	}
}

func TestStatusAndWarningsRenderInFooterSeparatorWithoutAddingLines(t *testing.T) {
	model := NewModel(emptyBoard())
	model.Board.Warnings = []store.Warning{{Path: "bad.md", Err: fmt.Errorf("bad")}}
	before := model.View()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	after := updated.(Model).View()

	if strings.Count(after, "\n") != strings.Count(before, "\n") {
		t.Fatalf("line count changed after status/warning snack; before=%d after=%d\nbefore:\n%s\nafter:\n%s", strings.Count(before, "\n"), strings.Count(after, "\n"), before, after)
	}
	if !strings.Contains(after, "Sort:") || !strings.Contains(after, "Warnings:") {
		t.Fatalf("view missing status/warning snack:\n%s", after)
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

func TestQOpensQuitConfirm(t *testing.T) {
	model := NewModel(emptyBoard())
	got, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := got.(Model)
	if cmd != nil {
		t.Fatal("q should not quit immediately")
	}
	if m.InteractionMode != InteractionQuitConfirm {
		t.Fatalf("InteractionMode = %v, want InteractionQuitConfirm", m.InteractionMode)
	}
}

func TestQuitConfirmYQuits(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_, cmd := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y in quit confirm should issue quit cmd")
	}
}

func TestQuitConfirmQQuits(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_, cmd := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q in quit confirm should issue quit cmd")
	}
}

func TestQuitConfirmNCancels(t *testing.T) {
	model := NewModel(emptyBoard())
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got2, cmd := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := got2.(Model)
	if cmd != nil {
		t.Fatal("n in quit confirm should not quit")
	}
	if m.InteractionMode == InteractionQuitConfirm {
		t.Fatal("still in quit confirm after n")
	}
}

func TestCtrlCQuitsImmediately(t *testing.T) {
	model := NewModel(emptyBoard())
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should quit immediately")
	}
}

func TestQFromDetailOpensQuitConfirm(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	m := NewModel(board)
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // enter detail
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := got2.(Model)
	if m2.InteractionMode != InteractionQuitConfirm {
		t.Fatalf("InteractionMode = %v, want InteractionQuitConfirm", m2.InteractionMode)
	}
	if m2.prevMode != ViewDetail {
		t.Fatalf("prevMode = %v, want ViewDetail", m2.prevMode)
	}
}

func TestQFromDetailConfirmNRestoresDetail(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	m := NewModel(board)
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got3, _ := got2.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := got3.(Model)
	if m3.Mode != ViewDetail {
		t.Fatalf("Mode = %v after esc, want ViewDetail", m3.Mode)
	}
}

func TestQFromMoveOpensQuitConfirm(t *testing.T) {
	m := enterMoveMode(t, newModelForSort(t, emptyBoard()))
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := got.(Model)
	if m2.InteractionMode != InteractionQuitConfirm {
		t.Fatalf("InteractionMode = %v, want InteractionQuitConfirm", m2.InteractionMode)
	}
	if m2.prevInteractionMode != InteractionMove {
		t.Fatalf("prevInteractionMode = %v, want InteractionMove", m2.prevInteractionMode)
	}
}

func TestQFromMoveConfirmNRestoresMove(t *testing.T) {
	m := enterMoveMode(t, newModelForSort(t, emptyBoard()))
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m2 := got2.(Model)
	if m2.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v after n, want InteractionMove", m2.InteractionMode)
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
	if !strings.Contains(got.notifText(), "Moved a.md to ready") {
		t.Fatalf("notification = %q, want 'Moved a.md to ready'", got.notifText())
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
	if !strings.Contains(got.notifText(), "Moved a.md to backlog") {
		t.Fatalf("notification = %q, want 'Moved a.md to backlog'", got.notifText())
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

func TestMoveSelectedRightLastColumnNoop(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateWontDo] = []store.StoredTicket{storedTicket("a.md", store.StateWontDo, "Task: a")}
	model := NewModel(board)
	model.SelectedCol = 4
	model = enterMoveMode(t, model)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(Model)
	if got.Status != "Ticket already in wont-do" {
		t.Fatalf("Status = %q, want already in wont-do", got.Status)
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
	if got.notifText() != "Edited ticket" {
		t.Fatalf("notification = %q, want 'Edited ticket'", got.notifText())
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
	if model.notifText() != "Deleted a.md" {
		t.Fatalf("notification = %q, want 'Deleted a.md'", model.notifText())
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

func (m Model) notifText() string {
	if m.notification == nil {
		return ""
	}
	return m.notification.text
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

func TestPProgressesTicketToNextColumn(t *testing.T) {
	tests := []struct {
		name    string
		from    store.State
		fromCol int
		to      store.State
		toCol   int
	}{
		{name: "backlog to ready", from: store.StateBacklog, fromCol: 0, to: store.StateReady, toCol: 1},
		{name: "ready to doing", from: store.StateReady, fromCol: 1, to: store.StateDoing, toCol: 2},
		{name: "doing to done", from: store.StateDoing, fromCol: 2, to: store.StateDone, toCol: 3},
		{name: "done to wont-do", from: store.StateDone, fromCol: 3, to: store.StateWontDo, toCol: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := store.Init(root); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			writeTUITestTicket(t, root, tt.from, "a.md", "Task: a")
			board, err := store.LoadBoard(root)
			if err != nil {
				t.Fatalf("LoadBoard() error = %v", err)
			}
			model := NewModelWithRoot(root, board)
			model.SelectedCol = tt.fromCol

			got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			m := got.(Model)

			if len(m.Board.Columns[tt.to]) != 1 {
				t.Fatalf("%s count = %d, want 1", tt.to, len(m.Board.Columns[tt.to]))
			}
			if len(m.Board.Columns[tt.from]) != 0 {
				t.Fatalf("%s count = %d, want 0", tt.from, len(m.Board.Columns[tt.from]))
			}
			if m.SelectedCol != tt.toCol {
				t.Fatalf("SelectedCol = %d, want %d", m.SelectedCol, tt.toCol)
			}
			if _, err := os.Stat(filepath.Join(root, string(tt.to), "a.md")); err != nil {
				t.Fatalf("moved ticket missing: %v", err)
			}
			if !strings.Contains(m.notifText(), fmt.Sprintf("Moved a.md to %s", tt.to)) {
				t.Fatalf("notification = %q, want 'Moved a.md to %s'", m.notifText(), tt.to)
			}
		})
	}
}

func TestPOnWontDoIsNoOp(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateWontDo] = []store.StoredTicket{storedTicket("a.md", store.StateWontDo, "Task: a")}
	model := NewModel(board)
	model.SelectedCol = 4

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := got.(Model)

	if len(m.Board.Columns[store.StateWontDo]) != 1 {
		t.Fatalf("wont-do count changed unexpectedly")
	}
	if m.Status != "Ticket already in wont-do" {
		t.Fatalf("Status = %q, want already in wont-do", m.Status)
	}
}

func TestBBacktracksTicketToPreviousColumn(t *testing.T) {
	tests := []struct {
		name    string
		from    store.State
		fromCol int
		to      store.State
		toCol   int
	}{
		{name: "wont-do to done", from: store.StateWontDo, fromCol: 4, to: store.StateDone, toCol: 3},
		{name: "done to doing", from: store.StateDone, fromCol: 3, to: store.StateDoing, toCol: 2},
		{name: "doing to ready", from: store.StateDoing, fromCol: 2, to: store.StateReady, toCol: 1},
		{name: "ready to backlog", from: store.StateReady, fromCol: 1, to: store.StateBacklog, toCol: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := store.Init(root); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			writeTUITestTicket(t, root, tt.from, "a.md", "Task: a")
			board, err := store.LoadBoard(root)
			if err != nil {
				t.Fatalf("LoadBoard() error = %v", err)
			}
			model := NewModelWithRoot(root, board)
			model.SelectedCol = tt.fromCol

			got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
			m := got.(Model)

			if len(m.Board.Columns[tt.to]) != 1 {
				t.Fatalf("%s count = %d, want 1", tt.to, len(m.Board.Columns[tt.to]))
			}
			if len(m.Board.Columns[tt.from]) != 0 {
				t.Fatalf("%s count = %d, want 0", tt.from, len(m.Board.Columns[tt.from]))
			}
			if m.SelectedCol != tt.toCol {
				t.Fatalf("SelectedCol = %d, want %d", m.SelectedCol, tt.toCol)
			}
			if _, err := os.Stat(filepath.Join(root, string(tt.to), "a.md")); err != nil {
				t.Fatalf("moved ticket missing: %v", err)
			}
			if !strings.Contains(m.notifText(), fmt.Sprintf("Moved a.md to %s", tt.to)) {
				t.Fatalf("notification = %q, want 'Moved a.md to %s'", m.notifText(), tt.to)
			}
		})
	}
}

func TestBOnBacklogIsNoOp(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	model := NewModel(board)

	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m := got.(Model)

	if len(m.Board.Columns[store.StateBacklog]) != 1 {
		t.Fatalf("backlog count changed unexpectedly")
	}
	if m.Status != "Ticket already in backlog" {
		t.Fatalf("Status = %q, want already in backlog", m.Status)
	}
}

func TestProgressBackNoSelectionShowsMessage(t *testing.T) {
	for _, key := range []rune{'p', 'b'} {
		t.Run(string(key), func(t *testing.T) {
			model := NewModel(emptyBoard())
			got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
			m := got.(Model)
			if m.Status != "No ticket selected" {
				t.Fatalf("Status = %q, want no selection", m.Status)
			}
		})
	}
}

func TestFooterShowsCriticalShortcutsOnly(t *testing.T) {
	model := NewModel(emptyBoard())
	footer := model.footerText()
	for _, want := range []string{"h/l columns", "j/k tickets", "enter detail", "m move", "n new", "? help", "q quit"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer = %q, want %q", footer, want)
		}
	}
	for _, redundant := range []string{"p progress", "b back", "s sort", "x del", "r reload", "c config"} {
		if strings.Contains(footer, redundant) {
			t.Fatalf("footer = %q, should omit non-critical shortcut %q", footer, redundant)
		}
	}
}

func TestQuestionMarkOpensKeyboardShortcutDialog(t *testing.T) {
	model := NewModel(emptyBoard())
	model.Height = 60
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := got.(Model)
	if m.InteractionMode != InteractionHelp {
		t.Fatalf("InteractionMode = %v, want InteractionHelp", m.InteractionMode)
	}
	view := m.View()
	for _, want := range []string{"Keyboard Shortcuts", "Board", "Move mode", "Detail", "s", "cycle sort mode", "p / b", "progress / move back"} {
		if !strings.Contains(view, want) {
			t.Fatalf("help view missing %q:\n%s", want, view)
		}
	}
}

func TestHelpDialogUsesEightyPercentHeight(t *testing.T) {
	m := NewModel(emptyBoard())
	m.Height = 20
	if got := m.helpDialogHeight(); got != 16 {
		t.Fatalf("helpDialogHeight = %d, want 16", got)
	}
	if got := m.helpBoxHeight(); got != 13 {
		t.Fatalf("helpBoxHeight = %d, want 13", got)
	}
}

func TestHelpDialogScrolls(t *testing.T) {
	model := NewModel(emptyBoard())
	model.Height = 20
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := got.(Model)

	view := m.View()
	if !strings.Contains(view, "↓") || !strings.Contains(view, "lines below") {
		t.Fatalf("help view missing below indicator:\n%s", view)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = got.(Model)
	if m.HelpScroll != 1 {
		t.Fatalf("HelpScroll = %d, want 1", m.HelpScroll)
	}
	view = m.View()
	if !strings.Contains(view, "↑") || !strings.Contains(view, "lines above") {
		t.Fatalf("help view missing above indicator after scroll:\n%s", view)
	}
}

func TestHelpDialogClosesAndRestoresPreviousInteraction(t *testing.T) {
	model := NewModel(emptyBoard())
	model.InteractionMode = InteractionMove
	got, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = got.(Model)
	if m.InteractionMode != InteractionMove {
		t.Fatalf("InteractionMode = %v, want InteractionMove", m.InteractionMode)
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
	if !strings.Contains(m4.notifText(), "Created") {
		t.Fatalf("notification = %q, want 'Created ...'", m4.notifText())
	}
}

func enterPostCreateDialog(t *testing.T, root string) Model {
	t.Helper()
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m2 := got.(Model)
	for _, ch := range "Test ticket" {
		got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m2 = got2.(Model)
	}
	got3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return got3.(Model)
}

func TestPostCreateDialogQDoesNotQuit(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := enterPostCreateDialog(t, root)
	if m.InteractionMode != InteractionPostCreate {
		t.Fatalf("InteractionMode = %v, want InteractionPostCreate", m.InteractionMode)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q during post-create dialog returned a cmd (should not quit)")
	}
}

func TestPostCreateDialogDontAskAgain(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := enterPostCreateDialog(t, root)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := got.(Model)
	if m2.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v after d, want InteractionBoard", m2.InteractionMode)
	}
	if !m2.Config.SkipEditorPrompt {
		t.Fatal("SkipEditorPrompt not set after d")
	}
	cfg, _ := store.LoadConfig(root)
	if !cfg.SkipEditorPrompt {
		t.Fatal("SkipEditorPrompt not persisted to config")
	}
}

func TestPostCreateSkipsDialogWhenPrefSet(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := store.SaveConfig(root, store.Config{SkipEditorPrompt: true}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m2 := got.(Model)
	for _, ch := range "Skip test" {
		got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m2 = got2.(Model)
	}
	got3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := got3.(Model)
	if m3.InteractionMode != InteractionBoard {
		t.Fatalf("InteractionMode = %v, want InteractionBoard (dialog should be skipped)", m3.InteractionMode)
	}
}

func TestPostCreateDialogRendered(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".tickcats")
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := enterPostCreateDialog(t, root)
	view := m.View()
	if !strings.Contains(view, "Ticket Created") {
		t.Fatalf("dialog missing title:\n%s", view)
	}
	if !strings.Contains(view, "Open in external editor") {
		t.Fatalf("dialog missing prompt text:\n%s", view)
	}
	if !strings.Contains(view, "don't ask again") {
		t.Fatalf("dialog missing don't-ask-again option:\n%s", view)
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

func TestRKeyShowsReloadNotification(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m2 := got.(Model)
	if m2.notifText() != "Board reloaded" {
		t.Fatalf("notification = %q, want 'Board reloaded'", m2.notifText())
	}
	if m2.notification.kind != notifSuccess {
		t.Fatalf("notification kind = %v, want notifSuccess", m2.notification.kind)
	}
	if cmd == nil {
		t.Fatal("r key returned nil cmd, want tick for auto-clear")
	}
}

func TestDeleteNotificationShownAndAutoClearsOnTick(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	writeTUITestTicket(t, root, store.StateReady, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)
	m.SelectedCol = 1

	// confirm delete
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got2, cmd := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m2 := got2.(Model)

	if m2.notifText() != "Deleted a.md" {
		t.Fatalf("notification = %q, want 'Deleted a.md'", m2.notifText())
	}
	if m2.notification.kind != notifSuccess {
		t.Fatalf("notification kind = %v, want notifSuccess", m2.notification.kind)
	}
	if cmd == nil {
		t.Fatal("delete returned nil cmd, want tick for auto-clear")
	}

	// simulate tick firing the clear message
	gen := m2.notification.gen
	got3, _ := m2.Update(clearNotificationMsg{gen: gen})
	m3 := got3.(Model)
	if m3.notification != nil {
		t.Fatalf("notification not cleared after clearNotificationMsg: %q", m3.notifText())
	}
}

func TestStaleTickDoesNotClearNewerNotification(t *testing.T) {
	m := NewModel(emptyBoard())
	m.notify("first", notifSuccess)
	gen1 := m.notification.gen
	m.notify("second", notifSuccess)

	// stale tick from the first notify fires
	got, _ := m.Update(clearNotificationMsg{gen: gen1})
	m2 := got.(Model)
	if m2.notification == nil || m2.notifText() != "second" {
		t.Fatalf("stale tick cleared newer notification: notification = %q", m2.notifText())
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

func TestConfigThemeCyclesWithHL(t *testing.T) {
	m := newModelForSort(t, emptyBoard())

	// Open config, tab to theme field
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyTab})
	mc := got2.(Model)
	if mc.configField != 1 {
		t.Fatalf("configField = %d after tab, want 1", mc.configField)
	}
	if mc.Config.Theme != 0 {
		t.Fatalf("initial theme = %d, want 0", mc.Config.Theme)
	}

	// Cycle right
	got3, _ := mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m3 := got3.(Model)
	if m3.Config.Theme != 1 {
		t.Fatalf("theme after l = %d, want 1", m3.Config.Theme)
	}

	// Cycle left
	got4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m4 := got4.(Model)
	if m4.Config.Theme != 0 {
		t.Fatalf("theme after h = %d, want 0", m4.Config.Theme)
	}
}

func TestVTogglesSelection(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	m := NewModel(board)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m2 := got.(Model)
	if m2.totalSelected() != 1 {
		t.Fatalf("totalSelected = %d after v, want 1", m2.totalSelected())
	}
	if !m2.MultiSelected[store.StateBacklog]["a.md"] {
		t.Fatal("a.md not in MultiSelected after v")
	}
	if !strings.Contains(m2.Status, "1 selected") {
		t.Fatalf("Status = %q, want '1 selected'", m2.Status)
	}
}

func TestVDeselects(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	m := NewModel(board)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	got2, _ := got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m2 := got2.(Model)
	if m2.totalSelected() != 0 {
		t.Fatalf("totalSelected = %d after second v, want 0", m2.totalSelected())
	}
}

func TestMultiSelectMovesAllWithL(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	writeTUITestTicket(t, root, store.StateBacklog, "b.md", "Task: b")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Select both backlog tickets
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = got.(Model)
	if m.totalSelected() != 2 {
		t.Fatalf("totalSelected = %d, want 2", m.totalSelected())
	}

	// Enter move mode and press l
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = got.(Model)

	if len(m.Board.Columns[store.StateBacklog]) != 0 {
		t.Fatalf("backlog len = %d, want 0", len(m.Board.Columns[store.StateBacklog]))
	}
	if len(m.Board.Columns[store.StateReady]) != 2 {
		t.Fatalf("ready len = %d, want 2", len(m.Board.Columns[store.StateReady]))
	}
	if m.totalSelected() != 2 {
		t.Fatalf("totalSelected after move = %d, want 2", m.totalSelected())
	}
	if !strings.Contains(m.notifText(), "Moved 2 ticket(s)") {
		t.Fatalf("notification = %q, want 'Moved 2 ticket(s)'", m.notifText())
	}
}

func TestCapitalLMovesSelectedToLastColumn(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Select the ticket then enter move mode and press L
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = got.(Model)

	if len(m.Board.Columns[store.StateWontDo]) != 1 {
		t.Fatalf("wont-do len = %d, want 1", len(m.Board.Columns[store.StateWontDo]))
	}
	if m.SelectedCol != 4 {
		t.Fatalf("SelectedCol = %d, want 4", m.SelectedCol)
	}
	if !strings.Contains(m.notifText(), "wont-do") {
		t.Fatalf("notification = %q, want mention of wont-do", m.notifText())
	}
}

func TestCapitalHMovesSelectedToFirstColumn(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateDone, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)
	m.SelectedCol = 3

	// Select the ticket then enter move mode and press H
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m = got.(Model)

	if len(m.Board.Columns[store.StateBacklog]) != 1 {
		t.Fatalf("backlog len = %d, want 1", len(m.Board.Columns[store.StateBacklog]))
	}
	if m.SelectedCol != 0 {
		t.Fatalf("SelectedCol = %d, want 0", m.SelectedCol)
	}
}

func TestMultiSelectVisualIndicator(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	m := NewModel(board)

	// Before selection: no * indicator
	view := m.View()
	if strings.Contains(view, "* [") {
		t.Fatal("view shows * before selection")
	}

	// After selection: * indicator present
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m2 := got.(Model)
	view = m2.View()
	if !strings.Contains(view, "*") {
		t.Fatalf("view missing * indicator after selection:\n%s", view)
	}
}

func TestCapitalLNoSelectionMovesToLastColumn(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	writeTUITestTicket(t, root, store.StateBacklog, "a.md", "Task: a")
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// No selection — L should move focused ticket to the last column.
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = got.(Model)
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = got.(Model)

	if len(m.Board.Columns[store.StateWontDo]) != 1 {
		t.Fatalf("wont-do len = %d, want 1", len(m.Board.Columns[store.StateWontDo]))
	}
	if m.SelectedCol != 4 {
		t.Fatalf("SelectedCol = %d, want 4", m.SelectedCol)
	}
}

func TestVisibleColumnCountNarrowWide(t *testing.T) {
	m := NewModel(emptyBoard())

	m.Width = 0
	if m.visibleColumnCount() != len(columnOrder) {
		t.Fatalf("width=0: visibleColumnCount = %d, want %d", m.visibleColumnCount(), len(columnOrder))
	}

	m.Width = 300
	if m.visibleColumnCount() != 5 {
		t.Fatalf("width=300: visibleColumnCount = %d, want 5", m.visibleColumnCount())
	}

	m.Width = 120
	if m.visibleColumnCount() != 2 {
		t.Fatalf("width=120: visibleColumnCount = %d, want 2", m.visibleColumnCount())
	}

	m.Width = 60
	if m.visibleColumnCount() != 1 {
		t.Fatalf("width=60: visibleColumnCount = %d, want 1", m.visibleColumnCount())
	}
}

func TestHorizontalScrollOnNarrowTerminal(t *testing.T) {
	m := NewModel(emptyBoard())
	m.Width = 60 // fits 2 columns

	// Navigate right past visible range
	for range 3 {
		m.moveColumn(1)
	}
	if m.SelectedCol != 3 {
		t.Fatalf("SelectedCol = %d, want 3", m.SelectedCol)
	}
	if m.ColScrollOffset == 0 {
		t.Fatal("ColScrollOffset = 0 after navigating past visible range, want > 0")
	}

	// Navigate back to col 0 — scroll should follow
	for range 3 {
		m.moveColumn(-1)
	}
	if m.SelectedCol != 0 {
		t.Fatalf("SelectedCol = %d, want 0", m.SelectedCol)
	}
	if m.ColScrollOffset != 0 {
		t.Fatalf("ColScrollOffset = %d after returning to col 0, want 0", m.ColScrollOffset)
	}
}

func TestHScrollIndicatorShownOnNarrowTerminal(t *testing.T) {
	m := NewModel(emptyBoard())
	m.Width = 56 // fits 2 columns

	// Navigate to col 3 so left cols are hidden
	for range 3 {
		m.moveColumn(1)
	}
	view := m.View()
	if !strings.Contains(view, "←") {
		t.Fatalf("view missing ← scroll indicator:\n%s", view)
	}
	if !strings.Contains(view, "Backlog") {
		t.Fatalf("view missing 'Backlog' in scroll indicator:\n%s", view)
	}
}

func TestHScrollIndicatorNotShownWhenAllFit(t *testing.T) {
	m := NewModel(emptyBoard())
	m.Width = 300

	if m.renderHScrollIndicator() != "" {
		t.Fatal("hscroll indicator shown when all columns fit")
	}
}

func TestNarrowTerminalOmitsHiddenColumns(t *testing.T) {
	board := emptyBoard()
	board.Columns[store.StateBacklog] = []store.StoredTicket{storedTicket("a.md", store.StateBacklog, "Task: a")}
	board.Columns[store.StateDone] = []store.StoredTicket{storedTicket("z.md", store.StateDone, "Task: z")}
	m := NewModel(board)
	m.Width = 56 // fits 2 columns; at offset 0 shows backlog+ready

	view := m.View()
	// backlog should be visible (col 0)
	if !strings.Contains(view, "Task: a") {
		t.Fatalf("view missing backlog ticket:\n%s", view)
	}
	// done ticket should be hidden
	if strings.Contains(view, "Task: z") {
		t.Fatalf("view shows done ticket when it should be scrolled off:\n%s", view)
	}

	// Navigate to done column
	for range 3 {
		m.moveColumn(1)
	}
	view = m.View()
	if !strings.Contains(view, "Task: z") {
		t.Fatalf("view missing done ticket after scroll:\n%s", view)
	}
	if strings.Contains(view, "Task: a") {
		t.Fatalf("view shows backlog ticket after scrolling right:\n%s", view)
	}
}

func TestConfigThemePersistedOnSave(t *testing.T) {
	root := t.TempDir()
	if err := store.Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	board, _ := store.LoadBoard(root)
	m := NewModelWithRoot(root, board)

	// Open config, tab to theme, cycle to theme 2, save
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyTab})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := got.(Model)

	if m2.Config.Theme != 2 {
		t.Fatalf("Config.Theme = %d after save, want 2", m2.Config.Theme)
	}
	cfg, _ := store.LoadConfig(root)
	if cfg.Theme != 2 {
		t.Fatalf("persisted theme = %d, want 2", cfg.Theme)
	}
}
