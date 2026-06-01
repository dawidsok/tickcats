// board.go provides LoadBoard, which scans every state directory and builds
// the in-memory Board representation, and Move, which atomically relocates a
// ticket file from one state directory to another.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dawidsok/tickcats/internal/ticket"
)

// Board holds the full in-memory state of a TickCats board.
// Columns maps each State to its sorted slice of tickets.
// Warnings records non-fatal parse errors encountered during LoadBoard so the
// TUI can surface them without blocking the rest of the board from loading.
type Board struct {
	Columns  map[State][]StoredTicket
	Warnings []Warning
}

// StoredTicket couples a parsed Ticket with the on-disk location details
// needed to move or delete it without re-scanning the board.
type StoredTicket struct {
	Path   string
	Name   string
	State  State
	Ticket ticket.Ticket
}

// Warning records a non-fatal error encountered while loading a ticket file.
type Warning struct {
	Path string
	Err  error
}

// LoadBoard scans every state directory under root and returns a Board with
// all valid tickets. Unreadable or unparseable files are added to Board.Warnings
// rather than causing a hard failure. Tickets with missing or duplicate IDs
// also produce warnings. Each column is sorted by filename for stable ordering.
func LoadBoard(root string) (Board, error) {
	board := Board{
		Columns:  make(map[State][]StoredTicket, len(ValidStates)),
		Warnings: make([]Warning, 0),
	}
	for _, state := range ValidStates {
		board.Columns[state] = []StoredTicket{}
	}

	ids := make(map[string]StoredTicket)
	for _, state := range ValidStates {
		dir := filepath.Join(root, string(state))
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Board{}, fmt.Errorf("read state directory %q: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: err})
				continue
			}

			parsed, err := ticket.ParseMarkdown(data)
			if err != nil {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: err})
				continue
			}

			stored := StoredTicket{
				Path:   path,
				Name:   entry.Name(),
				State:  state,
				Ticket: parsed,
			}
			board.Columns[state] = append(board.Columns[state], stored)
			if parsed.ID == "" {
				continue
			}
			if !ticket.ValidID(parsed.ID) {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: fmt.Errorf("invalid ticket id %q: expected TC-XXXXXX", parsed.ID)})
				continue
			}
			if first, exists := ids[parsed.ID]; exists {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: fmt.Errorf("duplicate ticket id %q also used by %s", parsed.ID, first.Path)})
				continue
			}
			ids[parsed.ID] = stored
		}

		sort.Slice(board.Columns[state], func(i, j int) bool {
			return board.Columns[state][i].Name < board.Columns[state][j].Name
		})
	}

	return board, nil
}

// Move relocates a ticket file from one state directory to another using an
// atomic rename. It validates the filename, parses the source file to confirm
// it is a valid ticket, and checks for conflicts before moving.
func Move(root string, name string, from State, to State) (string, error) {
	if _, err := ParseState(string(from)); err != nil {
		return "", err
	}
	if _, err := ParseState(string(to)); err != nil {
		return "", err
	}

	cleanName, err := validateTicketFilename(name)
	if err != nil {
		return "", err
	}

	source := filepath.Join(root, string(from), cleanName)
	targetDir := filepath.Join(root, string(to))
	target := filepath.Join(targetDir, cleanName)

	if _, _, err := readAndParseTicket(source); err != nil {
		return "", err
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create target directory %q: %w", targetDir, err)
	}
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("target ticket already exists %q", target)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("check target ticket %q: %w", target, err)
	}

	if err := os.Rename(source, target); err != nil {
		return "", fmt.Errorf("move ticket %q to %q: %w", source, target, err)
	}
	return target, nil
}
