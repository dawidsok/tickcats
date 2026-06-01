// search.go implements the slash-activated fuzzy search overlay. Pressing '/'
// from board mode opens a text field that filters ticket titles and bodies
// across all columns in real time. Esc clears the query and exits search.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) enterSearch() (tea.Model, tea.Cmd) {
	input := textinput.New()
	input.Placeholder = "fuzzy filter..."
	input.CharLimit = 100
	m.searchInput = input
	m.InteractionMode = InteractionSearch
	cmd := m.searchInput.Focus()
	return m, cmd
}

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.InteractionMode = InteractionBoard
			m.searchInput = textinput.Model{}
			for _, state := range columnOrder {
				tickets := m.Board.Columns[state]
				if m.SelectedRows[state] >= len(tickets) && len(tickets) > 0 {
					m.SelectedRows[state] = len(tickets) - 1
				}
			}
			return m, nil
		case "j", "down":
			m.moveInSearch(1)
			return m, nil
		case "k", "up":
			m.moveInSearch(-1)
			return m, nil
		case "h", "left":
			m.moveColumn(-1)
			return m, nil
		case "l", "right":
			m.moveColumn(1)
			return m, nil
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(keyMsg)
		return m, cmd
	}
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = sizeMsg.Width
		m.Height = sizeMsg.Height
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// filteredTickets returns the ticket slice for state filtered by the current
// search query. Returns the full slice when search is inactive or the query is
// empty.
func (m Model) filteredTickets(state store.State) []store.StoredTicket {
	if m.InteractionMode != InteractionSearch {
		return m.Board.Columns[state]
	}
	q := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if q == "" {
		return m.Board.Columns[state]
	}
	tickets := m.Board.Columns[state]
	result := make([]store.StoredTicket, 0, len(tickets))
	for _, t := range tickets {
		// Include priority so "p0" matches only P0 tickets, not P2.
		haystack := strings.ToLower(string(t.Ticket.Priority) + " " + t.Ticket.Title + " " + t.Ticket.Body)
		if fuzzyMatch(q, haystack) {
			result = append(result, t)
		}
	}
	return result
}

// moveInSearch moves the cursor within the filtered list of the current column.
// If the selected ticket is not in the filtered results, it jumps to the first
// (delta>0) or last (delta<0) filtered ticket.
func (m *Model) moveInSearch(delta int) {
	state := columnOrder[m.SelectedCol]
	filtered := m.filteredTickets(state)
	if len(filtered) == 0 {
		return
	}
	// Identify current selection by name so the index stays meaningful even
	// when the filtered list is a subset of the full list.
	selectedName := ""
	if full := m.Board.Columns[state]; m.SelectedRows[state] < len(full) {
		selectedName = full[m.SelectedRows[state]].Name
	}
	currentIdx := -1
	for i, t := range filtered {
		if t.Name == selectedName {
			currentIdx = i
			break
		}
	}
	var newIdx int
	if currentIdx < 0 {
		if delta > 0 {
			newIdx = 0
		} else {
			newIdx = len(filtered) - 1
		}
	} else {
		newIdx = clamp(currentIdx+delta, 0, len(filtered)-1)
	}
	// Map the filtered index back to a full-list index.
	targetName := filtered[newIdx].Name
	for i, t := range m.Board.Columns[state] {
		if t.Name == targetName {
			m.SelectedRows[state] = i
			m.ensureSelectedVisible(state)
			return
		}
	}
}

// fuzzyMatch reports whether every rune in pattern appears in text in order
// (case-insensitive — callers must lowercase both sides first).
func fuzzyMatch(pattern, text string) bool {
	patternRunes := []rune(pattern)
	pi := 0
	for _, r := range text {
		if pi >= len(patternRunes) {
			break
		}
		if r == patternRunes[pi] {
			pi++
		}
	}
	return pi >= len(patternRunes)
}

func (m Model) renderSearchBar() string {
	var b strings.Builder
	b.WriteString("/")
	b.WriteString(" ")
	b.WriteString(m.searchInput.View())
	q := strings.TrimSpace(m.searchInput.Value())
	if q != "" {
		total := 0
		for _, state := range columnOrder {
			total += len(m.filteredTickets(state))
		}
		b.WriteString("  ")
		b.WriteString(mutedStyle.Render(fmt.Sprintf("%d match(es)", total)))
	}
	return b.String()
}
