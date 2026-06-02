package store

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "lowercase", raw: "backlog", want: "backlog"},
		{name: "spaces", raw: "Code Review", want: "code-review"},
		{name: "special characters", raw: "Q&A Testing!", want: "qa-testing"},
		{name: "apostrophe", raw: "Won't Do", want: "wont-do"},
		{name: "trim hyphens", raw: "--ready--", want: "ready"},
		{name: "empty", raw: "!!!", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slugify(tt.raw); got != tt.want {
				t.Fatalf("slugify(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolveColumn(t *testing.T) {
	cfg := Config{Columns: []Column{
		{ID: "backlog", DisplayName: "Backlog"},
		{ID: "ready", DisplayName: "Ready"},
		{ID: "code-review", DisplayName: "Code Review"},
		{ID: "wont-do", DisplayName: "Won't Do"},
	}}

	tests := []struct {
		name string
		raw  string
		want State
		ok   bool
	}{
		{name: "exact ID", raw: "code-review", want: State("code-review"), ok: true},
		{name: "ID case insensitive", raw: "CODE-REVIEW", want: State("code-review"), ok: true},
		{name: "display name case insensitive", raw: "code review", want: State("code-review"), ok: true},
		{name: "slug compatible display", raw: "Code Review", want: State("code-review"), ok: true},
		{name: "apostrophe slug", raw: "wont do", want: StateWontDo, ok: true},
		{name: "invalid", raw: "unknown-column", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveColumn(cfg, tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("ResolveColumn() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("ResolveColumn() expected error")
			}
			if got != tt.want {
				t.Fatalf("ResolveColumn() = %q, want %q", got, tt.want)
			}
		})
	}
}
