// quit_confirm_dialog.go implements the "QUIT? y/q confirm  n/esc cancel"
// overlay that appears when the user presses q outside of text-input modes.
package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) enterQuitConfirm() (tea.Model, tea.Cmd) {
	return m.enterInteraction(InteractionQuitConfirm), nil
}

func (m Model) updateQuitConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "q":
		return m, tea.Quit
	case "n", "esc":
		return m.dismissInteraction(), nil
	}
	return m, nil
}
