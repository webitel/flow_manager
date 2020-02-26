package model

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

type Connection interface {
	Type() ConnectionType
	Id() string
	NodeId() string

	Execute(string, interface{}) (Response, *AppError)
	Get(key string) (string, bool)
	Set(key, value string) (Response, *AppError)
	ParseText(text string) string

	Close() *AppError
}
