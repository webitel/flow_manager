package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) formComponent(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.FormComponent

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter("formComponent", "name")
	}

	argv.Value, _ = conn.Get(argv.Id)

	conn.SetComponent(argv.Id, &argv)

	return model.CallResponseOK, nil
}
