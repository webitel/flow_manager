package flow

// moved from model/connection.go — see model/connection.go for re-export aliases

import (
	"context"
	"encoding/json"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
)

// ConnectionType identifies the transport type of a flow connection.
type ConnectionType int8

const (
	ConnectionTypeCall ConnectionType = iota
	ConnectionTypeGrpc
	ConnectionTypeEmail
	ConnectionTypeWebHook
	ConnectionTypeChat
	ConnectionTypeForm
	ConnectionTypeChannel
	ConnectionTypeIM
)

// Result wraps an application response together with an optional error.
type Result struct {
	Err error
	Res Response
}

// ResultChannel is a channel of Result values.
type ResultChannel chan Result

// ChannelExec carries the parameters for a cross-domain flow execution.
type ChannelExec struct {
	SchemaId  int                        `json:"schema_id"`
	DomainId  int64                      `json:"domain_id"`
	Variables map[string]json.RawMessage `json:"variables"`
}

// Server is the interface implemented by each transport provider.
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
type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string
	DomainId() int64

	Context() context.Context
	Get(key string) (string, bool)
	Set(ctx context.Context, vars Variables) (Response, error)
	ParseText(text string, ops ...ParseOption) string

	Close() error
	Variables() map[string]string
	Log() *wlog.Logger
}
