// render_detail.go renders the full-screen detail view for a single ticket.
// The view is split into two side-by-side panels: a scrollable content panel
// on the left showing the ticket body, and a fixed metadata panel on the right
// showing frontmatter fields (ID, state, priority, dates, labels, deadline).
package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
)

func (m Model) renderDetail() string {
	stored := m.findDetailTicket()
	if stored == nil {
		msg := "No ticket selected"
		if m.detailTicketName != "" {
			msg = "Ticket not found: " + m.detailTicketName
		}
		return msg + "\n\n" + mutedStyle.Render("esc back  q quit") + "\n"
	}

	contentWidth, metadataWidth := m.detailWidths()
	contentInnerWidth := max(1, contentWidth-2)
	metadataInnerWidth := max(1, metadataWidth-2)
	lines := m.detailDisplayLines(contentInnerWidth)
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
		Render(strings.Join(wrapLines(strings.Split(m.renderDetailMetadata(*stored), "\n"), metadataInnerWidth), "\n"))

	var b strings.Builder
	b.WriteString(bannerStyle.Render(fitText(stored.Ticket.Title, m.fullWidth())))
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
	fmt.Fprintf(&b, "ID: %s\n", displayTicketID(stored.Ticket.ID))
	fmt.Fprintf(&b, "Title: %s\n", stored.Ticket.Title)
	b.WriteString("State: " + m.colStyle(m.stateColIndex(stored.State)).Render(string(stored.State)) + "\n")
	b.WriteString("Priority: " + priorityStyle(stored.Ticket.Priority).Render(string(stored.Ticket.Priority)) + "\n")
	fmt.Fprintf(&b, "File: %s\n", stored.Name)
	if len(stored.Ticket.ParsedTitle.Labels) > 0 {
		fmt.Fprintf(&b, "Labels: %s\n", strings.Join(stored.Ticket.ParsedTitle.Labels, ", "))
	}
	if stored.Ticket.Deadline != nil {
		fmt.Fprintf(&b, "Deadline: %s\n", stored.Ticket.Deadline.Format(time.DateOnly))
	} else {
		b.WriteString("Deadline: —\n")
	}
	fmt.Fprintf(&b, "Created: %s\n", stored.Ticket.Created.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "Updated: %s", stored.Ticket.Updated.Format("2006-01-02 15:04"))
	return b.String()
}

func displayTicketID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "—"
	}
	return id
}

func (m Model) detailWidths() (int, int) {
	// lipgloss Width includes padding but excludes borders and margins.
	// The detail row renders two bordered panels, with a right margin on the
	// content panel, so reserve those five cells before splitting the remaining
	// width between content and metadata.
	available := m.fullWidth() - 5
	if available < 2 {
		available = 2
	}

	metadata := available / 3
	if available >= 38 && metadata < 18 {
		metadata = 18
	}
	if available >= 38 && available-metadata < 20 {
		metadata = available - 20
	}
	metadata = clamp(metadata, 1, available-1)
	content := available - metadata
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
	stored := m.findDetailTicket()
	if stored == nil {
		return nil
	}
	body := strings.TrimRight(stored.Ticket.Body, "\n")
	if body == "" {
		return []string{"empty body"}
	}
	return strings.Split(body, "\n")
}

func (m Model) detailDisplayLines(width int) []string {
	plainLines := m.detailLines()
	if len(plainLines) == 0 {
		return nil
	}
	if len(plainLines) == 1 && plainLines[0] == "empty body" {
		return []string{mutedStyle.Render("empty body")}
	}

	lines := make([]string, 0, len(plainLines))
	inFence := false
	for _, line := range plainLines {
		trimmed := strings.TrimSpace(line)
		if isFenceLine(trimmed) {
			for _, wrapped := range wrapCodeDetailLine(line, width) {
				lines = append(lines, m.detailCodeStyle().Render(wrapped))
			}
			inFence = !inFence
			continue
		}
		if inFence {
			for _, wrapped := range wrapCodeDetailLine(line, width) {
				lines = append(lines, m.detailCodeStyle().Render(wrapped))
			}
			continue
		}
		for _, wrapped := range wrapDetailLine(line, width) {
			lines = append(lines, m.highlightDetailLine(wrapped))
		}
	}
	return lines
}

func isFenceLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

var (
	headingLineRe = regexp.MustCompile(`^\s{0,3}#{1,6}(\s|$)`)
	listLineRe    = regexp.MustCompile(`^(\s*)((?:[-*+])|(?:\d+[.)]))(\s+)`)
	hrLineRe      = regexp.MustCompile(`^\s{0,3}[-*_][\s\-*_]*$`)
)

func isHorizontalRule(line string) bool {
	if !hrLineRe.MatchString(line) {
		return false
	}
	trimmed := strings.ReplaceAll(strings.TrimSpace(line), " ", "")
	if len(trimmed) < 3 {
		return false
	}
	first := trimmed[0]
	if first != '-' && first != '*' && first != '_' {
		return false
	}
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != first {
			return false
		}
	}
	return true
}

func (m Model) highlightDetailLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	if headingLineRe.MatchString(line) {
		return m.detailHeadingStyle().Render(line)
	}
	if isHorizontalRule(line) {
		return mutedStyle.Render(line)
	}
	if strings.HasPrefix(trimmed, ">") {
		return m.detailQuoteStyle().Render(line)
	}
	if match := listLineRe.FindStringSubmatchIndex(line); match != nil {
		indent := line[match[2]:match[3]]
		marker := line[match[4]:match[5]]
		space := line[match[6]:match[7]]
		rest := line[match[7]:]
		return indent + m.detailMarkerStyle().Render(marker) + space + m.highlightInlineMarkdown(rest)
	}
	return m.highlightInlineMarkdown(line)
}

func (m Model) highlightInlineMarkdown(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); {
		switch line[i] {
		case '`':
			if end := strings.IndexByte(line[i+1:], '`'); end >= 0 {
				end += i + 1
				b.WriteString(m.detailCodeStyle().Render(line[i : end+1]))
				i = end + 1
				continue
			}
		case '[':
			if closeText := strings.Index(line[i:], "]("); closeText >= 0 {
				closeText += i
				if closeURL := strings.IndexByte(line[closeText+2:], ')'); closeURL >= 0 {
					closeURL += closeText + 2
					b.WriteString(m.detailLinkStyle().Render(line[i : closeURL+1]))
					i = closeURL + 1
					continue
				}
			}
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

func (m Model) detailHeadingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(m.themeColor(m.stateColIndex(store.StateReady)))
}

func (m Model) detailMarkerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(m.themeColor(m.stateColIndex(store.StateBacklog)))
}

func (m Model) detailCodeStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("236"))
}

func (m Model) detailLinkStyle() lipgloss.Style {
	return lipgloss.NewStyle().Underline(true).Foreground(m.themeColor(m.stateColIndex(store.StateDoing)))
}

func (m Model) detailQuoteStyle() lipgloss.Style {
	return lipgloss.NewStyle().Italic(true).Foreground(m.themeColor(m.stateColIndex(store.StateDone)))
}

func wrapDetailLine(line string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if line == "" {
		return []string{""}
	}
	indentLen := len(line) - len(strings.TrimLeft(line, " \t"))
	indent := line[:indentLen]
	words := strings.Fields(strings.TrimSpace(line))
	if len(words) == 0 {
		return []string{line}
	}

	lines := make([]string, 0, 1)
	current := indent
	for _, word := range words {
		for lipgloss.Width(word) > width {
			part, rest := splitToWidth(word, width)
			if strings.TrimSpace(current) != "" {
				lines = append(lines, current)
				current = indent
			}
			lines = append(lines, part)
			word = rest
		}
		if strings.TrimSpace(current) == "" {
			current = indent + word
			continue
		}
		candidate := current + " " + word
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = indent + word
	}
	if current != indent || line == indent {
		lines = append(lines, current)
	}
	return lines
}

func wrapCodeDetailLine(line string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if line == "" {
		return []string{""}
	}
	lines := make([]string, 0, 1)
	for lipgloss.Width(line) > width {
		part, rest := splitToWidth(line, width)
		lines = append(lines, part)
		line = rest
	}
	lines = append(lines, line)
	return lines
}
