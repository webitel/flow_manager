package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type SipRedirectArgs []string

func (r *Router) SipRedirect(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv SipRedirectArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return call.Redirect(ctx, argv)
}
