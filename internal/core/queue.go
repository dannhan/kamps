package core

import (
	"sort"

	"github.com/ruyam/kamps/internal/model"
)

// Queue is the session-only export buffer. Decisions persist separately.
type Queue struct {
	pending map[string]model.Triage
}

// QueueItem pairs an assignment with the queued action.
type QueueItem struct {
	Assignment model.Assignment
	Action     model.Triage
}

// NewQueue returns an empty queue.
func NewQueue() *Queue {
	return &Queue{pending: map[string]model.Triage{}}
}

// Set queues one assignment.
func (q *Queue) Set(id string, action model.Triage) {
	if action == model.Undecided {
		delete(q.pending, id)
		return
	}
	q.pending[id] = action
}

// Has reports whether an assignment is queued.
func (q *Queue) Has(id string) bool {
	_, ok := q.pending[id]
	return ok
}

// Remove drops one assignment from the queue.
func (q *Queue) Remove(id string) { delete(q.pending, id) }

// Clear empties the queue.
func (q *Queue) Clear() { q.pending = map[string]model.Triage{} }

// Len reports how many decisions are queued.
func (q *Queue) Len() int { return len(q.pending) }

// Counts reports queued decisions by action.
func (q *Queue) Counts() (tracked, missed, ignored int) {
	for _, action := range q.pending {
		switch action {
		case model.Tracked:
			tracked++
		case model.Missed:
			missed++
		case model.Ignored:
			ignored++
		}
	}
	return
}

// Items resolves queued IDs against the current assignments.
func (q *Queue) Items(assignments []model.Assignment) []QueueItem {
	byID := make(map[string]model.Assignment, len(assignments))
	for _, assignment := range assignments {
		byID[assignment.ID] = assignment
	}
	items := make([]QueueItem, 0, len(q.pending))
	for id, action := range q.pending {
		if assignment, ok := byID[id]; ok {
			items = append(items, QueueItem{Assignment: assignment, Action: action})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Action != items[j].Action {
			return items[i].Action < items[j].Action
		}
		return items[i].Assignment.ID < items[j].Assignment.ID
	})
	return items
}
