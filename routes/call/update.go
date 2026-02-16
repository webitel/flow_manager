package call

import (
	"context"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UpdateCid struct {
	Name        string `json:"name"`
	Number      string `json:"number"`
	Destination string `json:"destination"`
}

type Update struct {
	Variables model.Variables `json:"variables"`
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

func (r *Router) updateCall(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (res model.Response, err *model.AppError) {
	var argv Update
	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if call.UserId() == 0 {
		return model.CallResponseError, model.NewRequestError("call.update", "this call is not an outbound")
	}

	if len(argv.Variables) != 0 {
		cp := make(model.Variables)
		for k, v := range argv.Variables {
			if strings.HasPrefix(k, "wbt_") {
				cp[k] = v
			} else {
				cp["usr_"+k] = v
			}
		}
		res, err = call.Set(ctx, cp)
		if err != nil {
			return res, err
		}
	}

	return call.Update(ctx)
}
