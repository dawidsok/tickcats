// render_board.go renders the kanban board view: the pick-next banner, the
// horizontal scroll indicator, and each visible column with its tickets.
//
// Deadlines: tickets with a deadline show the date directly below the title.
// The date color becomes more urgent as the deadline approaches.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) renderPickNext() string {
	result := store.PickNext(m.Board)
	text := "Next: none"
	if result.HasPick && result.NeedsChoice {
		text = fmt.Sprintf("Next: %d tied candidates", len(result.Tied))
	} else if result.HasPick {
		text = fmt.Sprintf("Next: [%s] %s", result.Ticket.Ticket.Priority, result.Ticket.Ticket.Title)
	}
	color := m.themeColor(m.stateColIndex(store.StateDoing))
	styled := lipgloss.NewStyle().Bold(true).Foreground(color).Render(text)
	return lipgloss.NewStyle().
		Width(max(1, m.fullWidth()-2)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Render(styled)
}

func (m Model) renderBoard() string {
	visible := m.visibleColumnCount()
	start := clamp(m.ColScrollOffset, 0, max(0, len(m.columnOrder)-visible))
	end := min(start+visible, len(m.columnOrder))
	columns := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, m.columnOrder[i]))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderHScrollIndicator() string {
	visible := m.visibleColumnCount()
	if visible >= len(m.columnOrder) {
		return ""
	}
	start := clamp(m.ColScrollOffset, 0, max(0, len(m.columnOrder)-visible))
	leftHidden := start
	rightHidden := len(m.columnOrder) - (start + visible)
	var parts []string
	if leftHidden > 0 {
		names := make([]string, leftHidden)
		for i := range leftHidden {
			names[i] = m.columnOrder[i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("← %s", strings.Join(names, ", ")))
	}
	if rightHidden > 0 {
		names := make([]string, rightHidden)
		for i := range rightHidden {
			names[i] = m.columnOrder[start+visible+i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("%s →", strings.Join(names, ", ")))
	}
	return mutedStyle.Render(fitText(strings.Join(parts, "  "), m.fullWidth())) + "\n"
}

func (m Model) renderColumn(index int, state store.State) string {
	header := strings.ToUpper(state.DisplayName())
	var headerBorderColor lipgloss.Color
	if index == m.SelectedCol {
		header = m.colStyle(index).Render(header)
		headerBorderColor = m.themeColor(index)
	} else {
		headerBorderColor = lipgloss.Color("240")
	}

	headerBox := lipgloss.NewStyle().
		Width(m.columnWidth()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(headerBorderColor).
		Padding(0, 1).
		Render(header)

	var body strings.Builder
	for _, line := range m.renderColumnLines(index, state) {
		body.WriteString(line)
		body.WriteString("\n")
	}
	bodyBox := lipgloss.NewStyle().
		Width(m.columnWidth()).
		Height(m.boardColumnInnerHeight()-3).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1).
		Render(body.String())

	return lipgloss.JoinVertical(lipgloss.Left, headerBox, bodyBox)
}

func (m Model) renderColumnLines(index int, state store.State) []string {
	tickets := m.filteredTickets(state)
	scroll := clamp(m.ColumnScroll[state], 0, max(0, len(tickets)-1))
	innerWidth := m.columnInnerWidth()
	maxLines := m.columnLineBudget()
	lines := make([]string, 0, maxLines)

	appendLine := func(line string) bool {
		if len(lines) >= maxLines {
			return false
		}
		lines = append(lines, line)
		return true
	}

	if len(tickets) == 0 {
		appendLine(mutedStyle.Render("  empty"))
		return lines
	}

	if scroll > 0 {
		appendLine(mutedStyle.Render(fmt.Sprintf("  ↑ %d above", scroll)))
	}

	selectedRow := m.SelectedRows[state]
	separator := mutedStyle.Render(strings.Repeat("─", innerWidth))
	for row := scroll; row < len(tickets); row++ {
		if row > scroll {
			if !appendLine(separator) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row, row-1, selectedRow)
			}
		}

		for _, line := range m.styledTicketColumnLines(index, state, row, innerWidth) {
			if !appendLine(line) {
				return appendColumnOverflow(lines, innerWidth, len(tickets)-row, row, selectedRow)
			}
		}
	}

	return lines
}

