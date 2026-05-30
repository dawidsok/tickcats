package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
)

var columnOrder = []store.State{store.StateBacklog, store.StateReady, store.StateDoing, store.StateDone}

var (
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bannerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
)

type ViewMode int

const (
	ViewBoard ViewMode = iota
	ViewDetail
)

type Model struct {
	Board        store.Board
	SelectedCol  int
	SelectedRows map[store.State]int
	Mode         ViewMode
	DetailScroll int
}

func NewModel(board store.Board) Model {
	return Model{
		Board:        board,
		SelectedRows: make(map[store.State]int),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Mode == ViewDetail {
			return m.updateDetail(msg)
		}
		return m.updateBoard(msg)
	}
	return m, nil
}

func (m Model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m.moveColumn(-1)
	case "l", "right":
		m.moveColumn(1)
	case "j", "down":
		m.moveRow(1)
	case "k", "up":
		m.moveRow(-1)
	case "enter":
		if m.selectedTicket() != nil {
			m.Mode = ViewDetail
			m.DetailScroll = 0
		}
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.Mode = ViewBoard
		m.DetailScroll = 0
	case "j", "down":
		m.moveDetailScroll(1)
	case "k", "up":
		m.moveDetailScroll(-1)
	}
	return m, nil
}

func (m Model) View() string {
	if m.Mode == ViewDetail {
		return m.renderDetail()
	}

	var b strings.Builder
	b.WriteString(m.renderPickNext())
	b.WriteString("\n\n")
	b.WriteString(m.renderBoard())
	b.WriteString("\n")
	b.WriteString(m.renderWarnings())
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("h/l column  j/k ticket  enter detail  q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m *Model) moveColumn(delta int) {
	m.SelectedCol = clamp(m.SelectedCol+delta, 0, len(columnOrder)-1)
}

func (m *Model) moveRow(delta int) {
	state := columnOrder[m.SelectedCol]
	rows := len(m.Board.Columns[state])
	if rows == 0 {
		m.SelectedRows[state] = 0
		return
	}
	m.SelectedRows[state] = clamp(m.SelectedRows[state]+delta, 0, rows-1)
}

func (m *Model) moveDetailScroll(delta int) {
	maxScroll := len(m.detailLines()) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.DetailScroll = clamp(m.DetailScroll+delta, 0, maxScroll)
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

func (m Model) renderPickNext() string {
	result := store.PickNext(m.Board)
	if !result.HasPick {
		return bannerStyle.Render("Next: none")
	}
	if result.NeedsChoice {
		return bannerStyle.Render(fmt.Sprintf("Next: %d tied candidates", len(result.Tied)))
	}
	return bannerStyle.Render(fmt.Sprintf("Next: [%s] %s", result.Ticket.Ticket.Priority, result.Ticket.Ticket.Title))
}

func (m Model) renderBoard() string {
	columns := make([]string, 0, len(columnOrder))
	for i, state := range columnOrder {
		columns = append(columns, m.renderColumn(i, state))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m Model) renderColumn(index int, state store.State) string {
	var b strings.Builder
	header := strings.ToUpper(string(state))
	if index == m.SelectedCol {
		header = selectedStyle.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n")

	tickets := m.Board.Columns[state]
	if len(tickets) == 0 {
		b.WriteString(mutedStyle.Render("  empty"))
		b.WriteString("\n")
	} else {
		selectedRow := m.SelectedRows[state]
		for row, stored := range tickets {
			line := fmt.Sprintf("  [%s] %s", stored.Ticket.Priority, stored.Ticket.Title)
			if index == m.SelectedCol && row == selectedRow {
				line = selectedStyle.Render("> " + line[2:])
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().Width(32).PaddingRight(2).Render(b.String())
}

func (m Model) renderDetail() string {
	stored := m.selectedTicket()
	if stored == nil {
		return "No ticket selected\n\n" + mutedStyle.Render("esc back  q quit") + "\n"
	}

	lines := m.detailLines()
	visible := lines[m.DetailScroll:]
	if len(visible) > 18 {
		visible = visible[:18]
	}

	var b strings.Builder
	b.WriteString(bannerStyle.Render(stored.Ticket.Title))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("State: %s  Priority: %s  File: %s\n", stored.State, stored.Ticket.Priority, stored.Name))
	if len(stored.Ticket.ParsedTitle.Labels) > 0 {
		b.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(stored.Ticket.ParsedTitle.Labels, ", ")))
	}
	b.WriteString("\n")
	b.WriteString(strings.Join(visible, "\n"))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("j/k scroll  esc back  q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) detailLines() []string {
	stored := m.selectedTicket()
	if stored == nil {
		return nil
	}
	body := strings.TrimRight(stored.Ticket.Body, "\n")
	if body == "" {
		return []string{mutedStyle.Render("empty body")}
	}
	return strings.Split(body, "\n")
}

func (m Model) renderWarnings() string {
	if len(m.Board.Warnings) == 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("Warnings: %d malformed ticket(s) skipped", len(m.Board.Warnings))) + "\n"
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
