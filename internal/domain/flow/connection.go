package flow

// moved from model/connection.go — see model/connection.go for re-export aliases

import "encoding/json"

// ConnectionType identifies the transport type of a flow connection.
type ConnectionType int8

const (
	ConnectionTypeCall    ConnectionType = iota
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
