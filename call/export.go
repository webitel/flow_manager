package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ExportArg []string

func (r *Router) export(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv ExportArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}
	return call.Export(ctx, argv)
}
