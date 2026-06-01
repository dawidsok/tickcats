package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderPostCreateDialog() string {
	name := filepath.Base(m.createPending)

	content := strings.Join([]string{
		mutedStyle.Render("Created:"),
		"",
		selectedStyle.Render(name),
		"",
		"Open in external editor?",
		"",
		selectedStyle.Render("y") + "  open in editor",
		mutedStyle.Render("n") + "  stay in TickCats",
		mutedStyle.Render("d") + "  don't ask again",
	}, "\n")

	formWidth := 48
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 3).
		Width(formWidth).
		Render(content)

	h := m.Height
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(m.fullWidth(), h, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("Ticket Created")+"\n\n"+box)
}
