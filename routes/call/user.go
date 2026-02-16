package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type SetUser struct {
	Id int64
}

func (r *Router) SetUser(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (res model.Response, err *model.AppError) {
	var argv SetUser

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == 0 {
		return nil, ErrorRequiredParameter("SetUser", "id")
	}

	if err = r.fm.SetCallUserId(call.DomainId(), call.Id(), argv.Id); err != nil {
		return nil, err
	}

	return
}
