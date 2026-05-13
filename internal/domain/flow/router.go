package flow

// moved from model/router.go — see model/router.go for re-export aliases

// ApplicationRequest describes a single step in a flow schema.
type ApplicationRequest interface {
	Id() string
	Args() interface{}
}

// Router dispatches a Connection through a domain-specific flow.
type Router interface {
	Handle(conn Connection) error
	GlobalVariable(domainId int64, name string) string
}
