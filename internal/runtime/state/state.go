package state

// Status is the lifecycle state of an execution record.
type Status string

const (
	StatusRunning   Status = "running"
	StatusSuspended Status = "suspended"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Frame is one entry in the call stack. NodeID identifies the container node
// in the parsed Tree whose children are being iterated; Position is the index
// of the next child to execute.
type Frame struct {
	NodeID   string `json:"node_id"`
	Position int    `json:"position"`
}

// ExecState is the complete serializable execution state. A snapshot at any
// suspension boundary can fully resume the flow from where it stopped.
//
// All durable data must live in Variables — JS VM state is NOT persisted (the
// VM is rebuilt from Variables on every resume).
type ExecState struct {
	SchemaID      int               `json:"schema_id"`
	SchemaVersion uint64            `json:"schema_version"` // hash of normalized schema
	Stack         []Frame           `json:"stack"`          // call stack, top = last element
	Variables     map[string]string `json:"variables"`      // only durable state
	Tags          map[string]string `json:"tags"`           // tag name → NodeID (cached at parse time)
	GotoCounter   int16             `json:"goto_counter"`
	Status        Status            `json:"status"`
	Pending       *PendingIntent    `json:"pending,omitempty"`
	// Timezone is the IANA timezone name set by the "timezone" op (e.g. "Europe/Kyiv").
	// Empty means UTC / system default.
	Timezone string `json:"timezone,omitempty"`
}

// NewExecState returns an initial running state for the given schema, ready to
// start from the root frame.
func NewExecState(schemaID int, schemaVersion uint64, tags map[string]string) ExecState {
	return ExecState{
		SchemaID:      schemaID,
		SchemaVersion: schemaVersion,
		Stack:         []Frame{{NodeID: "root", Position: 0}},
		Variables:     make(map[string]string),
		Tags:          tags,
		Status:        StatusRunning,
	}
}
