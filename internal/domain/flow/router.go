package flow

// moved from model/router.go — see model/router.go for re-export aliases

// ApplicationRequest describes a single step in a flow schema.
type ApplicationRequest interface {
	Id() string
	Args() interface{}
}
