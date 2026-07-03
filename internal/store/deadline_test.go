package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dawidsok/tickcats/internal/ticket"
)

func TestSetDeadlineAddsChangesAndRemovesFrontmatter(t *testing.T) {
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
	deadline := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	if err := setDeadlineAt(root, "a.md", StateReady, &deadline, now); err != nil {
		t.Fatalf("setDeadlineAt: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if !strings.Contains(string(data), "deadline: 2026-06-15\n") || !strings.Contains(string(data), "Keep this body.") {
		t.Fatalf("deadline/body not preserved:\n%s", data)
	}
	parsed, err := ticket.ParseMarkdown(data)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if parsed.Deadline == nil || !parsed.Deadline.Equal(deadline) {
		t.Fatalf("Deadline = %v, want %s", parsed.Deadline, deadline)
	}
	if !parsed.Updated.Equal(now) {
		t.Fatalf("Updated = %s, want %s", parsed.Updated, now)
	}

	if err := setDeadlineAt(root, "a.md", StateReady, nil, now.Add(time.Hour)); err != nil {
		t.Fatalf("clear deadline: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if strings.Contains(string(data), "deadline:") {
		t.Fatalf("deadline frontmatter not removed:\n%s", data)
	}
}
