package model

import (
	"context"
	"encoding/json"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/discovery"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

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

// Server is the interface implemented by each transport provider.
// Kept here (not aliased) because its Consume() method returns model.Connection
// which references *AppError — moving it would create an import cycle.
type Server interface {
	Name() string
	Start() error
	Stop()
	Host() string
	Port() int
	Consume() <-chan Connection
	Type() ConnectionType
	Cluster(discovery discovery.ServiceDiscovery) error
}

// Connection is the core runtime context passed through a flow execution.
// Kept here (not aliased) because Set() returns *AppError — moving it would
// create an import cycle until AppError is extracted (Phase 5.2).
type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string
	DomainId() int64

	Context() context.Context
	Get(key string) (string, bool)
	Set(ctx context.Context, vars Variables) (Response, *AppError)
	ParseText(text string, ops ...ParseOption) string

	Close() error
	Variables() map[string]string
	Log() *wlog.Logger
}

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
