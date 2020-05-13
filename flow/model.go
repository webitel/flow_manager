package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type ApplicationHandlers map[string]*Application
type ApplicationHandler func(ctx context.Context, scope *Flow, args interface{}) model.ResultChannel
type ApplicationArgsParser func(c model.Connection, args ...interface{}) interface{}

type Application struct {
	AllowNoConnect bool
	Handler        ApplicationHandler
	ArgsParser     ApplicationArgsParser
}

func UnionApplicationMap(src ...ApplicationHandlers) ApplicationHandlers {
	res := make(ApplicationHandlers)
	for _, m := range src {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

type Router interface {
	model.Router
	Handlers() ApplicationHandlers
}
