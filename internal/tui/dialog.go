// dialog.go provides shared primitives used by all dialog implementations.
// Each dialog file (help_dialog.go, quit_confirm_dialog.go, etc.) builds on
// these three building blocks:
//   - enterInteraction / dismissInteraction: save and restore the previous
//     mode so dialogs can be dismissed back to wherever the user was.
//   - dialogBoxStyle: the standard rounded-border lipgloss style used by every
//     modal box in the application.
//   - placeDialog: centres a titled box on screen using lipgloss.Place.
package tui

import "github.com/charmbracelet/lipgloss"

// enterInteraction saves the current Mode and InteractionMode, then switches
// to the given dialog interaction mode.
func (m Model) enterInteraction(mode InteractionMode) Model {
	m.prevMode = m.Mode
	m.prevInteractionMode = m.InteractionMode
	m.InteractionMode = mode
	return m
}

func (m Model) dismissInteraction() Model {
	m.Mode = m.prevMode
	m.InteractionMode = m.prevInteractionMode
	return m
}

// dialogBoxStyle returns the standard lipgloss style for modal dialog boxes.
func dialogBoxStyle(width, height int) lipgloss.Style {
	s := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(width)
	if height > 0 {
		s = s.Height(height)
	}
	return s
}

// placeDialog centers a titled dialog box on screen. footer is appended after the box.
func (m Model) placeDialog(title, box, footer string, screenHeight int) string {
	return lipgloss.Place(m.fullWidth(), screenHeight, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render(title)+"\n\n"+box+footer)
}
