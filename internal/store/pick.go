// pick.go implements the "pick next" algorithm that identifies which Ready
// ticket should be worked on next. A ticket is eligible only when it has a
// non-empty title, an "## Acceptance Criteria" section with real content, and
// none of the "[blocked]" or "[to refine]" labels.
//
// Eligible tickets are ranked by priority (P0 first) then creation date
// (oldest first), then filename as a stable tiebreaker. When two or more
// tickets share the same priority and creation timestamp exactly, NeedsChoice
// is set to true so the caller can present the tie candidates to the user.
package store

import (
	"sort"

	"github.com/dawidsok/tickcats/internal/ticket"
)

// PickResult is the outcome of PickNext. HasPick indicates whether any eligible
// ticket was found. NeedsChoice is true when multiple tickets are tied at the
// top rank and the user must choose between them.
type PickResult struct {
	Ticket      StoredTicket
	Tied        []StoredTicket
	HasPick     bool
	NeedsChoice bool
}

func PickNext(board Board) PickResult {
	candidates := eligibleTickets(board.Columns[StateReady])
	if len(candidates) == 0 {
		return PickResult{}
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]

		if left.Ticket.Priority != right.Ticket.Priority {
			return left.Ticket.Priority.HigherThan(right.Ticket.Priority)
		}
		if !left.Ticket.Created.Equal(right.Ticket.Created) {
			return left.Ticket.Created.Before(right.Ticket.Created)
		}
		return left.Name < right.Name
	})

	best := candidates[0]
	tied := []StoredTicket{best}
	for _, candidate := range candidates[1:] {
		if samePickRank(best.Ticket, candidate.Ticket) {
			tied = append(tied, candidate)
			continue
		}
		break
	}

	return PickResult{
		Ticket:      best,
		Tied:        tied,
		HasPick:     true,
		NeedsChoice: len(tied) > 1,
	}
}

// IsReadyForPick reports whether a ticket meets all eligibility criteria for
// the pick-next algorithm: in the Ready column, non-empty title, acceptance
// criteria present, not blocked, not marked "to refine".
func IsReadyForPick(stored StoredTicket) bool {
	parsedTitle := stored.Ticket.ParsedTitle
	return stored.State == StateReady &&
		stored.Ticket.Title != "" &&
		stored.Ticket.HasAcceptanceCriteria &&
		!parsedTitle.Blocked() &&
		!parsedTitle.ToRefine()
}

func eligibleTickets(tickets []StoredTicket) []StoredTicket {
	eligible := make([]StoredTicket, 0, len(tickets))
	for _, stored := range tickets {
		if IsReadyForPick(stored) {
			eligible = append(eligible, stored)
		}
	}
	return eligible
}

func samePickRank(left ticket.Ticket, right ticket.Ticket) bool {
	return left.Priority == right.Priority && left.Created.Equal(right.Created)
}
