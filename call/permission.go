package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

//grantee_id

type GranteeArgs struct {
	Id int64 `json:"id"`
}

func (r *Router) SetGrantee(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv GranteeArgs
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id < 1 {
		return nil, ErrorRequiredParameter("SetGrantee", "id")
	}

	err := r.fm.SetCallGranteeId(call.DomainId(), call.Id(), argv.Id)
	if err != nil {
		return nil, err
	}

	return call.Set(ctx, model.Variables{
		model.GranteeHeader: argv.Id,
	})
}
