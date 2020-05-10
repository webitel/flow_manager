package call

import "github.com/webitel/flow_manager/model"

type HangupArg string

func (r *Router) hangup(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv HangupArg

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	return call.Hangup(string(argv))
}
