package call

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UpdateCid struct {
	Name        string `json:"name"`
	Number      string `json:"number"`
	Destination string `json:"destination"`
}

func (r *Router) UpdateCid(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (res model.Response, err *model.AppError) {
	var argv UpdateCid
	var name, number, destination *string

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Name == "" && argv.Number == "" && argv.Destination == "" {
		return nil, ErrorRequiredParameter("UpdateCid", "name and number")
	}

	if argv.Number != "" {
		number = &argv.Number
	}

	if argv.Name != "" {
		name = &argv.Name
	}

	if argv.Destination != "" {
		destination = &argv.Destination
	}

	if res, err = call.UpdateCid(ctx, name, number, destination); err != nil {
		return nil, err
	}

	if err = r.fm.UpdateCallFrom(call.Id(), name, number, destination); err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
