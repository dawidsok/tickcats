// navigation.go handles cursor movement within the board: switching columns,
// moving the row cursor within a column, and maintaining scroll offsets so the
// focused ticket is always visible. Column scroll uses a line-budget algorithm
// rather than a simple row count because each ticket can wrap across multiple
// terminal lines depending on the title length and column width.
package tui

import "github.com/dawidsok/tickcats/internal/store"

func (m *Model) moveColumn(delta int) {
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(m.columnOrder)-1)
	m.ensureColVisible()
	m.ensureSelectedVisible(m.columnOrder[m.SelectedCol])
}

func (m *Model) ensureColVisible() {
	visible := m.visibleColumnCount()
	if m.SelectedCol < m.ColScrollOffset {
		m.ColScrollOffset = m.SelectedCol
	}
	if m.SelectedCol >= m.ColScrollOffset+visible {
		m.ColScrollOffset = m.SelectedCol - visible + 1
	}
	m.ColScrollOffset = clamp(m.ColScrollOffset, 0, max(0, len(m.columnOrder)-visible))
}

func (m *Model) moveRow(delta int) {
	state := m.columnOrder[m.SelectedCol]
	rows := len(m.Board.Columns[state])
	if rows == 0 {
		m.SelectedRows[state] = 0
		m.ColumnScroll[state] = 0
		return
	}
	m.SelectedRows[state] = clamp(m.SelectedRows[state]+delta, 0, rows-1)
	m.ensureSelectedVisible(state)
}

// ensureSelectedVisible adjusts ColumnScroll so that the selected row is
// visible given the column's line budget. The algorithm first tries to scroll
// up (reduce scroll) to keep the selection on screen, then tries to scroll
// down only as far as needed. This ensures the selection is always rendered
// as close to the top as possible while staying in view.
func (m *Model) ensureSelectedVisible(state store.State) {
	rows := len(m.Board.Columns[state])
	if rows == 0 {
		m.SelectedRows[state] = 0
		m.ColumnScroll[state] = 0
		return
	}

	selected := clamp(m.SelectedRows[state], 0, rows-1)
	m.SelectedRows[state] = selected
	scroll := clamp(m.ColumnScroll[state], 0, rows-1)
	if scroll > selected {
		scroll = selected
	}

	budget := m.columnLineBudget()
	innerWidth := m.columnInnerWidth()
	for scroll < selected && !m.columnRangeFits(state, scroll, selected, budget, innerWidth) {
		scroll++
	}
	for scroll > 0 && m.columnRangeFits(state, scroll-1, selected, budget, innerWidth) {
		scroll--
	}

	m.ColumnScroll[state] = clamp(scroll, 0, rows-1)
}

func (m Model) columnRangeFits(state store.State, start int, selected int, budget int, innerWidth int) bool {
	if budget <= 0 {
		return false
	}
	used := 0
	if start > 0 {
		used++ // "above" indicator
	}
	for row := start; row <= selected; row++ {
		if row > start {
			used++ // separator before this ticket
		}
		used += len(m.ticketColumnLines(state, row, innerWidth))
		if used > budget {
			return false
		}
	}
	return true
}

func (m *Model) moveDetailScroll(delta int) {
	maxScroll := len(m.detailLines()) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.DetailScroll = clamp(m.DetailScroll+delta, 0, maxScroll)
}

func (m *Model) pageRows(dir int) {
	half := max(1, m.visibleTicketRows()/2)
	for range half {
		m.moveRow(dir)
	}
}

func (m Model) detailHalfPage() int {
	return max(1, m.detailPanelInnerHeight()/2)
}

func (m Model) helpPageSize() int {
	return max(1, m.helpVisibleLineCount()/2)
}

func (m *Model) moveHelpScroll(delta int) {
	maxScroll := max(0, len(helpLines())-m.helpVisibleLineCount())
	m.HelpScroll = clamp(m.HelpScroll+delta, 0, maxScroll)
}
