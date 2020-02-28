package model

import "context"

type ConnectionType int8

const (
	ConnectionTypeCall ConnectionType = iota
	ConnectionTypeGrpc
)

type Server interface {
	Name() string
	Start() *AppError
	Stop()
	Host() string
	Port() int
	Consume() <-chan Connection
	Type() ConnectionType
	GetApplication(string) (*Application, *AppError)
}

type Variables map[string]interface{}

type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string

	Execute(context.Context, string, interface{}) (Response, *AppError)
	Get(key string) (string, bool)
	Set(vars Variables) (Response, *AppError)
	ParseText(text string) string

	Close() *AppError
}
