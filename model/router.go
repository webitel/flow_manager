package model

type ApplicationRequest interface {
	Id() string
	Args() interface{}
}

type Router interface {
	Handle(conn Connection) *AppError
	Request(con Connection, req ApplicationRequest) (Response, *AppError)
	Handlers() ApplicationHandlers
}

type GRPCRouter interface {
	Router
}
