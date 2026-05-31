package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dawidsok/tickcats/internal/ticket"
)

func TestTicketSlug(t *testing.T) {
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
			got := ticketSlug(tt.raw)
			if got != tt.want {
				t.Fatalf("ticketSlug() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateWritesTicketInBacklog(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	path, err := Create(boardRoot, ticket.KindFeature, "My New Feature", nil, ticket.PriorityP2, now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !strings.HasPrefix(filepath.Dir(path), filepath.Join(boardRoot, string(StateBacklog))) {
		t.Fatalf("path not in backlog: %q", path)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Fatalf("path not .md: %q", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if !strings.Contains(string(data), "Feat: My New Feature") {
		t.Fatalf("content missing title: %s", data)
	}
	if !strings.Contains(string(data), "P2") {
		t.Fatalf("content missing priority: %s", data)
	}
}

func TestCreateFilenameContainsSlug(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	path, err := Create(boardRoot, ticket.KindTask, "Fix the Bug!", nil, ticket.PriorityP1, now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	name := filepath.Base(path)
	if !strings.Contains(name, "fix-the-bug") {
		t.Fatalf("filename %q does not contain slug", name)
	}
}

func TestCreateWithToRefineLabel(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	path, err := Create(boardRoot, ticket.KindFeature, "Needs Design", []string{ticket.LabelToRefine}, ticket.PriorityP2, now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ticket: %v", err)
	}
	if !strings.Contains(string(data), "[to refine]") {
		t.Fatalf("content missing [to refine] label: %s", data)
	}
}

func TestCreateInitsBoard(t *testing.T) {
	boardRoot := filepath.Join(t.TempDir(), ".tickcats")
	now := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)

	if _, err := Create(boardRoot, ticket.KindBug, "init test", nil, ticket.PriorityP3, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for _, state := range ValidStates {
		info, err := os.Stat(filepath.Join(boardRoot, string(state)))
		if err != nil || !info.IsDir() {
			t.Fatalf("state dir %q missing after Create", state)
		}
	}
}
