package ticket

import (
	"fmt"
	"strings"
)

type Priority string

const (
	PriorityP0 Priority = "P0"
	PriorityP1 Priority = "P1"
	PriorityP2 Priority = "P2"
	PriorityP3 Priority = "P3"
)

func ParsePriority(raw string) (Priority, error) {
	priority := Priority(strings.ToUpper(strings.TrimSpace(raw)))
	switch priority {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3:
		return priority, nil
	default:
		return "", fmt.Errorf("invalid priority %q", raw)
	}
}

func (p Priority) Rank() int {
	switch p {
	case PriorityP0:
		return 0
	case PriorityP1:
		return 1
	case PriorityP2:
		return 2
	case PriorityP3:
		return 3
	default:
		return 99
	}
}

func (p Priority) HigherThan(other Priority) bool {
	return p.Rank() < other.Rank()
}
