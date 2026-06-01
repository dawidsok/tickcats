package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHelpDialog() string {
	lines := helpLines()
	visibleLines := m.helpVisibleLineCount()
	maxScroll := max(0, len(lines)-visibleLines)
	scroll := clamp(m.HelpScroll, 0, maxScroll)
	visible := lines[scroll:min(len(lines), scroll+visibleLines)]

	contentLines := make([]string, 0, visibleLines+2)
	if scroll > 0 {
		contentLines = append(contentLines, mutedStyle.Render(fmt.Sprintf("↑ %d lines above", scroll)))
	}
	contentLines = append(contentLines, visible...)
	below := len(lines) - (scroll + len(visible))
	if below > 0 {
		contentLines = append(contentLines, mutedStyle.Render(fmt.Sprintf("↓ %d lines below", below)))
	}
	content := strings.Join(contentLines, "\n")

	width := min(72, max(48, m.fullWidth()-8))
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(width).
		Height(m.helpBoxHeight()).
		Render(content)

	height := m.Height
	if height <= 0 {
		height = 36
	}
	return lipgloss.Place(m.fullWidth(), height, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("Keyboard Shortcuts")+"\n\n"+box+"\n"+mutedStyle.Render("j/k scroll  d/u half-page  ?/enter/esc close  q quit"))
}

func helpLines() []string {
	return []string{
		bannerStyle.Render("Board"),
		"h/l, ←/→  move between columns",
		"j/k, ↓/↑  move between tickets",
		"d/u       half-page down/up",
		"enter/o   open detail view",
		"v         toggle multi-select",
		"m         move mode",
		"p / b     progress / move back one column",
		"n         new ticket",
		"e         open selected ticket in editor",
		"x         delete selected ticket",
		"s         cycle sort mode",
		"r         reload board",
		"c         config",
		"",
		bannerStyle.Render("Move mode"),
		"h/l       move focused/selected tickets one column",
		"H/L       move focused/selected tickets to first/last column",
		"j/k       reorder within column when manual sort is active",
		"esc       return to board",
		"",
		bannerStyle.Render("Detail"),
		"j/k       scroll ticket body",
		"d/u       half-page scroll",
		"e         open in editor",
		"c         config",
		"esc       return to board",
		"",
		bannerStyle.Render("Dialogs and global"),
		"tab       next field in forms",
		"h/l       change focused option in forms",
		"space     toggle checkbox in forms",
		"enter     confirm / create / save",
		"y/n       confirm or cancel prompts",
		"?         open or close this help",
		"q         quit confirmation",
		"ctrl+c    quit immediately",
	}
}

func (m Model) helpDialogHeight() int {
	return max(8, int(float64(m.helpScreenHeight())*0.8))
}

func (m Model) helpBoxHeight() int {
	// Reserve space for title, spacer, and close hint so the whole dialog fits
	// within roughly 80% of the terminal height.
	return max(6, m.helpDialogHeight()-3)
}

func (m Model) helpVisibleLineCount() int {
	// 2 border rows + 2 vertical padding rows; indicators share content space.
	return max(3, m.helpBoxHeight()-4)
}

func (m Model) helpScreenHeight() int {
	if m.Height <= 0 {
		return 36
	}
	return m.Height
}
