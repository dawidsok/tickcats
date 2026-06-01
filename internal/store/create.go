// create.go handles new ticket creation. It initialises the board if needed,
// generates a unique ID, builds the markdown content, and writes the file into
// the backlog directory. Filename format is "<id-lowercase>-<slug>.md".
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

// Create writes a new ticket file into the backlog directory and returns its
// absolute path. ac is an optional acceptance-criteria string included in the
// generated markdown template.
func Create(boardRoot string, kind ticket.Kind, title string, labels []string, priority ticket.Priority, now time.Time, ac ...string) (string, error) {
	if err := Init(boardRoot); err != nil {
		return "", fmt.Errorf("init board: %w", err)
	}

	board, err := LoadBoard(boardRoot)
	if err != nil {
		return "", fmt.Errorf("load board ids: %w", err)
	}
	id, err := ticket.GenerateID(existingTicketIDs(board))
	if err != nil {
		return "", err
	}

	fullTitle := ticket.ParsedTitle{Kind: kind, Text: strings.TrimSpace(title), Labels: labels, HadPrefix: true}.NormalizedTitle()
	content := ticket.NewMarkdownFullWithID(id, fullTitle, priority, now, ac...)
	name := ticketFilename(id, title)
	path := filepath.Join(boardRoot, string(StateBacklog), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write ticket %q: %w", path, err)
	}
	return path, nil
}

func existingTicketIDs(board Board) map[string]bool {
	ids := make(map[string]bool)
	for _, tickets := range board.Columns {
		for _, stored := range tickets {
			if ticket.ValidID(stored.Ticket.ID) {
				ids[stored.Ticket.ID] = true
			}
		}
	}
	return ids
}

func ticketFilename(id string, title string) string {
	return strings.ToLower(id) + "-" + ticketSlug(title) + ".md"
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
