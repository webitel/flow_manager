package model

import (
	"encoding/json"
	"time"
)

// EmailEventState is the stage of an email-service event in the Inbox.
type EmailEventState string

const (
	EmailEventPending    EmailEventState = "pending"
	EmailEventProcessing EmailEventState = "processing"
	EmailEventAccepted   EmailEventState = "accepted"
	EmailEventFailed     EmailEventState = "failed"
)

// EmailEvent is a row of flow.email_event, the Inbox of email-service events.
type EmailEvent struct {
	EventId       string
	DomainId      int64
	EventType     string
	SchemaVersion int32
	ThreadId      int64
	MessageId     int64
	FlowId        *int64
	Payload       json.RawMessage
	State         EmailEventState
	Attempts      int32
	NextAttemptAt time.Time
	LockedBy      string
	LockedUntil   *time.Time
	CreatedAt     time.Time
}

// EmailEventAcceptResult tells how MarkAccepted ended for the worker that called it.
type EmailEventAcceptResult int

const (
	// EmailEventApplied means this call accepted the row.
	EmailEventApplied EmailEventAcceptResult = iota + 1
	// EmailEventAlreadyAccepted means the row was accepted before, e.g. by a commit
	// whose reply was lost.
	EmailEventAlreadyAccepted
	// EmailEventOwnershipLost means the row is held by another worker, back in
	// another state or gone.
	EmailEventOwnershipLost
)

func (r EmailEventAcceptResult) String() string {
	switch r {
	case EmailEventApplied:
		return "applied"
	case EmailEventAlreadyAccepted:
		return "already_accepted"
	case EmailEventOwnershipLost:
		return "ownership_lost"
	default:
		return "unknown"
	}
}
