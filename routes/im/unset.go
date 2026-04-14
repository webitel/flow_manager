package im

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UnSetArg []string

func (r *Router) UnSet(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv UnSetArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if len(argv) == 0 {
		return nil, flow.ErrorRequiredParameter("UnSet", "unSet")
	}

	return conv.UnSet(ctx, argv)
}
