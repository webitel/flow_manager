package model

import (
	"context"
	"github.com/webitel/engine/discovery"
)

type ConnectionType int8

const (
	ConnectionTypeCall ConnectionType = iota
	ConnectionTypeGrpc
	ConnectionTypeEmail
	ConnectionTypeWebChat
	ConnectionTypeChat
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
}

type Result struct {
	Err *AppError
	Res Response
}

type ResultChannel chan Result
