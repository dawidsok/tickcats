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
		haystack := strings.ToLower(t.Ticket.Title + " " + t.Ticket.Body)
		if fuzzyMatch(q, haystack) {
			result = append(result, t)
		}
	}
	return result
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
