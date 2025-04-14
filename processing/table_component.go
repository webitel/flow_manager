package processing

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) formTable(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.FormTable

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter("tableComponent", "name")
	}

	conn.SetComponent(argv.Id, argv)

	return model.CallResponseOK, nil
}
