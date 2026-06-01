// util.go provides stateless helper functions used across the tui package:
// integer arithmetic (min/max/clamp), terminal-safe text wrapping and
// truncation, safe terminal-dimension accessors (safeHeight/safeWidth), and a
// generic option-list renderer used by the create and config forms.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dawidsok/tickcats/internal/store"
	"github.com/dawidsok/tickcats/internal/ticket"
)

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func wrapLines(lines []string, width int) []string {
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapText(line, width)...)
	}
	return wrapped
}

func wrapText(value string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	if value == "" {
		return []string{""}
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, 1)
	current := ""
	for _, word := range words {
		for lipgloss.Width(word) > width {
			part, rest := splitToWidth(word, width)
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, part)
			word = rest
		}
		if current == "" {
			current = word
			continue
		}
		candidate := current + " " + word
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func splitToWidth(value string, width int) (string, string) {
	runes := []rune(value)
	if width >= len(runes) {
		return value, ""
	}
	return string(runes[:width]), string(runes[width:])
}

func fitText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	plain := lipgloss.Width(value)
	if plain <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-1]) + "…"
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m Model) safeHeight(fallback int) int {
	if m.Height <= 0 {
		return fallback
	}
	return m.Height
}

func (m Model) safeWidth(fallback int) int {
	if m.Width <= 0 {
		return fallback
	}
	return m.Width
}

// renderSelectOptions renders a horizontal list of selectable string-based
// options. The selected option is highlighted (bold+colour when focused,
// bold-only when not), and unselected options are muted. Used by the Kind and
// Priority fields in the create form and by the editor/theme pickers in config.
func renderSelectOptions[T ~string](options []T, selected T, isFocused bool) string {
	parts := make([]string, 0, len(options))
	for _, opt := range options {
		name := string(opt)
		if opt == selected {
			if isFocused {
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

func priorityStyle(p ticket.Priority) lipgloss.Style {
	switch p {
	case ticket.PriorityP0:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff5f5f"))
	case ticket.PriorityP1:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffaf5f"))
	case ticket.PriorityP2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#d7d75f"))
	default:
		return mutedStyle
	}
}

func stateColIndex(state store.State) int {
	for i, s := range columnOrder {
		if s == state {
			return i
		}
	}
	return 0
}

func (m Model) colStyle(colIndex int) lipgloss.Style {
	themeIdx := clamp(m.Config.Theme, 0, len(colorThemes)-1)
	colors := colorThemes[themeIdx].colors
	color := colors[clamp(colIndex, 0, len(colors)-1)]
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}
