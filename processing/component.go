package processing

import (
	"context"
	"github.com/webitel/flow_manager/pkg/processing"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) formComponent(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv processing.FormComponent

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter("formComponent", "name")
	}

	val, _ := conn.Get(argv.Id)
	if argv.View.Component == "wt-input" { // TODO DEV-5230
		argv.Value = val
	} else {
		argv.Value = setToJson(val)
	}

	conn.SetComponent(argv.Id, argv)

	return model.CallResponseOK, nil
}
