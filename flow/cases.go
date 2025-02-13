package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type GetCasesArgs struct {
	Contact struct {
		Id int64
	}
	SetVar string
}

func (r *router) getCases(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetCasesArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if argv.SetVar == "" {
		// todo
	}

	// run

	res := `{"id":"123123", "case": "name case"}`

	return conn.Set(ctx, model.Variables{
		argv.SetVar: res,
	})
}
