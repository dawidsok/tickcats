package store

import (
	"fmt"
	"strings"
)

const columnColorDigits = "048adf"

// ColumnColorPalette returns the deterministic set of colors users may assign
// to columns. Colors use the compact #rgb form and only the digits 0,4,8,a,d,f.
func ColumnColorPalette() []string {
	palette := make([]string, 0, len(columnColorDigits)*len(columnColorDigits)*len(columnColorDigits))
	for _, r := range columnColorDigits {
		for _, g := range columnColorDigits {
			for _, b := range columnColorDigits {
				palette = append(palette, fmt.Sprintf("#%c%c%c", r, g, b))
			}
		}
	}
	return palette
}

// NormalizeColumnColor validates and normalizes a column color. The empty
// string is valid and means "no custom color" / theme fallback.
func NormalizeColumnColor(raw string) (string, error) {
	color := strings.ToLower(strings.TrimSpace(raw))
	if color == "" {
		return "", nil
	}
	if len(color) != 4 || color[0] != '#' {
		return "", fmt.Errorf("invalid column color %q: expected #rgb using only 0,4,8,a,d,f", raw)
	}
	for i := 1; i < len(color); i++ {
		if !strings.ContainsRune(columnColorDigits, rune(color[i])) {
			return "", fmt.Errorf("invalid column color %q: expected #rgb using only 0,4,8,a,d,f", raw)
		}
	}
	return color, nil
}

// IsColumnColor reports whether value is one of the allowed non-empty column
// colors. Use NormalizeColumnColor when the empty reset value should be allowed.
func IsColumnColor(value string) bool {
	color, err := NormalizeColumnColor(value)
	return err == nil && color != ""
}
