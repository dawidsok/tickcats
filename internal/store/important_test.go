package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dawidsok/tickcats/internal/ticket"
)

func TestSetImportantAddsAndRemovesFrontmatter(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	path := filepath.Join(root, string(StateReady), "a.md")
	content := `---
title: Task: a
priority: P2
created: 2026-05-30T10:00:00Z
updated: 2026-05-30T10:00:00Z
deadline: 2026-06-15
---

## Context

Keep this body.

## Acceptance Criteria
- done
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ticket: %v", err)
	}
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	if err := setImportantAt(root, "a.md", StateReady, true, now); err != nil {
		t.Fatalf("setImportantAt true: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if !strings.Contains(string(data), "important: true\n") {
		t.Fatalf("missing important frontmatter:\n%s", data)
	}
	if !strings.Contains(string(data), "deadline: 2026-06-15\n") || !strings.Contains(string(data), "Keep this body.") {
		t.Fatalf("deadline/body not preserved:\n%s", data)
	}
	parsed, err := ticket.ParseMarkdown(data)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if !parsed.Important {
		t.Fatal("Important = false, want true")
	}
	if !parsed.Updated.Equal(now) {
		t.Fatalf("Updated = %s, want %s", parsed.Updated, now)
	}

	if err := setImportantAt(root, "a.md", StateReady, false, now.Add(time.Hour)); err != nil {
		t.Fatalf("setImportantAt false: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if strings.Contains(string(data), "important:") {
		t.Fatalf("important frontmatter not removed:\n%s", data)
	}
	parsed, err = ticket.ParseMarkdown(data)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if parsed.Important {
		t.Fatal("Important = true, want false")
	}
}

func TestSetImportantRejectsInvalidTicketName(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := SetImportant(root, "../a.md", StateReady, true); err == nil {
		t.Fatal("SetImportant expected invalid filename error")
	}
}
