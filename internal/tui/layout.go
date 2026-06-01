// layout.go computes derived dimensions from the terminal size (m.Width,
// m.Height) used by render functions throughout the TUI. All dimension helpers
// return sensible fallback values when the terminal reports zero size (e.g.
// during tests or before the first WindowSizeMsg).
// renderFooter and footerText also live here because the footer is tightly
// coupled to layout dimensions and interaction mode display strings.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) boardColumnHeight() int {
	return m.boardColumnInnerHeight() + 2
}

func (m Model) boardColumnInnerHeight() int {
	if m.Height <= 0 {
		return 18
	}
	height := m.Height - 11
	if height < 6 {
		return 6
	}
	return height
}

func (m Model) visibleTicketRows() int {
	rows := m.boardColumnInnerHeight() - 3
	if rows < 1 {
		return 1
	}
	return rows
}

func (m Model) columnLineBudget() int {
	lines := m.boardColumnInnerHeight() - 1 // reserve one line for the column header
	if lines < 1 {
		return 1
	}
	return lines
}

func (m Model) detailPanelHeight() int {
	return m.detailPanelInnerHeight() + 2
}

func (m Model) detailPanelInnerHeight() int {
	if m.Height <= 0 {
		return 18
	}
	height := m.Height - 7
	if height < 6 {
		return 6
	}
	return height
}

func (m Model) fullWidth() int {
	if m.Width <= 0 {
		return 120
	}
	if m.Width < 40 {
		return 40
	}
	return m.Width
}

func (m Model) visibleColumnCount() int {
	if m.Width <= 0 {
		return len(columnOrder)
	}
	count := m.Width / minColumnWidth
	if count < 1 {
		count = 1
	}
	if count > len(columnOrder) {
		count = len(columnOrder)
	}
	return count
}

func (m Model) columnWidth() int {
	visible := m.visibleColumnCount()
	if m.Width <= 0 {
		return 32
	}
	width := (m.Width / visible) - 2
	if width < 20 {
		return 20
	}
	return width
}

func (m Model) columnInnerWidth() int {
	width := m.columnWidth() - 2
	if width < 1 {
		return 1
	}
	return width
}

func (m Model) renderFooter() string {
	line := m.renderFooterSeparator()
	return line + "\n" + mutedStyle.Render(m.footerText()) + "\n"
}

func (m Model) footerText() string {
	if m.InteractionMode == InteractionQuitConfirm {
		return "QUIT? y/q confirm  n/esc cancel"
	}
	if m.InteractionMode == InteractionHelp {
		return "HELP: ?/enter/esc close  q quit"
	}
	if m.InteractionMode == InteractionPostCreate {
		return "y open editor  n/esc stay  d don't ask again"
	}
	if m.InteractionMode == InteractionDeleteConfirm {
		return "DELETE? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionSortPrompt {
		return "Switch to manual sort? y confirm  n/esc cancel  q quit"
	}
	if m.InteractionMode == InteractionMove {
		sel := m.totalSelected()
		if sel > 0 {
			return fmt.Sprintf("MOVE (%d): h/l move  H/L ends  ? help  esc board  q quit", sel)
		}
		return "MOVE: h/l move  H/L ends  j/k reorder  ? help  esc board  q quit"
	}
	if m.Mode == ViewDetail {
		return "DETAIL: j/k scroll  e edit  ? help  esc board  q quit"
	}
	return "BOARD: h/l columns  j/k tickets  enter detail  m move  n new  ? help  q quit"
}

func (m Model) renderFooterSeparator() string {
	line := strings.Repeat("─", m.fullWidth())
	snack := m.renderSnack()
	if snack == "" {
		return mutedStyle.Render(line)
	}
	plainWidth := lipgloss.Width(snack)
	if plainWidth+2 >= m.fullWidth() {
		return snack
	}
	return snack + mutedStyle.Render(strings.Repeat("─", m.fullWidth()-plainWidth))
}

func (m Model) renderSnack() string {
	if m.notification != nil {
		switch m.notification.kind {
		case notifSuccess:
			return notifSuccessStyle.Render("✓ " + m.notification.text + " ")
		case notifError:
			return notifErrorStyle.Render("✗ " + m.notification.text + " ")
		default:
			return mutedStyle.Render(m.notification.text + " ")
		}
	}
	parts := make([]string, 0, 2)
	if m.Status != "" {
		parts = append(parts, m.Status)
	}
	if len(m.Board.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("Warnings: %d ticket issue(s)", len(m.Board.Warnings)))
	}
	if len(parts) == 0 {
		return ""
	}
	return mutedStyle.Render(strings.Join(parts, "  •  ") + " ")
}
