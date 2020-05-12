package model

type ApplicationRequest interface {
	Id() string
	Args() interface{}
}

type ParseText func(conn Connection, text string) string

type Router interface {
	Handle(conn Connection) *AppError
}
