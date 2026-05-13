package flow

// moved from model/router.go — see model/router.go for re-export aliases
// NOTE: Router interface is NOT moved here because Handle() returns *model.AppError
//       which would create an import cycle. It remains defined in model/router.go.

// ApplicationRequest describes a single step in a flow schema.
type ApplicationRequest interface {
	Id() string
	Args() interface{}
}
