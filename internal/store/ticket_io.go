// ticket_io.go contains low-level file helpers shared by board, delete, and
// ids operations. Centralising these prevents the same validation and error
// message patterns from diverging across files.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dawidsok/tickcats/internal/ticket"
)

// validateTicketFilename checks that name is a plain filename (no directory
// component) ending in ".md". Returns the cleaned name on success.
func validateTicketFilename(name string) (string, error) {
	cleanName := filepath.Base(name)
	if cleanName != name {
		return "", fmt.Errorf("ticket name must be a file name, got %q", name)
	}
	if !strings.HasSuffix(cleanName, ".md") {
		return "", fmt.Errorf("ticket name must end with .md, got %q", name)
	}
	return cleanName, nil
}

// readAndParseTicket reads a ticket file and parses its markdown content.
// Returns the raw bytes (for callers that need to rewrite the file) alongside
// the parsed Ticket value.
func readAndParseTicket(path string) ([]byte, ticket.Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ticket.Ticket{}, fmt.Errorf("read source ticket %q: %w", path, err)
	}
	parsed, err := ticket.ParseMarkdown(data)
	if err != nil {
		return nil, ticket.Ticket{}, fmt.Errorf("parse source ticket %q: %w", path, err)
	}
	return data, parsed, nil
}
