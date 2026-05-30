package main

import (
	"strings"
	"testing"
)

func TestParseNewKind(t *testing.T) {
	tests := []struct {
		raw string
		ok  bool
	}{
		{raw: "feat", ok: true},
		{raw: "feature", ok: true},
		{raw: "task", ok: true},
		{raw: "bug", ok: true},
		{raw: "fix", ok: true},
		{raw: "chore", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := parseNewKind(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("parseNewKind() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("parseNewKind() expected error")
			}
		})
	}
}

func TestSplitTitleAndAcceptance(t *testing.T) {
	title, acceptance := splitTitleAndAcceptance([]string{"write", "README", "--ac", "README", "explains", "usage"})
	if got := strings.Join(title, " "); got != "write README" {
		t.Fatalf("title = %q, want write README", got)
	}
	if acceptance != "README explains usage" {
		t.Fatalf("acceptance = %q, want README explains usage", acceptance)
	}
}

func TestSlug(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "Add Import Validation", want: "add-import-validation"},
		{raw: "  crash on empty backlog!!! ", want: "crash-on-empty-backlog"},
		{raw: "!!!", want: "ticket"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := slug(tt.raw)
			if got != tt.want {
				t.Fatalf("slug() = %q, want %q", got, tt.want)
			}
		})
	}
}
