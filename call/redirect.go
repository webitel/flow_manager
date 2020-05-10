package call

import (
	"github.com/webitel/flow_manager/model"
)

type SipRedirectArgs []string

func (r *Router) SipRedirect(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv SipRedirectArgs

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	return call.Redirect(argv)
}
