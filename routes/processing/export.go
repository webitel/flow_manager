package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ExportArg []string

func (r *Router) export(ctx context.Context, scope *flow.Flow, conn Connection, args any) (model.Response, *model.AppError) {
	var argv ExportArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	conn.Export(ctx, argv)

	return model.CallResponseOK, nil
}
