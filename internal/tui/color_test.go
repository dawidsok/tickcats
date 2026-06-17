package tui

import "testing"

func TestThemeColorEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		colIndex     int
		totalColumns int
		want         string
	}{
		{name: "one column uses backlog", colIndex: 0, totalColumns: 1, want: "#88a"},
		{name: "two columns uses end", colIndex: 1, totalColumns: 2, want: "#88a"},
		{name: "five columns start", colIndex: 1, totalColumns: 5, want: "#ff88dd"},
		{name: "five columns end", colIndex: 4, totalColumns: 5, want: "#8888aa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := themeColor(0, tt.colIndex, tt.totalColumns)
			if string(got) != tt.want {
				t.Fatalf("themeColor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDimSumThemeUsesDimSumPalette(t *testing.T) {
	theme := colorThemes[len(colorThemes)-1]
	if theme.name != "dim-sum" || string(theme.backlogColor) != "#86837a" || string(theme.startColor) != "#b77a4a" || string(theme.endColor) != "#87965f" {
		t.Fatalf("dim-sum theme = %+v", theme)
	}
}

func TestThemeColorIsDeterministicForAllThemesAndColumnCounts(t *testing.T) {
	for themeIdx := range colorThemes {
		for totalColumns := 1; totalColumns <= 32; totalColumns++ {
			for col := range totalColumns {
				first := themeColor(themeIdx, col, totalColumns)
				for range 100 {
					if got := themeColor(themeIdx, col, totalColumns); got != first {
						t.Fatalf("themeColor(%d, %d, %d) was not deterministic: %q vs %q", themeIdx, col, totalColumns, got, first)
					}
				}
			}
		}
	}
}

func TestLerpHexColorHSLEndpoints(t *testing.T) {
	if got := lerpHexColor("#ff0000", "#0000ff", 0); got != "#ff0000" {
		t.Fatalf("start endpoint = %q, want #ff0000", got)
	}
	if got := lerpHexColor("#ff0000", "#0000ff", 1); got != "#0000ff" {
		t.Fatalf("end endpoint = %q, want #0000ff", got)
	}
}

func TestParseHexSupportsShortAndLongForms(t *testing.T) {
	r, g, b := parseHex("#f8d")
	if r != 255 || g != 136 || b != 221 {
		t.Fatalf("parseHex(#f8d) = (%d, %d, %d), want (255, 136, 221)", r, g, b)
	}

	r, g, b = parseHex("#5fd787")
	if r != 95 || g != 215 || b != 135 {
		t.Fatalf("parseHex(#5fd787) = (%d, %d, %d), want (95, 215, 135)", r, g, b)
	}
}
