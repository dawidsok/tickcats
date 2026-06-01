package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dawidsok/tickcats/internal/ticket"
)

type IDMigrationResult struct {
	Migrated []IDMigration
}

type IDMigration struct {
	OldPath string
	NewPath string
	ID      string
}

func MigrateIDs(boardRoot string) (IDMigrationResult, error) {
	board, err := LoadBoard(boardRoot)
	if err != nil {
		return IDMigrationResult{}, err
	}
	if duplicateIDWarnings(board.Warnings) > 0 {
		return IDMigrationResult{}, fmt.Errorf("cannot migrate ids while duplicate ticket ids exist")
	}

	existing := existingTicketIDs(board)
	result := IDMigrationResult{Migrated: make([]IDMigration, 0)}
	for _, state := range ValidStates {
		for _, stored := range board.Columns[state] {
			if stored.Ticket.ID != "" {
				continue
			}
			id, err := ticket.GenerateID(existing)
			if err != nil {
				return result, err
			}
			existing[id] = true

			data, err := os.ReadFile(stored.Path)
			if err != nil {
				return result, fmt.Errorf("read ticket %q: %w", stored.Path, err)
			}
			updated, err := addIDToMarkdown(data, id)
			if err != nil {
				return result, fmt.Errorf("add id to ticket %q: %w", stored.Path, err)
			}

			newName := uniqueTicketFilename(boardRoot, state, ticketFilename(id, stored.Ticket.Title))
			newPath := filepath.Join(boardRoot, string(state), newName)
			if err := os.WriteFile(stored.Path, updated, 0o644); err != nil {
				return result, fmt.Errorf("write ticket %q: %w", stored.Path, err)
			}
			if newPath != stored.Path {
				if err := os.Rename(stored.Path, newPath); err != nil {
					return result, fmt.Errorf("rename ticket %q to %q: %w", stored.Path, newPath, err)
				}
			}
			result.Migrated = append(result.Migrated, IDMigration{OldPath: stored.Path, NewPath: newPath, ID: id})
		}
	}
	return result, nil
}

func duplicateIDWarnings(warnings []Warning) int {
	count := 0
	for _, warning := range warnings {
		if strings.Contains(warning.Err.Error(), "duplicate ticket id") {
			count++
		}
	}
	return count
}

func addIDToMarkdown(data []byte, id string) ([]byte, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, fmt.Errorf("missing frontmatter opening fence")
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			return nil, fmt.Errorf("frontmatter title field not found")
		}
		if strings.HasPrefix(line, "id:") {
			return data, nil
		}
		if strings.HasPrefix(line, "title:") {
			out := make([]string, 0, len(lines)+1)
			out = append(out, lines[:i+1]...)
			out = append(out, "id: "+id)
			out = append(out, lines[i+1:]...)
			return []byte(strings.Join(out, "\n")), nil
		}
	}
	return nil, fmt.Errorf("missing frontmatter closing fence")
}

func uniqueTicketFilename(boardRoot string, state State, preferred string) string {
	dir := filepath.Join(boardRoot, string(state))
	path := filepath.Join(dir, preferred)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return preferred
	}
	ext := filepath.Ext(preferred)
	base := strings.TrimSuffix(preferred, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}
