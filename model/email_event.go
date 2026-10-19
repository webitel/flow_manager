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

// EmailRouteState is the route of an event whose Thread had no active attempt.
type EmailRouteState string

const (
	// EmailRouteUnset is a row written before routing existed or not woken up yet.
	EmailRouteUnset     EmailRouteState = ""
	EmailRouteNotNeeded EmailRouteState = "not_needed"
	// EmailRoutePending waits for the intent of the Flow until the deadline.
	EmailRoutePending EmailRouteState = "pending"
	// EmailRouteJoining joins the queue the Flow chose before the deadline.
	EmailRouteJoining EmailRouteState = "joining"
	// EmailRouteResolving joins the fallback queue after the deadline.
	EmailRouteResolving EmailRouteState = "resolving"
	EmailRouteRouted    EmailRouteState = "routed"
	// EmailRouteUnrouted ends without a queue; the reason says why.
	EmailRouteUnrouted EmailRouteState = "unrouted"
	// EmailRouteFailed ends after the retries of unexpected errors ran out.
	EmailRouteFailed EmailRouteState = "route_failed"
)

// Sources of a route queue.
const (
	EmailRouteSourceFlow     = "flow"
	EmailRouteSourceFallback = "fallback"
)

// EmailRoute is the route part of an Inbox row.
type EmailRoute struct {
	EventId         string
	DomainId        int64
	ThreadId        int64
	State           EmailRouteState
	Source          string
	QueueId         *int32
	FallbackQueueId *int32
	// DuePassed reports that the deadline of the Flow is over, by the database clock.
	DuePassed bool
	Attempts  int32
}

// EmailRouteResult is the outcome of a join that ends a route step: the new state, and
// for routed the queue call_center answered.
type EmailRouteResult struct {
	State     EmailRouteState
	QueueId   *int32
	Source    string
	Reason    string
	LastError string
}
