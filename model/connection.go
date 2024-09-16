package model

import (
	"context"
	"encoding/json"
	"github.com/webitel/wlog"

	"github.com/webitel/engine/discovery"
)

type ConnectionType int8

const (
	ConnectionTypeCall ConnectionType = iota
	ConnectionTypeGrpc
	ConnectionTypeEmail
	ConnectionTypeWebHook
	ConnectionTypeChat
	ConnectionTypeForm
	ConnectionTypeChannel
)

type Server interface {
	Name() string
	Start() *AppError
	Stop()
	Host() string
	Port() int
	Consume() <-chan Connection
	Type() ConnectionType
	Cluster(discovery discovery.ServiceDiscovery) *AppError
}

type Variables map[string]interface{}

type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string
	DomainId() int64

	Context() context.Context
	Get(key string) (string, bool)
	Set(ctx context.Context, vars Variables) (Response, *AppError)
	ParseText(text string) string

	Close() *AppError
	Variables() map[string]string
	Log() *wlog.Logger
}

type Result struct {
	Err *AppError
	Res Response
}

type ResultChannel chan Result

type ChannelExec struct {
	SchemaId  int                        `json:"schema_id"`
	DomainId  int64                      `json:"domain_id"`
	Variables map[string]json.RawMessage `json:"variables"`
}

func (v *Variables) ToJson() []byte {
	if v == nil {
		return nil
	}
	d, _ := json.Marshal(v)
	return d
}

func (v *Variables) ToString() *string {
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
