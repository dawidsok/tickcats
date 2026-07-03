package tui

import (
	"time"

	"github.com/dawidsok/tickcats/internal/store"
)

func matrixTicketLess(left, right store.StoredTicket, now time.Time) bool {
	li, ri := matrixBucket(left, now), matrixBucket(right, now)
	if li != ri {
		return li < ri
	}
	return deadlinePriorityCreatedNameLess(left, right)
}

func matrixBucket(stored store.StoredTicket, now time.Time) int {
	urgent := ticketUrgent(stored, now)
	important := stored.Ticket.Important
	switch {
	case urgent && important:
		return 0
	case important:
		return 1
	case urgent:
		return 2
	default:
		return 3
	}
}

func ticketUrgent(stored store.StoredTicket, now time.Time) bool {
	return stored.Ticket.Deadline != nil && daysUntil(*stored.Ticket.Deadline, now) <= 7
}

func deadlinePriorityCreatedNameLess(left, right store.StoredTicket) bool {
	if c := compareDeadlines(left, right); c != 0 {
		return c < 0
	}
	li, ri := left.Ticket.Priority.Rank(), right.Ticket.Priority.Rank()
	if li != ri {
		return li < ri
	}
	if !left.Ticket.Created.Equal(right.Ticket.Created) {
		return left.Ticket.Created.Before(right.Ticket.Created)
	}
	return left.Name < right.Name
}

func compareDeadlines(left, right store.StoredTicket) int {
	ld, rd := left.Ticket.Deadline, right.Ticket.Deadline
	switch {
	case ld == nil && rd == nil:
		return 0
	case ld == nil:
		return 1
	case rd == nil:
		return -1
	}
	leftDate := dateOnly(ld.UTC())
	rightDate := dateOnly(rd.UTC())
	if leftDate.Before(rightDate) {
		return -1
	}
	if leftDate.After(rightDate) {
		return 1
	}
	return 0
}
