package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) enterQuitConfirm() (tea.Model, tea.Cmd) {
	m.prevMode = m.Mode
	m.prevInteractionMode = m.InteractionMode
	m.InteractionMode = InteractionQuitConfirm
	return m, nil
}

func (m Model) enterHelp() (tea.Model, tea.Cmd) {
	m.prevMode = m.Mode
	m.prevInteractionMode = m.InteractionMode
	m.InteractionMode = InteractionHelp
	m.HelpScroll = 0
	return m, nil
}

func (m Model) updateQuitConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "q":
		return m, tea.Quit
	case "n", "esc":
		m.Mode = m.prevMode
		m.InteractionMode = m.prevInteractionMode
	}
	return m, nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "enter":
		m.Mode = m.prevMode
		m.InteractionMode = m.prevInteractionMode
		m.HelpScroll = 0
	case "j", "down":
		m.moveHelpScroll(1)
	case "k", "up":
		m.moveHelpScroll(-1)
	case "d":
		m.moveHelpScroll(m.helpPageSize())
	case "u":
		m.moveHelpScroll(-m.helpPageSize())
	}
	return m, nil
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
