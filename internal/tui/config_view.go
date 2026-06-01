// config_view.go implements the Config settings form (ViewConfig). It exposes
// two settings: the editor command (preset list + custom text input) and the
// colour theme. Changes are written to config.json on Enter.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) enterConfig() (tea.Model, tea.Cmd) {
	idx := len(editorPresets) // default to "custom"
	for i, p := range editorPresets {
		if p == m.Config.Editor {
			idx = i
			break
		}
	}
	input := textinput.New()
	input.Placeholder = "editor command"
	input.CharLimit = 200
	m.configField = 0
	m.configEditorIdx = idx
	m.configEditorInput = input
	m.Mode = ViewConfig
	m.Status = ""
	if idx == len(editorPresets) {
		m.configEditorInput.SetValue(m.Config.Editor)
		cmd := m.configEditorInput.Focus()
		return m, cmd
	}
	return m, nil
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.Mode = ViewBoard
			m.Status = ""
			return m, nil
		case "tab", "shift+tab":
			m.configField = (m.configField + 1) % 2
			if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
				cmd := m.configEditorInput.Focus()
				return m, cmd
			}
			m.configEditorInput.Blur()
			return m, nil
		case "h", "left":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme-1, 0, len(colorThemes)-1)
			} else {
				if m.configEditorIdx > 0 {
					m.configEditorIdx--
				}
				m.configEditorInput.Blur()
			}
			return m, nil
		case "l", "right":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme+1, 0, len(colorThemes)-1)
			} else {
				if m.configEditorIdx < len(editorPresets) {
					m.configEditorIdx++
				}
				if m.configEditorIdx == len(editorPresets) {
					cmd := m.configEditorInput.Focus()
					return m, cmd
				}
				m.configEditorInput.Blur()
			}
			return m, nil
		case "enter":
			m.saveConfig()
			m.Mode = ViewBoard
			return m, m.notify("Config saved", notifSuccess)
		default:
			if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
				var cmd tea.Cmd
				m.configEditorInput, cmd = m.configEditorInput.Update(keyMsg)
				return m, cmd
			}
		}
		return m, nil
	}
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		var cmd tea.Cmd
		m.configEditorInput, cmd = m.configEditorInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) saveConfig() {
	if m.configEditorIdx == len(editorPresets) {
		m.Config.Editor = strings.TrimSpace(m.configEditorInput.Value())
	} else {
		m.Config.Editor = editorPresets[m.configEditorIdx]
	}
	_ = store.SaveConfig(m.Root, m.Config)
}

func (m Model) renderConfig() string {
	formWidth := m.fullWidth() - 4
	if formWidth > 64 {
		formWidth = 64
	}
	if formWidth < 44 {
		formWidth = 44
	}

	labelW := 9
	labelStyle := mutedStyle.Width(labelW)
	activeStyle := func(active bool) lipgloss.Style {
		if active {
			return selectedStyle.Width(labelW)
		}
		return labelStyle
	}

	// Editor row
	editorLabel := activeStyle(m.configField == 0).Render("Editor")
	presetNames := make([]string, 0, len(editorPresets)+1)
	for _, p := range editorPresets {
		name := p
		if name == "" {
			name = "$EDITOR"
		}
		presetNames = append(presetNames, name)
	}
	presetNames = append(presetNames, "custom")
	editorParts := make([]string, 0, len(presetNames))
	for i, name := range presetNames {
		if i == m.configEditorIdx {
			if m.configField == 0 {
				editorParts = append(editorParts, selectedStyle.Render("["+name+"]"))
			} else {
				editorParts = append(editorParts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			editorParts = append(editorParts, mutedStyle.Render(name))
		}
	}
	editorRow := editorLabel + strings.Join(editorParts, "  ")

	// Theme row
	themeLabel := activeStyle(m.configField == 1).Render("Theme")
	themeParts := make([]string, 0, len(colorThemes))
	for i, t := range colorThemes {
		if i == m.Config.Theme {
			if m.configField == 1 {
				themeParts = append(themeParts, selectedStyle.Render("["+t.name+"]"))
			} else {
				themeParts = append(themeParts, lipgloss.NewStyle().Bold(true).Render("["+t.name+"]"))
			}
		} else {
			themeParts = append(themeParts, mutedStyle.Render(t.name))
		}
	}
	themeRow := themeLabel + strings.Join(themeParts, "  ")

	var rows []string
	rows = append(rows, editorRow)

	// Show text input when custom is selected
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		inputWidth := formWidth - labelW - 6
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.configEditorInput.Width = inputWidth
		rows = append(rows, "", labelStyle.Render("")+m.configEditorInput.View())
	}

	rows = append(rows, "", themeRow)
	rows = append(rows, "", mutedStyle.Render("tab field  h/l select  enter save  esc cancel"))
	content := strings.Join(rows, "\n")

	box := dialogBoxStyle(formWidth, 0).Render(content)

	var statusLine string
	if m.Status != "" {
		statusLine = "\n" + mutedStyle.Render(m.Status)
	}

	return m.placeDialog("Config", box, statusLine, m.safeHeight(24))
}
