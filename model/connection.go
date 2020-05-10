package model

type ConnectionType int8

const (
	ConnectionTypeCall ConnectionType = iota
	ConnectionTypeGrpc
	ConnectionTypeEmail
)

type Server interface {
	Name() string
	Start() *AppError
	Stop()
	Host() string
	Port() int
	Consume() <-chan Connection
	Type() ConnectionType
}

type Variables map[string]interface{}

type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string
	DomainId() int64

	Get(key string) (string, bool)
	Set(vars Variables) (Response, *AppError)
	ParseText(text string) string

	Close() *AppError
}
