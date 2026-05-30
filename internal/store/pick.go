package store

import (
	"sort"

	"github.com/dawidsok/tickcats/internal/ticket"
)

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
