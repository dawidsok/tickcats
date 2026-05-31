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
)

var ValidStates = []State{StateBacklog, StateReady, StateDoing, StateDone}

func ParseState(raw string) (State, error) {
	state := State(strings.ToLower(strings.TrimSpace(raw)))
	for _, valid := range ValidStates {
		if state == valid {
			return state, nil
		}
	}
	return "", fmt.Errorf("invalid state %q", raw)
}

func IsValidState(raw string) bool {
	_, err := ParseState(raw)
	return err == nil
}

