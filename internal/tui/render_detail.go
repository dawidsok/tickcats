package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) renderDetail() string {
	stored := m.selectedTicket()
	if stored == nil {
		return "No ticket selected\n\n" + mutedStyle.Render("esc back  q quit") + "\n"
	}

	contentWidth, metadataWidth := m.detailWidths()
	contentInnerWidth := contentWidth - 2
	lines := wrapLines(m.detailLines(), contentInnerWidth)
	visible, above, below := m.visibleDetailLines(lines)
	contentText := strings.Join(visible, "\n")
	if above > 0 {
		contentText = mutedStyle.Render(fitText(fmt.Sprintf("↑ %d lines above", above), contentInnerWidth)) + "\n" + contentText
	}
	if below > 0 {
		contentText += "\n" + mutedStyle.Render(fitText(fmt.Sprintf("↓ %d lines below", below), contentInnerWidth))
	}
	contentText = bannerStyle.Render("CONTENT") + "\n\n" + contentText
	content := lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.detailPanelHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginRight(1).
		Render(contentText)
	metadata := lipgloss.NewStyle().
		Width(metadataWidth).
		Height(m.detailPanelHeight()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(wrapLines(strings.Split(m.renderDetailMetadata(*stored), "\n"), metadataWidth-2), "\n"))

	var b strings.Builder
	b.WriteString(bannerStyle.Render(stored.Ticket.Title))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, content, metadata))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("j/k scroll  esc back  q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderDetailMetadata(stored store.StoredTicket) string {
	var b strings.Builder
	b.WriteString(bannerStyle.Render("METADATA"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("ID: %s\n", displayTicketID(stored.Ticket.ID)))
	b.WriteString(fmt.Sprintf("Title: %s\n", stored.Ticket.Title))
	b.WriteString("State: " + m.colStyle(stateColIndex(stored.State)).Render(string(stored.State)) + "\n")
	b.WriteString("Priority: " + priorityStyle(stored.Ticket.Priority).Render(string(stored.Ticket.Priority)) + "\n")
	b.WriteString(fmt.Sprintf("File: %s\n", stored.Name))
	if len(stored.Ticket.ParsedTitle.Labels) > 0 {
		b.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(stored.Ticket.ParsedTitle.Labels, ", ")))
	}
	if stored.Ticket.Deadline != nil {
		b.WriteString(fmt.Sprintf("Deadline: %s\n", stored.Ticket.Deadline.Format(time.DateOnly)))
	} else {
		b.WriteString("Deadline: —\n")
	}
	b.WriteString(fmt.Sprintf("Created: %s\n", stored.Ticket.Created.Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("Updated: %s", stored.Ticket.Updated.Format("2006-01-02 15:04")))
	return b.String()
}

func displayTicketID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "—"
	}
	return id
}

func (m Model) detailWidths() (int, int) {
	width := m.Width
	if width <= 0 {
		width = 96
	}
	metadata := width / 3
	content := width - metadata - 3
	if metadata < 24 {
		metadata = 24
	}
	if content < 40 {
		content = 40
	}
	return content, metadata
}

func (m Model) visibleDetailLines(lines []string) ([]string, int, int) {
	maxBodyLines := m.detailPanelHeight() - 4 // 2 for borders, 2 for "CONTENT\n" header
	if maxBodyLines < 3 {
		maxBodyLines = 3
	}
	start := clamp(m.DetailScroll, 0, max(0, len(lines)-1))
	visibleSlots := maxBodyLines
	if start > 0 {
		visibleSlots--
	}
	below := max(0, len(lines)-start-visibleSlots)
	if below > 0 {
		visibleSlots--
	}
	if visibleSlots < 1 {
		visibleSlots = 1
	}
	end := min(len(lines), start+visibleSlots)
	return lines[start:end], start, len(lines) - end
}

func (m Model) detailLines() []string {
	stored := m.selectedTicket()
	if stored == nil {
		return nil
	}
	body := strings.TrimRight(stored.Ticket.Body, "\n")
	if body == "" {
		return []string{mutedStyle.Render("empty body")}
	}
	return strings.Split(body, "\n")
}
