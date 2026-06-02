// util.go provides stateless helper functions used across the tui package:
// integer arithmetic (min/max/clamp), terminal-safe text wrapping and
// truncation, safe terminal-dimension accessors (safeHeight/safeWidth), and a
// generic option-list renderer used by the create and config forms.
package tui

import (
	"fmt"
	"math"
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

func (m Model) stateColIndex(state store.State) int {
	for i, s := range m.columnOrder {
		if s == state {
			return i
		}
	}
	return 0
}

func (m Model) themeColor(colIndex int) lipgloss.Color {
	return themeColor(m.Config.Theme, colIndex, len(m.columnOrder))
}

func (m Model) colStyle(colIndex int) lipgloss.Style {
	return colStyle(m.Config.Theme, colIndex, len(m.columnOrder))
}

func themeColor(themeIdx int, colIndex int, totalColumns int) lipgloss.Color {
	idx := clamp(themeIdx, 0, len(colorThemes)-1)
	theme := colorThemes[idx]
	if totalColumns < 1 {
		totalColumns = 1
	}
	colIndex = clamp(colIndex, 0, totalColumns-1)

	if colIndex == 0 {
		return theme.backlogColor
	}
	if totalColumns == 2 {
		return theme.endColor
	}

	t := float64(colIndex-1) / float64(totalColumns-2)
	return lipgloss.Color(lerpHexColor(string(theme.startColor), string(theme.endColor), t))
}

func colStyle(themeIdx int, colIndex int, totalColumns int) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(themeColor(themeIdx, colIndex, totalColumns))
}

func lerpHexColor(startHex string, endHex string, t float64) string {
	t = maxFloat(0, minFloat(1, t))
	sr, sg, sb := parseHex(startHex)
	er, eg, eb := parseHex(endHex)

	sh, ss, sl := rgbToHSL(sr, sg, sb)
	eh, es, el := rgbToHSL(er, eg, eb)

	dh := eh - sh
	if dh > 180 {
		dh -= 360
	} else if dh < -180 {
		dh += 360
	}
	h := sh + dh*t
	if h < 0 {
		h += 360
	} else if h >= 360 {
		h -= 360
	}

	s := ss + (es-ss)*t
	l := sl + (el-sl)*t
	r, g, b := hslToRGB(h, s, l)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func parseHex(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		return parseHexDigit(hex[0]) * 17, parseHexDigit(hex[1]) * 17, parseHexDigit(hex[2]) * 17
	}
	if len(hex) == 6 {
		return parseHexDigit(hex[0])*16 + parseHexDigit(hex[1]),
			parseHexDigit(hex[2])*16 + parseHexDigit(hex[3]),
			parseHexDigit(hex[4])*16 + parseHexDigit(hex[5])
	}
	return 128, 128, 128
}

func parseHexDigit(d byte) int {
	switch {
	case d >= '0' && d <= '9':
		return int(d - '0')
	case d >= 'a' && d <= 'f':
		return int(d-'a') + 10
	case d >= 'A' && d <= 'F':
		return int(d-'A') + 10
	default:
		return 0
	}
}

func rgbToHSL(r int, g int, b int) (float64, float64, float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255
	maxVal := maxFloat(rf, gf, bf)
	minVal := minFloat(rf, gf, bf)
	l := (maxVal + minVal) / 2
	if maxVal == minVal {
		return 0, 0, l
	}

	d := maxVal - minVal
	s := d / (1 - absFloat(2*l-1))
	var h float64
	switch maxVal {
	case rf:
		h = 60 * ((gf - bf) / d)
		if h < 0 {
			h += 360
		}
	case gf:
		h = 60 * ((bf-rf)/d + 2)
	case bf:
		h = 60 * ((rf-gf)/d + 4)
	}
	return h, s, l
}

func hslToRGB(h float64, s float64, l float64) (int, int, int) {
	c := (1 - absFloat(2*l-1)) * s
	x := c * (1 - absFloat(math.Mod(h/60, 2)-1))
	m := l - c/2

	var rp, gp, bp float64
	switch {
	case h < 60:
		rp, gp, bp = c, x, 0
	case h < 120:
		rp, gp, bp = x, c, 0
	case h < 180:
		rp, gp, bp = 0, c, x
	case h < 240:
		rp, gp, bp = 0, x, c
	case h < 300:
		rp, gp, bp = x, 0, c
	default:
		rp, gp, bp = c, 0, x
	}
	return int((rp+m)*255 + 0.5), int((gp+m)*255 + 0.5), int((bp+m)*255 + 0.5)
}

func maxFloat(values ...float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minFloat(values ...float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
