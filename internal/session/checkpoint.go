package session

import (
	"time"

	"github.com/webitel/flow_manager/model"
)

type Status string

const (
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

// Checkpoint holds the state of a long-lived flow session that must survive process restarts.
// Short-lived channels (call, webhook, processing) are excluded — only stateful channels are tracked.
type Checkpoint struct {
	ID           string
	ConnectionID string
	DomainID     int64
	Channel      model.ConnectionType
	SchemaID     int
	AppID        string // node/instance that owns this checkpoint
	Variables    map[string]string
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

// IsStateful reports whether a connection type requires session recovery.
func IsStateful(t model.ConnectionType) bool {
	switch t {
	case model.ConnectionTypeChat, model.ConnectionTypeIM, model.ConnectionTypeEmail:
		return true
	}
	return false
}

func New(conn model.Connection, schemaID int, appID string) *Checkpoint {
	now := time.Now().UTC()
	return &Checkpoint{
		ConnectionID: conn.Id(),
		DomainID:     conn.DomainId(),
		Channel:      conn.Type(),
		SchemaID:     schemaID,
		AppID:        appID,
		Variables:    conn.Variables(),
		Status:       StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (c *Checkpoint) Refresh(vars map[string]string) {
	c.Variables = vars
	c.UpdatedAt = time.Now().UTC()
}

func (c *Checkpoint) Close() {
	now := time.Now().UTC()
	c.Status = StatusClosed
	c.ClosedAt = &now
	c.UpdatedAt = now
}
