// priority.go defines the Priority type and its numeric ordering.
// P0 is highest urgency (rank 0), P3 is lowest (rank 3).
// The Rank method provides a stable integer ordering used for sorting and
// pick-next comparisons.
package ticket

import (
	"fmt"
	"strings"
)

// Priority is the urgency level of a ticket, stored as "P0"–"P3" in frontmatter.
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

// Rank returns the numeric sort position (0 = most urgent). Unknown priorities
// return 99 so they sort to the bottom rather than panicking.
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
