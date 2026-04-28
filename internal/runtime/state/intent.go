package state

// PendingIntent records a side-effect that is about to happen (or has
// already started). It is persisted BEFORE the side effect executes so that
// on crash/resume the worker knows exactly which op was in flight and can
// check idempotency before retrying.
//
// On successful completion the Pending field is cleared.
type PendingIntent struct {
	OpName         string            `json:"op"`               // which op was executing
	NodeID         string            `json:"node_id"`          // where in the tree
	IdempotencyKey string            `json:"key"`              // unique per attempt
	Args           map[string]string `json:"args"`             // op-specific metadata
	ResumeKey      string            `json:"resume,omitempty"` // non-empty for async waits
	// VarFromPayload maps resume-payload keys to variable names.
	// On resume, driver writes payload[key] → variables[varName] for each entry.
	VarFromPayload map[string]string `json:"var_from_payload,omitempty"`
}
