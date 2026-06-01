package tui

import (
	"path/filepath"
	"testing"
)

func TestEditorCommandUsesEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cmd := editorCommand("ticket.md", "")
	if filepath.Base(cmd.Args[0]) != "nano" {
		t.Fatalf("Args[0] = %q, want nano", cmd.Args[0])
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "ticket.md" {
		t.Fatalf("Args = %#v", cmd.Args)
	}
}

func TestEditorCommandFallback(t *testing.T) {
	t.Setenv("EDITOR", "")
	cmd := editorCommand("ticket.md", "")
	if filepath.Base(cmd.Args[0]) != "vi" {
		t.Fatalf("Args[0] = %q, want vi", cmd.Args[0])
	}
}

func TestEditorCommandPreferredOverridesEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	cmd := editorCommand("ticket.md", "nvim")
	if filepath.Base(cmd.Args[0]) != "nvim" {
		t.Fatalf("Args[0] = %q, want nvim", cmd.Args[0])
	}
}
