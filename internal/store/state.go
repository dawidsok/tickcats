// Package store is the data layer for TickCats. It owns all file system
// operations: creating, moving, and deleting ticket files, loading the full
// board into memory, and persisting configuration. Business logic that depends
// on board state (pick-next, ID migration) also lives here.
//
// state.go defines the State type for the five ticket columns and the
// input-normalisation logic that makes CLI state arguments case/punctuation
// insensitive (e.g. "wont-do", "WONT DO", and "won't-do" all parse to
// StateWontDo).
package store

import (
	"fmt"
	"strings"
)

// RootDir is the default board directory relative to the working directory.
const RootDir = ".tickcats"

// State is the name of a kanban column, stored as the subdirectory name on disk.
type State string

const (
	StateBacklog State = "backlog"
	StateReady   State = "ready"
	StateDoing   State = "doing"
	StateDone    State = "done"
	StateWontDo  State = "wont-do"
)

var ValidStates = []State{StateBacklog, StateReady, StateDoing, StateDone, StateWontDo}

// ParseState parses a user-supplied state string with normalisation: lowercase,
// trim, collapse whitespace to hyphens, strip smart-quotes.
func ParseState(raw string) (State, error) {
	state := State(normalizeStateInput(raw))
	for _, valid := range ValidStates {
		if state == valid {
			return state, nil
		}
	}
	return "", fmt.Errorf("invalid state %q", raw)
}

func normalizeStateInput(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "’", "'")
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")
	return normalized
}

func (s State) DisplayName() string {
	switch s {
	case StateBacklog:
		return "Backlog"
	case StateReady:
		return "Ready"
	case StateDoing:
		return "Doing"
	case StateDone:
		return "Done"
	case StateWontDo:
		return "Won't Do"
	default:
		return formatDisplayName(string(s))
	}
}

func IsValidState(raw string) bool {
	_, err := ParseState(raw)
	return err == nil
}
