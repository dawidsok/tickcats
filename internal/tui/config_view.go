// config_view.go implements the Config settings form (ViewConfig). It exposes
// the editor command, colour theme, and keyboard-first column management.
// Changes to editor/theme are written to config.json on Enter; column mutations
// are applied immediately through the store layer.
package tui

import (
	"fmt"
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
	m.configColIdx = clamp(m.configColIdx, 0, len(m.Config.GetColumns())-1)
	m.configAction = configActionNone
	m.configColumnInput = textinput.Model{}
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
		if m.configAction != configActionNone {
			return m.updateConfigAction(keyMsg)
		}

		if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
			switch keyMsg.String() {
			case "ctrl+c", "esc", "tab", "shift+tab", "enter":
				// handled by the normal config switch below
			default:
				var cmd tea.Cmd
				m.configEditorInput, cmd = m.configEditorInput.Update(keyMsg)
				return m, cmd
			}
		}

		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.Mode = ViewBoard
			m.Status = ""
			return m, nil
		case "tab":
			m.configField = (m.configField + 1) % 3
			return m.focusConfigField()
		case "shift+tab":
			m.configField = (m.configField + 2) % 3
			return m.focusConfigField()
		case "h", "left":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme-1, 0, len(colorThemes)-1)
			} else if m.configField == 0 {
				if m.configEditorIdx > 0 {
					m.configEditorIdx--
				}
				if m.configEditorIdx != len(editorPresets) {
					m.configEditorInput.Blur()
				}
			}
			return m, nil
		case "l", "right":
			if m.configField == 1 {
				m.Config.Theme = clamp(m.Config.Theme+1, 0, len(colorThemes)-1)
			} else if m.configField == 0 {
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
		case "j", "down":
			if m.configField == 2 {
				m.moveConfigColumnSelection(1)
			}
			return m, nil
		case "k", "up":
			if m.configField == 2 {
				m.moveConfigColumnSelection(-1)
			}
			return m, nil
		case "a":
			if m.configField == 2 {
				m.configAction = configActionAddName
				input := textinput.New()
				input.Placeholder = "New column name"
				input.CharLimit = 80
				input.Focus()
				m.configColumnInput = input
				m.Status = ""
				return m, nil
			}
		case "r":
			if m.configField == 2 {
				cols := m.Config.GetColumns()
				if m.configColIdx >= 0 && m.configColIdx < len(cols) {
					m.configAction = configActionRename
					input := textinput.New()
					input.SetValue(cols[m.configColIdx].DisplayName)
					input.CharLimit = 80
					input.Focus()
					m.configColumnInput = input
					m.Status = ""
					return m, nil
				}
			}
		case "d":
			if m.configField == 2 {
				cols := m.Config.GetColumns()
				if m.configColIdx >= 0 && m.configColIdx < len(cols) {
					if m.configColIdx == 0 {
						m.Status = "The first column cannot be deleted"
						return m, nil
					}
					m.configAction = configActionDeleteConfirm
					m.Status = fmt.Sprintf("Delete %s? y/n", cols[m.configColIdx].DisplayName)
					return m, nil
				}
			}
		case "K":
			if m.configField == 2 {
				return m.reorderSelectedColumn(-1)
			}
		case "J":
			if m.configField == 2 {
				return m.reorderSelectedColumn(1)
			}
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

func (m Model) focusConfigField() (tea.Model, tea.Cmd) {
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		cmd := m.configEditorInput.Focus()
		return m, cmd
	}
	m.configEditorInput.Blur()
	return m, nil
}

func (m *Model) moveConfigColumnSelection(delta int) {
	cols := m.Config.GetColumns()
	if len(cols) == 0 {
		m.configColIdx = 0
		return
	}
	m.configColIdx = clamp(m.configColIdx+delta, 0, len(cols)-1)
}

