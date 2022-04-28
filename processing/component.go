package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ComponentArgs map[string]interface{}

func (r *Router) component(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ComponentArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
