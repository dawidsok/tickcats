package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dawidsok/tickcats/internal/ticket"
)

func Create(boardRoot string, kind ticket.Kind, title string, labels []string, priority ticket.Priority, now time.Time, ac ...string) (string, error) {
	if err := Init(boardRoot); err != nil {
		return "", fmt.Errorf("init board: %w", err)
	}

	content := ticket.NewMarkdownWithLabels(kind, title, labels, priority, now, ac...)
	name := ticketFilename(now, title)
	path := filepath.Join(boardRoot, string(StateBacklog), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write ticket %q: %w", path, err)
	}
	return path, nil
}

func ticketFilename(now time.Time, title string) string {
	return now.UTC().Format("20060102-1504") + "-" + ticketSlug(title) + ".md"
}

var nonSlugCharsRe = regexp.MustCompile(`[^a-z0-9]+`)

func ticketSlug(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	s := nonSlugCharsRe.ReplaceAllString(lower, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "ticket"
	}
	return s
}
