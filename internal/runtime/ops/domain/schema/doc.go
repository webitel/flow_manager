package schema

import "github.com/webitel/flow_manager/internal/runtime/ops"

func (s *schemaOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Calls another Webitel flow (schema) by ID. " +
			"Useful for reusable sub-flows shared across multiple flows " +
			"(e.g. dynamic amount calculation, notification logic). " +
			"With async: false (default) — waits for the sub-flow to finish and merges its variables back. " +
			"With async: true — fires and forgets; variables are independent.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"id": {
				Type:        "integer",
				Required:    true,
				Description: "Schema ID to execute.",
			},
			"async": {
				Type:        "boolean",
				Default:     false,
				Description: "If true — runs the sub-schema in background without waiting.",
			},
		},
		Notes: []string{
			"Sync mode (async: false) supports suspendable ops inside the sub-flow (e.g. recvMessage, softSleep).",
			"Triggers declared in the calling schema are inherited by the sub-schema; sub-schema triggers take priority on name collision.",
			"Async mode does not support suspension — the goroutine exits on the first suspend point.",
		},
		Examples: map[string]ops.Example{
			"sync_by_id": {
				Description: "Call sub-flow synchronously by ID",
				Schema:      `{"schema": {"id": 165, "async": false}}`,
			},
			"async_by_id": {
				Description: "Fire sub-flow asynchronously by ID",
				Schema:      `{"schema": {"id": 165, "async": true}}`,
			},
		},
	}
}
