package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) enterDeadlineDialog() (tea.Model, tea.Cmd) {
	if m.selectedTicket() == nil && m.findDetailTicket() == nil {
		m.Status = "No ticket selected"
		return m, nil
	}
	m = m.enterInteraction(InteractionDeadline)
	m.deadlineCustom = false
	m.deadlineInput = textinput.Model{}
	return m, nil
}

func (m Model) updateDeadlineDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.deadlineCustom {
		switch msg.String() {
		case "esc":
			m.deadlineCustom = false
			m.deadlineInput = textinput.Model{}
			return m, nil
		case "enter":
			deadline, err := time.Parse(time.DateOnly, strings.TrimSpace(m.deadlineInput.Value()))
			if err != nil {
				m.Status = "Use YYYY-MM-DD"
				return m, nil
			}
			return m.applyDeadline(&deadline)
		default:
			var cmd tea.Cmd
			m.deadlineInput, cmd = m.deadlineInput.Update(msg)
			return m, cmd
		}
	}

	today := dateOnly(time.Now().UTC())
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "n":
		m = m.dismissInteraction()
		return m, nil
	case "t":
		return m.applyDeadline(&today)
	case "m":
		tomorrow := today.AddDate(0, 0, 1)
		return m.applyDeadline(&tomorrow)
	case "w":
		nextWeek := today.AddDate(0, 0, 7)
		return m.applyDeadline(&nextWeek)
	case "o":
		return m.applyDeadline(nil)
	case "c":
		input := textinput.New()
		input.Placeholder = "YYYY-MM-DD"
		input.CharLimit = len(time.DateOnly)
		input.Width = len(time.DateOnly)
		m.deadlineInput = input
		m.deadlineCustom = true
		return m, m.deadlineInput.Focus()
	}
	return m, nil
}

func (m Model) applyDeadline(deadline *time.Time) (tea.Model, tea.Cmd) {
	stored := m.selectedTicket()
	if m.Mode == ViewDetail {
		stored = m.findDetailTicket()
	}
	if stored == nil {
		m = m.dismissInteraction()
		m.Status = "No ticket selected"
		return m, nil
	}
	name := stored.Name
	state := stored.State
	if err := store.SetDeadline(m.Root, name, state, deadline); err != nil {
		m.Status = "Deadline failed: " + err.Error()
		return m, nil
	}
	m = m.dismissInteraction()
	m.deadlineCustom = false
	m.deadlineInput = textinput.Model{}
	m.reloadBoard()
	if m.Mode == ViewDetail {
		m.detailTicketName = name
		m.resolveDetailCursor()
	}
	if deadline == nil {
		return m, m.notify("Deadline cleared", notifSuccess)
	}
	return m, m.notify("Deadline: "+deadline.UTC().Format(time.DateOnly), notifSuccess)
}

func (m Model) renderDeadlineDialog() string {
	var content string
	if m.deadlineCustom {
		m.deadlineInput.Width = len(time.DateOnly)
		content = "Custom deadline\n\n" + m.deadlineInput.View() + "\n\n" + mutedStyle.Render("enter save  esc options")
	} else {
		content = strings.Join([]string{
			"Set deadline / SLA",
			"",
			selectedStyle.Render("t") + "  today",
			selectedStyle.Render("m") + "  tomorrow",
			selectedStyle.Render("w") + "  next week",
			selectedStyle.Render("c") + "  custom date input",
			selectedStyle.Render("o") + "  off / clear deadline",
		}, "\n")
	}
	box := dialogBoxStyle(36, 0).Render(content)
	footer := "\n" + mutedStyle.Render(fmt.Sprintf("D deadline  esc cancel"))
	return m.placeDialog("Deadline", box, footer, m.safeHeight(24))
}
