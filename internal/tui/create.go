package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

func (m Model) enterCreate() (tea.Model, tea.Cmd) {
	input := textinput.New()
	input.Placeholder = "ticket title"
	input.CharLimit = 200
	m.createInput = input
	m.createKind = ticket.KindFeature
	m.createPriority = ticket.PriorityP2
	m.createToRefine = true
	m.createField = 1
	m.Mode = ViewCreate
	m.Status = ""
	cmd := m.createInput.Focus()
	return m, cmd
}

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.Mode = ViewBoard
			m.Status = ""
			return m, nil
		case "tab":
			m.createField = (m.createField + 1) % 4
			return m, m.syncCreateFocus()
		case "shift+tab":
			m.createField = (m.createField + 3) % 4
			return m, m.syncCreateFocus()
		case "h", "left":
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 0 {
				m.cycleKind(-1)
			} else if m.createField == 2 {
				m.cyclePriority(-1)
			}
			return m, nil
		case "l", "right":
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 0 {
				m.cycleKind(1)
			} else if m.createField == 2 {
				m.cyclePriority(1)
			}
			return m, nil
		case "enter":
			if strings.TrimSpace(m.createInput.Value()) == "" {
				m.Status = "Title required"
				return m, nil
			}
			return m.submitCreate()
		default:
			if m.createField == 1 {
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(keyMsg)
				return m, cmd
			}
			if m.createField == 3 && keyMsg.String() == " " {
				m.createToRefine = !m.createToRefine
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.createInput, cmd = m.createInput.Update(msg)
	return m, cmd
}

func (m *Model) syncCreateFocus() tea.Cmd {
	if m.createField == 1 {
		return m.createInput.Focus()
	}
	m.createInput.Blur()
	return nil
}

func (m *Model) cycleKind(delta int) {
	for i, k := range createKinds {
		if k == m.createKind {
			m.createKind = createKinds[(i+len(createKinds)+delta)%len(createKinds)]
			return
		}
	}
	m.createKind = createKinds[0]
}

func (m *Model) cyclePriority(delta int) {
	for i, p := range createPriorities {
		if p == m.createPriority {
			m.createPriority = createPriorities[(i+len(createPriorities)+delta)%len(createPriorities)]
			return
		}
	}
	m.createPriority = ticket.PriorityP2
}

func (m Model) submitCreate() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.createInput.Value())
	var labels []string
	if m.createToRefine {
		labels = []string{ticket.LabelToRefine}
	}
	path, err := store.Create(m.Root, m.createKind, title, labels, m.createPriority, time.Now().UTC())
	if err != nil {
		m.Status = "Create failed: " + err.Error()
		return m, nil
	}
	board, err := store.LoadBoard(m.Root)
	if err != nil {
		m.Status = "Reload failed: " + err.Error()
		return m, nil
	}
	m.Board = board
	m.syncManualOrder()
	m.applySortToBoard()
	m.Mode = ViewBoard
	m.createPending = path
	if m.Config.SkipEditorPrompt {
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(path), notifSuccess)
	}
	m.InteractionMode = InteractionPostCreate
	m.Status = ""
	return m, nil
}

func (m Model) updatePostCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y":
		cmd := editorCommand(m.createPending, m.Config.Editor)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		m.InteractionMode = InteractionBoard
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	case "n", "esc":
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(m.createPending), notifSuccess)
	case "d":
		m.Config.SkipEditorPrompt = true
		_ = store.SaveConfig(m.Root, m.Config)
		m.InteractionMode = InteractionBoard
		return m, m.notify("Created "+filepath.Base(m.createPending)+" (won't ask again)", notifSuccess)
	}
	return m, nil
}

func (m Model) renderCreate() string {
	formWidth := m.fullWidth() - 4
	if formWidth > 60 {
		formWidth = 60
	}
	if formWidth < 44 {
		formWidth = 44
	}

	labelW := 9
	labelStyle := mutedStyle.Width(labelW)
	activeLabel := selectedStyle.Width(labelW)

	kindLabel := labelStyle.Render("Kind")
	titleLabel := labelStyle.Render("Title")
	priorityLabel := labelStyle.Render("Priority")
	refineLabel := labelStyle.Render("To Refine")
	switch m.createField {
	case 0:
		kindLabel = activeLabel.Render("Kind")
	case 1:
		titleLabel = activeLabel.Render("Title")
	case 2:
		priorityLabel = activeLabel.Render("Priority")
	case 3:
		refineLabel = activeLabel.Render("To Refine")
	}

	inputWidth := formWidth - labelW - 6
	if inputWidth < 10 {
		inputWidth = 10
	}
	m.createInput.Width = inputWidth

	checkbox := "[ ]"
	if m.createToRefine {
		checkbox = selectedStyle.Render("[x]")
	}

	kindRow := kindLabel + m.renderKindOptions()
	titleRow := titleLabel + m.createInput.View()
	priorityRow := priorityLabel + m.renderPriorityOptions()
	refineRow := refineLabel + checkbox
	helpRow := mutedStyle.Render("tab/shift-tab field  h/l change  space toggle  enter create  esc cancel")

	content := strings.Join([]string{kindRow, "", titleRow, "", priorityRow, "", refineRow, "", helpRow}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(formWidth).
		Render(content)

	var statusLine string
	if m.Status != "" {
		statusLine = "\n" + mutedStyle.Render(m.Status)
	}

	h := m.Height
	if h <= 0 {
		h = 24
	}
	return lipgloss.Place(m.fullWidth(), h, lipgloss.Center, lipgloss.Center,
		selectedStyle.Render("New Ticket")+"\n\n"+box+statusLine)
}

func (m Model) renderKindOptions() string {
	parts := make([]string, 0, len(createKinds))
	for _, k := range createKinds {
		name := string(k)
		if k == m.createKind {
			if m.createField == 0 {
				parts = append(parts, selectedStyle.Render("["+name+"]"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			parts = append(parts, mutedStyle.Render(name))
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderPriorityOptions() string {
	parts := make([]string, 0, len(createPriorities))
	for _, p := range createPriorities {
		name := string(p)
		if p == m.createPriority {
			if m.createField == 2 {
				parts = append(parts, selectedStyle.Render("["+name+"]"))
			} else {
				parts = append(parts, lipgloss.NewStyle().Bold(true).Render("["+name+"]"))
			}
		} else {
			parts = append(parts, mutedStyle.Render(name))
		}
	}
	return strings.Join(parts, "  ")
}
