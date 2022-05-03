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

	if argv.Name == "" {
		return nil, model.ErrorRequiredParameter("formComponent", "name")
	}

	conn.SetComponent(argv.Name, argv.View)

	return model.CallResponseOK, nil
}
