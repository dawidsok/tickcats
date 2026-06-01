// render_board.go renders the kanban board view: the pick-next banner, the
// horizontal scroll indicator, and each visible column with its tickets.
//
// Deadline SLA bar: tickets with a deadline show a compact progress indicator
// "SLA |||::." where each segment represents a state (ready=|||, doing=::,
// done=.). Active segments are coloured with the column's theme colour; inactive
// segments are muted. The number of active segments encodes time remaining:
// 6 = overdue, 5 = ≤1 day, 4 = ≤3 days, 3 = ≤7 days, 2 = ≤14 days, 1 = >14 days.
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
	return lipgloss.NewStyle().
		Width(m.fullWidth()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(bannerStyle.Render(text))
}

func (m Model) renderBoard() string {
	visible := m.visibleColumnCount()
	start := clamp(m.ColScrollOffset, 0, max(0, len(columnOrder)-visible))
	end := min(start+visible, len(columnOrder))
	columns := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, columnOrder[i]))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderHScrollIndicator() string {
	visible := m.visibleColumnCount()
	if visible >= len(columnOrder) {
		return ""
	}
	start := clamp(m.ColScrollOffset, 0, max(0, len(columnOrder)-visible))
	leftHidden := start
	rightHidden := len(columnOrder) - (start + visible)
	var parts []string
	if leftHidden > 0 {
		names := make([]string, leftHidden)
		for i := range leftHidden {
			names[i] = columnOrder[i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("← %s", strings.Join(names, ", ")))
	}
	if rightHidden > 0 {
		names := make([]string, rightHidden)
		for i := range rightHidden {
			names[i] = columnOrder[start+visible+i].DisplayName()
		}
		parts = append(parts, fmt.Sprintf("%s →", strings.Join(names, ", ")))
	}
	return mutedStyle.Render(strings.Join(parts, "  ")) + "\n"
}

func (m Model) renderColumn(index int, state store.State) string {
	var b strings.Builder
	header := strings.ToUpper(state.DisplayName())
	if index == m.SelectedCol {
		header = m.colStyle(index).Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n")

	for _, line := range m.renderColumnLines(index, state) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(m.columnWidth()).
		Height(m.boardColumnHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1).
		Render(b.String())
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
		lines = append(lines, deadlineIndicatorPlain(*stored.Ticket.Deadline, time.Now()))
	}
	return lines
}

func (m Model) styledTicketColumnLines(index int, state store.State, row int, innerWidth int) []string {
	tickets := m.filteredTickets(state)
	if row < 0 || row >= len(tickets) {
		return nil
	}
	stored := tickets[row]
	isFocused := m.InteractionMode != InteractionSearch && index == m.SelectedCol && row == m.SelectedRows[state]
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
		wrapped = append(wrapped, m.renderDeadlineIndicator(*stored.Ticket.Deadline, time.Now()))
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

func deadlineIndicatorPlain(deadline time.Time, now time.Time) string {
	return "  SLA " + deadlineBarPattern()
}

func (m Model) renderDeadlineIndicator(deadline time.Time, now time.Time) string {
	activeBars := deadlineBarCount(deadline, now)
	var b strings.Builder
	b.WriteString(mutedStyle.Render("  SLA "))
	b.WriteString(m.renderDeadlineBarSegment(store.StateReady, min(activeBars, 3), 3, "|"))
	b.WriteString(m.renderDeadlineBarSegment(store.StateDoing, clamp(activeBars-3, 0, 2), 2, ":"))
	b.WriteString(m.renderDeadlineBarSegment(store.StateDone, clamp(activeBars-5, 0, 1), 1, "."))
	return b.String()
}

func (m Model) renderDeadlineBarSegment(state store.State, active int, total int, marker string) string {
	active = clamp(active, 0, total)
	var b strings.Builder
	if active > 0 {
		b.WriteString(m.colStyle(stateColIndex(state)).Render(strings.Repeat(marker, active)))
	}
	if inactive := total - active; inactive > 0 {
		b.WriteString(mutedStyle.Render(strings.Repeat(marker, inactive)))
	}
	return b.String()
}

func deadlineBarPattern() string {
	return "|||::."
}

func deadlineBarStates() []store.State {
	return []store.State{
		store.StateReady,
		store.StateReady,
		store.StateReady,
		store.StateDoing,
		store.StateDoing,
		store.StateDone,
	}
}

func deadlineBarCount(deadline time.Time, now time.Time) int {
	days := daysUntil(deadline, now)
	switch {
	case days <= 0:
		return 6
	case days <= 1:
		return 5
	case days <= 3:
		return 4
	case days <= 7:
		return 3
	case days <= 14:
		return 2
	default:
		return 1
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