// ticketColumnLines returns the plain (unstyled) wrapped lines for a ticket,
// used by the layout engine to measure how many lines a ticket occupies.
func (m Model) ticketColumnLines(state store.State, row int, innerWidth int) []string {
	tickets := m.filteredTickets(state)
	if row < 0 || row >= len(tickets) {
		return nil
	}
	stored := tickets[row]
	prefix := "  "
	if m.MultiSelected[state] != nil && m.MultiSelected[state][stored.Name] {
		prefix = "* "
	}
	lines := wrapText(fmt.Sprintf("%s[%s] %s", prefix, stored.Ticket.Priority, stored.Ticket.Title), innerWidth)
	if stored.Ticket.Deadline != nil {
		lines = append(lines, deadlineDatePlain(*stored.Ticket.Deadline, time.Now()))
	}
	return lines
}

func (m Model) styledTicketColumnLines(index int, state store.State, row int, innerWidth int) []string {
	tickets := m.filteredTickets(state)
	if row < 0 || row >= len(tickets) {
		return nil
	}
	stored := tickets[row]
	var isFocused bool
	if m.InteractionMode == InteractionSearch {
		// In search mode the filtered list has different indices than the full
		// list, so identify the selected ticket by name instead.
		selectedName := ""
		if full := m.Board.Columns[state]; m.SelectedRows[state] < len(full) {
			selectedName = full[m.SelectedRows[state]].Name
		}
		isFocused = index == m.SelectedCol && selectedName != "" && stored.Name == selectedName
	} else {
		isFocused = index == m.SelectedCol && row == m.SelectedRows[state]
	}
	isSelected := m.MultiSelected[state] != nil && m.MultiSelected[state][stored.Name]

	var prefix string
	switch {
	case isFocused && isSelected:
		prefix = ">*"
	case isFocused:
		prefix = "> "
	case isSelected:
		prefix = "* "
	default:
		prefix = "  "
	}

	wrapped := wrapText(fmt.Sprintf("%s[%s] %s", prefix, stored.Ticket.Priority, stored.Ticket.Title), innerWidth)
	for i, line := range wrapped {
		switch {
		case isFocused:
			wrapped[i] = m.colStyle(index).Render(line)
		case isSelected:
			wrapped[i] = selectedStyle.Render(line)
		}
	}
	if stored.Ticket.Deadline != nil {
		wrapped = append(wrapped, m.renderDeadlineDate(state, *stored.Ticket.Deadline, time.Now()))
	}
	return wrapped
}

// appendColumnOverflow replaces the last rendered line with a "↓ N below"
// indicator when the column content overflows the available height. The
// replacement is skipped when the last rendered row is the selected one,
// because hiding the selected ticket would be confusing.
func appendColumnOverflow(lines []string, width int, below int, lastRenderedRow int, selectedRow int) []string {
	if below <= 0 || len(lines) == 0 || lastRenderedRow == selectedRow {
		return lines
	}
	lines[len(lines)-1] = mutedStyle.Render(fmt.Sprintf("  ↓ %d below", below))
	return lines
}

func deadlineDatePlain(deadline time.Time, now time.Time) string {
	return deadline.Format(time.DateOnly)
}

func (m Model) renderDeadlineDate(state store.State, deadline time.Time, now time.Time) string {
	if state == store.StateDone {
		return mutedStyle.Render(deadlineDatePlain(deadline, now))
	}
	style := lipgloss.NewStyle().Foreground(m.deadlineDateColor(state, deadline, now))
	if daysUntil(deadline, now) <= 1 {
		style = style.Bold(true)
	}
	return style.Render(deadlineDatePlain(deadline, now))
}

func (m Model) deadlineDateColor(state store.State, deadline time.Time, now time.Time) lipgloss.Color {
	return deadlineDateColor(m.Config.Theme, state, deadline, now)
}

func deadlineDateColor(themeIdx int, state store.State, deadline time.Time, now time.Time) lipgloss.Color {
	if state == store.StateDone {
		return lipgloss.Color("240")
	}
	step := deadlineUrgencyStep(deadline, now)
	// Use five virtual gradient stops from the currently selected theme:
	// closest deadlines use the theme's start color, distant deadlines use its end color.
	return themeColor(themeIdx, 1+(4-step), 6)
}

func deadlineUrgencyStep(deadline time.Time, now time.Time) int {
	days := daysUntil(deadline, now)
	switch {
	case days <= 1:
		return 4
	case days <= 3:
		return 3
	case days <= 7:
		return 2
	case days <= 14:
		return 1
	default:
		return 0
	}
}

func daysUntil(deadline time.Time, now time.Time) int {
	dueDate := dateOnly(deadline.UTC())
	today := dateOnly(now.UTC())
	return int(dueDate.Sub(today).Hours() / 24)
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