func (m Model) updateConfigAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.configAction {
	case configActionAddName:
		switch msg.String() {
		case "esc":
			m.cancelConfigAction()
			return m, nil
		case "enter":
			val := strings.TrimSpace(m.configColumnInput.Value())
			if val == "" {
				m.Status = "Column name cannot be empty"
				return m, nil
			}
			if err := store.AddColumn(m.Root, val); err != nil {
				m.Status = err.Error()
				return m, nil
			}
			m.cancelConfigAction()
			m.syncConfigAndOrder()
			m.configColIdx = len(m.Config.GetColumns()) - 1
			return m, m.notify("Column added", notifSuccess)
		default:
			var cmd tea.Cmd
			m.configColumnInput, cmd = m.configColumnInput.Update(msg)
			return m, cmd
		}
	case configActionRename:
		switch msg.String() {
		case "esc":
			m.cancelConfigAction()
			return m, nil
		case "enter":
			val := strings.TrimSpace(m.configColumnInput.Value())
			if val == "" {
				m.Status = "Column name cannot be empty"
				return m, nil
			}
			cols := m.Config.GetColumns()
			if m.configColIdx < 0 || m.configColIdx >= len(cols) {
				m.Status = "No column selected"
				return m, nil
			}
			if err := store.RenameColumn(m.Root, cols[m.configColIdx].ID, val); err != nil {
				m.Status = err.Error()
				return m, nil
			}
			m.cancelConfigAction()
			m.syncConfigAndOrder()
			return m, m.notify("Column renamed", notifSuccess)
		default:
			var cmd tea.Cmd
			m.configColumnInput, cmd = m.configColumnInput.Update(msg)
			return m, cmd
		}
	case configActionDeleteConfirm:
		switch msg.String() {
		case "y", "Y":
			cols := m.Config.GetColumns()
			if m.configColIdx < 0 || m.configColIdx >= len(cols) {
				m.Status = "No column selected"
				return m, nil
			}
			if err := store.DeleteColumn(m.Root, cols[m.configColIdx].ID); err != nil {
				m.Status = err.Error()
				return m, nil
			}
			m.cancelConfigAction()
			m.configColIdx = clamp(m.configColIdx, 0, len(cols)-2)
			m.syncConfigAndOrder()
			return m, m.notify("Column deleted", notifSuccess)
		case "n", "N", "esc":
			m.cancelConfigAction()
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) cancelConfigAction() {
	m.configAction = configActionNone
	m.configColumnInput = textinput.Model{}
	m.Status = ""
}

func (m *Model) syncConfigAndOrder() {
	if !m.loadAndResortBoard() {
		return
	}
	cols := m.Config.GetColumns()
	m.columnOrder = statesFromColumns(cols)
	m.configColIdx = clamp(m.configColIdx, 0, len(cols)-1)
	m.SelectedCol = clamp(m.SelectedCol, 0, len(m.columnOrder)-1)
	m.ColScrollOffset = clamp(m.ColScrollOffset, 0, max(0, len(m.columnOrder)-m.visibleColumnCount()))
}

func (m Model) reorderSelectedColumn(delta int) (tea.Model, tea.Cmd) {
	cols := m.Config.GetColumns()
	if len(cols) == 0 {
		return m, nil
	}
	newIdx := m.configColIdx + delta
	if newIdx < 0 || newIdx >= len(cols) {
		return m, nil
	}

	newOrder := make([]string, len(cols))
	for i, col := range cols {
		newOrder[i] = col.ID
	}
	newOrder[m.configColIdx], newOrder[newIdx] = newOrder[newIdx], newOrder[m.configColIdx]

	if err := store.ReorderColumns(m.Root, newOrder); err != nil {
		m.Status = err.Error()
		return m, nil
	}

	m.configColIdx = newIdx
	m.syncConfigAndOrder()
	return m, m.notify("Columns reordered", notifSuccess)
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
	if formWidth > 88 {
		formWidth = 88
	}
	if formWidth < 52 {
		formWidth = 52
	}

	labelW := 9
	labelStyle := mutedStyle.Width(labelW)
	activeStyle := func(active bool) lipgloss.Style {
		if active {
			return selectedStyle.Width(labelW)
		}
		return labelStyle
	}

	editorRow := m.renderConfigEditorRow(activeStyle(m.configField == 0))
	themeRow := m.renderConfigThemeRow(activeStyle(m.configField == 1))
	columnsBlock := m.renderConfigColumnsBlock(activeStyle(m.configField == 2), formWidth-labelW)

	var rows []string
	rows = append(rows, editorRow)
	if m.configField == 0 && m.configEditorIdx == len(editorPresets) {
		inputWidth := formWidth - labelW - 6
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.configEditorInput.Width = inputWidth
		rows = append(rows, "", labelStyle.Render("")+m.configEditorInput.View())
	}

	rows = append(rows, "", themeRow, "", columnsBlock)
	rows = append(rows, "", mutedStyle.Render("tab field  h/l select  j/k row  a add  r rename  K/J reorder  d delete  enter save  esc cancel"))
	content := strings.Join(rows, "\n")

	box := dialogBoxStyle(formWidth, 0).Render(content)

	var statusLine string
	if m.Status != "" {
		statusLine = "\n" + mutedStyle.Render(m.Status)
	}

	return m.placeDialog("Config", box, statusLine, m.safeHeight(28))
}

func (m Model) renderConfigEditorRow(editorLabel lipgloss.Style) string {
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
	return editorLabel.Render("Editor") + strings.Join(editorParts, "  ")
}

func (m Model) renderConfigThemeRow(themeLabel lipgloss.Style) string {
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
	return themeLabel.Render("Theme") + strings.Join(themeParts, "  ")
}

func (m Model) renderConfigColumnsBlock(columnsLabel lipgloss.Style, width int) string {
	cols := m.Config.GetColumns()
	var lines []string
	lines = append(lines, columnsLabel.Render("Columns")+mutedStyle.Render("#  Name                  Folder ID             Actions"))
	for i, col := range cols {
		marker := " "
		if m.configField == 2 && i == m.configColIdx {
			marker = ">"
		}
		actions := "a r K/J"
		if i == 0 {
			actions += " -"
		} else {
			actions += " d"
		}
		line := fmt.Sprintf("%s %d  %-20s  %-20s  %s", marker, i+1, fitText(col.DisplayName, 20), fitText(col.ID, 20), actions)
		if m.configField == 2 && i == m.configColIdx {
			line = selectedStyle.Render(line)
		} else {
			line = mutedStyle.Render(line)
		}
		lines = append(lines, strings.Repeat(" ", 9)+line)
	}

	if m.configAction == configActionAddName || m.configAction == configActionRename {
		inputWidth := width - 8
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.configColumnInput.Width = inputWidth
		prompt := "Name"
		if m.configAction == configActionAddName {
			prompt = "Add"
		}
		lines = append(lines, strings.Repeat(" ", 9)+prompt+": "+m.configColumnInput.View())
	} else if m.configAction == configActionDeleteConfirm {
		lines = append(lines, strings.Repeat(" ", 9)+selectedStyle.Render("Confirm delete? y/n"))
	}
	return strings.Join(lines, "\n")
}
