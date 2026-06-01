package ticket

import (
	"strings"
	"testing"
	"time"
)

func TestNewMarkdownGeneratesParseableTickets(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{name: "feature", kind: KindFeature, want: "Feat: add import validation"},
		{name: "task", kind: KindTask, want: "Task: add import validation"},
		{name: "bug", kind: KindBug, want: "Bug: add import validation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := NewMarkdown(tt.kind, "add import validation", PriorityP2, now)
			got, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}
			if got.Title != tt.want {
				t.Fatalf("Title = %q, want %q", got.Title, tt.want)
			}
			if got.Priority != PriorityP2 {
				t.Fatalf("Priority = %q, want %q", got.Priority, PriorityP2)
			}
			if !got.Created.Equal(now) {
				t.Fatalf("Created = %s, want %s", got.Created, now)
			}
			if got.HasAcceptanceCriteria {
				t.Fatalf("HasAcceptanceCriteria = true, want false for placeholder '-' only")
			}
		})
	}
}

func TestNewMarkdownWithAcceptanceCriteria(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	content := NewMarkdown(KindTask, "write README", PriorityP2, now, "README explains usage")
	got, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}
	if !got.HasAcceptanceCriteria {
		t.Fatalf("HasAcceptanceCriteria = false, want true")
	}
}

func TestNewMarkdownOmitsDeadlineByDefault(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	content := NewMarkdown(KindTask, "write README", PriorityP2, now)
	if strings.Contains(content, "deadline:") {
		t.Fatalf("NewMarkdown() included deadline by default:\n%s", content)
	}
	got, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}
	if got.Deadline != nil {
		t.Fatalf("Deadline = %v, want nil", got.Deadline)
	}
}

func TestParseMarkdownOptionalDeadline(t *testing.T) {
	content := strings.Replace(validTicketContent("Task: write README", "- done"), "updated: 2026-05-30T10:00:00Z\n", "updated: 2026-05-30T10:00:00Z\ndeadline: 2026-06-15\n", 1)
	got, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}
	if got.Deadline == nil {
		t.Fatal("Deadline = nil, want parsed date")
	}
	want := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if !got.Deadline.Equal(want) {
		t.Fatalf("Deadline = %s, want %s", got.Deadline, want)
	}
}

func TestParseMarkdownParsesTitleLabelsAndDefaultsKind(t *testing.T) {
	content := `---
title: [idea, to refine] write README
priority: P1
created: 2026-05-30T10:00:00Z
updated: 2026-05-30T10:00:00Z
---

## Context

Need docs.

## Acceptance Criteria
- README explains usage
`

	got, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}
	if got.ParsedTitle.Kind != KindTask {
		t.Fatalf("Kind = %q, want %q", got.ParsedTitle.Kind, KindTask)
	}
	if got.ParsedTitle.HadPrefix {
		t.Fatalf("HadPrefix = true, want false")
	}
	if !got.ParsedTitle.HasLabel("idea") || !got.ParsedTitle.ToRefine() {
		t.Fatalf("Labels = %#v, want idea + to refine", got.ParsedTitle.Labels)
	}
	if !got.HasAcceptanceCriteria {
		t.Fatalf("HasAcceptanceCriteria = false, want true")
	}
}

func TestParseMarkdownDetectsBlockedLabel(t *testing.T) {
	content := validTicketContent("[blocked] Bug: crash on empty backlog", "- no crash")
	got, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}
	if !got.ParsedTitle.Blocked() {
		t.Fatalf("Blocked() = false, want true")
	}
}

func TestParseMarkdownAcceptanceCriteriaEmpty(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "blank", body: ""},
		{name: "placeholder dash", body: "-"},
		{name: "whitespace dash", body: "-   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := validTicketContent("Task: write README", tt.body)
			got, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}
			if got.HasAcceptanceCriteria {
				t.Fatalf("HasAcceptanceCriteria = true, want false")
			}
		})
	}
}

func TestParseMarkdownMalformedFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "missing opening fence", content: "title: Task: x", wantErr: "missing frontmatter opening fence"},
		{name: "missing closing fence", content: "---\ntitle: Task: x\n", wantErr: "missing frontmatter closing fence"},
		{name: "invalid line", content: "---\ntitle\n---\n", wantErr: "invalid frontmatter line"},
		{name: "missing title", content: strings.Replace(validTicketContent("Task: x", "- done"), "title: Task: x\n", "", 1), wantErr: "missing required frontmatter field \"title\""},
		{name: "invalid priority", content: strings.Replace(validTicketContent("Task: x", "- done"), "priority: P2", "priority: high", 1), wantErr: "invalid priority"},
		{name: "invalid created", content: strings.Replace(validTicketContent("Task: x", "- done"), "created: 2026-05-30T10:00:00Z", "created: yesterday", 1), wantErr: "invalid created timestamp"},
		{name: "invalid deadline", content: strings.Replace(validTicketContent("Task: x", "- done"), "updated: 2026-05-30T10:00:00Z\n", "updated: 2026-05-30T10:00:00Z\ndeadline: soon\n", 1), wantErr: "invalid deadline date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMarkdown([]byte(tt.content))
			if err == nil {
				t.Fatalf("ParseMarkdown() expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err, tt.wantErr)
			}
		})
	}
}

func validTicketContent(title string, acceptance string) string {
	return `---
title: ` + title + `
priority: P2
created: 2026-05-30T10:00:00Z
updated: 2026-05-30T10:00:00Z
---

## Context

Context here.

## Acceptance Criteria
` + acceptance + `
`
}
