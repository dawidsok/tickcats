// actions.go implements user-triggered operations that modify board state:
// deleting a ticket, editing in an external editor, moving selected tickets
// in bulk, reordering within a column, and cycling the sort mode.
//
// The reload pattern used throughout this file: after any store mutation the
// board must be reloaded from disk, the manual order synced to the new state,
// and the sort applied. loadAndResortBoard encapsulates this sequence;
// reloadBoard extends it with cursor preservation and multi-select cleanup.
package tui

import (
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m *Model) enterDeleteConfirm() {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return
	}
	m.InteractionMode = InteractionDeleteConfirm
	m.Status = fmt.Sprintf("Delete %s?", stored.Name)
}

func (m *Model) deleteSelected() tea.Cmd {
	stored := m.selectedTicket()
	if stored == nil {
		m.InteractionMode = InteractionBoard
		m.Status = "No ticket selected"
		return nil
	}

	name := stored.Name
	if _, err := store.Trash(m.Root, name, stored.State); err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Delete failed: " + err.Error()
		return nil
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.InteractionMode = InteractionBoard
		m.Status = "Reload failed: " + err.Error()
		return nil
	}

	m.Board = board
	m.InteractionMode = InteractionBoard
	m.Status = ""
	return m.notify("Deleted "+name, notifSuccess)
}

func (m Model) editSelected() (tea.Model, tea.Cmd) {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return m, nil
	}

	cmd := editorCommand(stored.Path, m.Config.Editor)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

type editorFinishedMsg struct {
	err error
}

func (m *Model) handleEditorFinished(msg editorFinishedMsg) tea.Cmd {
	if msg.err != nil {
		m.Status = "Edit failed: " + msg.err.Error()
		return nil
	}
	m.reloadBoard()
	return m.notify("Edited ticket", notifSuccess)
}

// loadAndResortBoard reloads the board from disk and re-applies sort.
// It is the base reload step; reloadBoard extends this with cursor restoration.
func (m *Model) loadAndResortBoard() bool {
	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return false
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	return true
}

// reloadBoard reloads the board and attempts to restore the cursor to the same
// ticket it was on before the reload. Used for file-watch events and the manual
// reload key so the cursor does not jump unexpectedly when tickets are added or
// removed externally.
func (m *Model) reloadBoard() bool {
	state := columnOrder[m.SelectedCol]
	focusedName := ""
	if tickets := m.Board.Columns[state]; m.SelectedRows[state] < len(tickets) {
		focusedName = tickets[m.SelectedRows[state]].Name
	}

	if !m.loadAndResortBoard() {
		return false
	}
	m.syncMultiSelected()

	// In detail view, the ticket may have moved to a different column.
	// Search all columns to restore the cursor to its new location.
	if m.detailTicketName != "" {
		m.resolveDetailCursor()
		return true
	}

	if focusedName == "" {
		return true
	}
	newTickets := m.Board.Columns[state]
	for i, t := range newTickets {
		if t.Name == focusedName {
			m.SelectedRows[state] = i
			m.ensureSelectedVisible(state)
			return true
		}
	}
	if m.SelectedRows[state] >= len(newTickets) && len(newTickets) > 0 {
		m.SelectedRows[state] = len(newTickets) - 1
	}
	m.ensureSelectedVisible(state)
	return true
}

func (m *Model) cycleSortMode() {
	for i, mode := range store.SortModes {
		if mode == m.SortMode {
			m.SortMode = store.SortModes[(i+1)%len(store.SortModes)]
			m.syncManualOrder()
			m.applySortToBoard()
			m.saveSortConfig()
			m.Status = "Sort: " + string(m.SortMode)
			return
		}
	}
	m.SortMode = store.SortPriority
}

func (m *Model) saveSortConfig() {
	_ = store.SaveSortConfig(m.Root, store.SortConfig{
		Mode:        m.SortMode,
		ManualOrder: m.ManualOrder,
	})
}

// syncManualOrder reconciles the persisted ManualOrder map with the current
// board state: new tickets are appended to the end of the order list, and
// tickets that no longer exist in the board are removed.
func (m *Model) syncManualOrder() {
	if m.ManualOrder == nil {
		m.ManualOrder = make(map[store.State][]string)
	}
	for state, tickets := range m.Board.Columns {
		existing := m.ManualOrder[state]
		existingSet := make(map[string]bool, len(existing))
		for _, name := range existing {
			existingSet[name] = true
		}
		// Append new tickets not yet in manual order
		for _, t := range tickets {
			if !existingSet[t.Name] {
				m.ManualOrder[state] = append(m.ManualOrder[state], t.Name)
			}
		}
		// Remove tickets that no longer exist
		ticketSet := make(map[string]bool, len(tickets))
		for _, t := range tickets {
			ticketSet[t.Name] = true
		}
		filtered := m.ManualOrder[state][:0]
		for _, name := range m.ManualOrder[state] {
			if ticketSet[name] {
				filtered = append(filtered, name)
			}
		}
		m.ManualOrder[state] = filtered
	}
}

func (m *Model) applySortToBoard() {
	for _, state := range columnOrder {
		tickets := m.Board.Columns[state]
		if len(tickets) <= 1 {
			continue
		}
		sorted := make([]store.StoredTicket, len(tickets))
		copy(sorted, tickets)
		switch m.SortMode {
		case store.SortPriority:
			sort.SliceStable(sorted, func(i, j int) bool {
				ri, rj := sorted[i].Ticket.Priority.Rank(), sorted[j].Ticket.Priority.Rank()
				if ri != rj {
					return ri < rj
				}
				return sorted[i].Name < sorted[j].Name
			})
		case store.SortTitle:
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Ticket.Title < sorted[j].Ticket.Title
			})
		case store.SortDate:
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Ticket.Created.Before(sorted[j].Ticket.Created)
			})
		case store.SortManual:
			order := m.ManualOrder[state]
			idx := make(map[string]int, len(order))
			for i, name := range order {
				idx[name] = i
			}
			sort.SliceStable(sorted, func(i, j int) bool {
				ii, iok := idx[sorted[i].Name]
				ji, jok := idx[sorted[j].Name]
				if iok && jok {
					return ii < ji
				}
				if iok {
					return true
				}
				if jok {
					return false
				}
				return sorted[i].Name < sorted[j].Name
			})
		}
		m.Board.Columns[state] = sorted
	}
}

