package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type ExecuteArgs struct {
	Name  string
	Async bool
}

func (r *router) execute(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ExecuteArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Name == "" {
		return nil, ErrorRequiredParameter("execute", "name")
	}

	fnScope, err := scope.FunctionScope(argv.Name)
	if err != nil {
		return nil, err
	} else {
		if argv.Async {
			go Route(ctx, fnScope, scope.handler)
		} else {
			Route(ctx, fnScope, scope.handler)
		}
	}

	if fnScope.IsCancel() {
		scope.SetCancel()
	}

	return ResponseOK, nil
}
