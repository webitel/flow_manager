package builtin

import "github.com/webitel/flow_manager/internal/runtime/ops"

// expand is a convenience wrapper for ops.ExpandStr without global-variable support.
// Used by the set op which has no domain context.
func expand(s string, vars map[string]string) string {
	return ops.ExpandStr(s, vars, nil)
}
