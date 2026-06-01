package store

import (
	"fmt"
	"strings"
)

const RootDir = ".tickcats"

type State string

const (
	StateBacklog State = "backlog"
	StateReady   State = "ready"
	StateDoing   State = "doing"
	StateDone    State = "done"
	StateWontDo  State = "wont-do"
)

var ValidStates = []State{StateBacklog, StateReady, StateDoing, StateDone, StateWontDo}

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
		return string(s)
	}
}

func IsValidState(raw string) bool {
	_, err := ParseState(raw)
	return err == nil
}
