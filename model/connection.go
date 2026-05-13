package model

import "encoding/json"

import "github.com/webitel/flow_manager/internal/domain/flow"

// Re-exports for backward compatibility.
type ConnectionType = flow.ConnectionType
type Result = flow.Result
type ResultChannel = flow.ResultChannel
type ChannelExec = flow.ChannelExec

const (
	ConnectionTypeCall    = flow.ConnectionTypeCall
	ConnectionTypeGrpc    = flow.ConnectionTypeGrpc
	ConnectionTypeEmail   = flow.ConnectionTypeEmail
	ConnectionTypeWebHook = flow.ConnectionTypeWebHook
	ConnectionTypeChat    = flow.ConnectionTypeChat
	ConnectionTypeForm    = flow.ConnectionTypeForm
	ConnectionTypeChannel = flow.ConnectionTypeChannel
	ConnectionTypeIM      = flow.ConnectionTypeIM
)

// Re-export interfaces for backward compatibility.
type Server = flow.Server
type Connection = flow.Connection

// VariablesToJson serialises a Variables map to JSON bytes.
// Replaces the former *Variables.ToJson() method (cannot define methods on aliased types).
func VariablesToJson(v *Variables) []byte {
	if v == nil {
		return nil
	}
	d, _ := json.Marshal(v)
	return d
}

// VariablesToString serialises a Variables map to a JSON string pointer.
// Replaces the former *Variables.ToString() method.
func VariablesToString(v *Variables) *string {
	if v == nil {
		return nil
	}
	d, _ := json.Marshal(v)
	return NewString(string(d))
}

func VariablesFromStringMap(m map[string]string) Variables {
	vars := make(Variables)
	for k, v := range m {
		vars[k] = v
	}
	return vars
}
