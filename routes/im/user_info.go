package im

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type IMUserInfoRequest struct {
	Set  string                `json:"set"`
	Type IMUserInfoRequestType `json:"type"`
}

func (r *IMUserInfoRequest) Validate() *model.AppError {
	if r == nil {
		return model.NewRequestError("im.user_info.validate.nil_pointer", "received nil pointer call for IM user info request")
	}

	switch r.Type {
	case IMUserInfoRequestTypeDevice, IMUserInfoRequestTypeGate:
	default:
		return model.NewRequestError("im.user_info.validate.unsupported_info_type", "received unsupported request info type: "+string(r.Type))
	}

	return nil
}

func (r *Router) IMUserInfo(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv IMUserInfoRequest
	if err := r.Decode(scope, args, &argv); err != nil {
		return model.CallResponseError, err
	}

	if err := argv.Validate(); err != nil {
		return model.CallResponseError, err
	}

	result, err := r.userInfoHandlersFabric.Handle(ctx, argv.Type, conv)
	if err != nil {
		return model.CallResponseError, err
	}

	conv.Set(ctx, model.Variables{argv.Set: result})

	return model.CallResponseOK, nil
}
