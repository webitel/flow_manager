package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UpdateCid struct {
	Name   *string `json:"name"`
	Number *string `json:"number"`
}

func (r *Router) UpdateCid(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (res model.Response, err *model.AppError) {
	var argv UpdateCid

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Name == nil && argv.Number == nil {
		return nil, ErrorRequiredParameter("UpdateCid", "name and number")
	}

	if res, err = call.UpdateCid(ctx, argv.Name, argv.Number); err != nil {
		return nil, err
	}

	if err = r.fm.UpdateCallFrom(call.Id(), argv.Name, argv.Number); err != nil {
		return nil, err
	}

	return
}
