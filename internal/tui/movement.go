// movement.go handles single-ticket column moves triggered by the p/b keys in
// board mode and the h/l keys in move mode. For bulk moves see moveAllSelectedBy
// and moveAllSelectedTo in actions.go.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m *Model) moveSelected(delta int) tea.Cmd {
	stored := m.selectedTicket()
	if stored == nil {
		m.Status = "No ticket selected"
		return nil
	}

	from := columnOrder[m.SelectedCol]
	toIndex := m.SelectedCol + delta
	if toIndex < 0 {
		m.Status = fmt.Sprintf("Ticket already in %s", columnOrder[0])
		return nil
	}
	if toIndex >= len(columnOrder) {
		m.Status = fmt.Sprintf("Ticket already in %s", columnOrder[len(columnOrder)-1])
		return nil
	}
	to := columnOrder[toIndex]

	if _, err := store.Move(m.Root, stored.Name, from, to); err != nil {
		m.Status = "Move failed: " + err.Error()
		return nil
	}

	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return nil
	}

	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	m.SelectedCol = toIndex
	m.ensureColVisible()
	m.SelectedRows[to] = findTicketRow(m.Board.Columns[to], stored.Name)
	m.ensureSelectedVisible(to)
	return m.notify(fmt.Sprintf("Moved %s to %s", stored.Name, to), notifSuccess)
}

func (m Model) selectedTicket() *store.StoredTicket {
	state := columnOrder[m.SelectedCol]
	tickets := m.Board.Columns[state]
	if len(tickets) == 0 {
		return nil
	}
	row := clamp(m.SelectedRows[state], 0, len(tickets)-1)
	return &tickets[row]
}

func findTicketRow(tickets []store.StoredTicket, name string) int {
	for i, stored := range tickets {
		if stored.Name == name {
			return i
		}
	}
	return 0
}