func (m *Model) moveSelectedInColumn(delta int) {
	state := columnOrder[m.SelectedCol]
	stored := m.selectedTicket()
	if stored == nil {
		return
	}
	order := m.ManualOrder[state]
	for i, name := range order {
		if name == stored.Name {
			newI := i + delta
			if newI < 0 || newI >= len(order) {
				return
			}
			order[i], order[newI] = order[newI], order[i]
			m.ManualOrder[state] = order
			m.applySortToBoard()
			m.SelectedRows[state] = findTicketRow(m.Board.Columns[state], stored.Name)
			m.ensureSelectedVisible(state)
			m.saveSortConfig()
			return
		}
	}
}

func (m Model) totalSelected() int {
	n := 0
	for _, s := range m.MultiSelected {
		n += len(s)
	}
	return n
}

func (m *Model) toggleSelection() {
	stored := m.selectedTicket()
	if stored == nil {
		return
	}
	state := columnOrder[m.SelectedCol]
	if m.MultiSelected[state] == nil {
		m.MultiSelected[state] = make(map[string]bool)
	}
	if m.MultiSelected[state][stored.Name] {
		delete(m.MultiSelected[state], stored.Name)
		if len(m.MultiSelected[state]) == 0 {
			delete(m.MultiSelected, state)
		}
	} else {
		m.MultiSelected[state][stored.Name] = true
	}
}

func (m *Model) syncMultiSelected() {
	for state, names := range m.MultiSelected {
		ticketSet := make(map[string]bool, len(m.Board.Columns[state]))
		for _, t := range m.Board.Columns[state] {
			ticketSet[t.Name] = true
		}
		for name := range names {
			if !ticketSet[name] {
				delete(names, name)
			}
		}
		if len(names) == 0 {
			delete(m.MultiSelected, state)
		}
	}
}

type selectedRef struct {
	name   string
	state  store.State
	colIdx int
}

func (m *Model) allSelectedRefs() []selectedRef {
	var refs []selectedRef
	for colIdx, state := range columnOrder {
		for name := range m.MultiSelected[state] {
			refs = append(refs, selectedRef{name, state, colIdx})
		}
	}
	return refs
}

func (m *Model) moveAllSelectedBy(delta int) tea.Cmd {
	if m.totalSelected() == 0 {
		return m.moveSelected(delta)
	}

	refs := m.allSelectedRefs()
	if delta > 0 {
		sort.Slice(refs, func(i, j int) bool { return refs[i].colIdx > refs[j].colIdx })
	} else {
		sort.Slice(refs, func(i, j int) bool { return refs[i].colIdx < refs[j].colIdx })
	}

	moved := 0
	for _, r := range refs {
		toIdx := r.colIdx + delta
		if toIdx < 0 || toIdx >= len(columnOrder) {
			continue
		}
		if _, err := store.Move(m.Root, r.name, r.state, columnOrder[toIdx]); err != nil {
			m.Status = "Move failed: " + err.Error()
			return nil
		}
		moved++
	}

	if !m.loadAndResortBoard() {
		return nil
	}

	newSelected := make(map[store.State]map[string]bool)
	for _, r := range refs {
		newIdx := r.colIdx + delta
		if newIdx < 0 || newIdx >= len(columnOrder) {
			newIdx = r.colIdx
		}
		newState := columnOrder[newIdx]
		if newSelected[newState] == nil {
			newSelected[newState] = make(map[string]bool)
		}
		newSelected[newState][r.name] = true
	}
	m.MultiSelected = newSelected
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(columnOrder)-1)
	m.ensureColVisible()

	if moved == 0 {
		if delta > 0 {
			m.Status = "Ticket(s) already at last column"
		} else {
			m.Status = "Ticket(s) already at first column"
		}
		return nil
	}
	return m.notify(fmt.Sprintf("Moved %d ticket(s)", moved), notifSuccess)
}

func (m *Model) moveAllSelectedTo(targetCol int) tea.Cmd {
	if m.totalSelected() == 0 {
		return m.moveSelected(targetCol - m.SelectedCol)
	}

	refs := m.allSelectedRefs()
	moved := 0
	for _, r := range refs {
		if r.colIdx == targetCol {
			continue
		}
		if _, err := store.Move(m.Root, r.name, r.state, columnOrder[targetCol]); err != nil {
			m.Status = "Move failed: " + err.Error()
			return nil
		}
		moved++
	}

	if !m.loadAndResortBoard() {
		return nil
	}

	newSelected := make(map[store.State]map[string]bool)
	targetState := columnOrder[targetCol]
	newSelected[targetState] = make(map[string]bool)
	for _, r := range refs {
		newSelected[targetState][r.name] = true
	}
	m.MultiSelected = newSelected
	m.SelectedCol = targetCol
	m.ensureColVisible()

	if moved == 0 {
		m.Status = fmt.Sprintf("Ticket(s) already at %s", columnOrder[targetCol])
		return nil
	}
	return m.notify(fmt.Sprintf("Moved %d ticket(s) to %s", moved, columnOrder[targetCol]), notifSuccess)
}
