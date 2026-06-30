package im

import (
	"context"
	"errors"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type IMUserInfoRequest struct {
	Set string `json:"set"`
}

func (r *Router) IMUserInfo(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv IMUserInfoRequest
	if err := r.Decode(scope, args, &argv); err != nil {
		return model.CallResponseError, err
	}

	deviceID := conv.DeviceID()
	session, err := conv.GetAuthSession(ctx, deviceID)
	if err != nil {
		if errors.Is(err, model.ErrAuthSesionNotFound) {
			wlog.Warn("zero devices found for requested device id", wlog.String("device_id", deviceID), wlog.Err(err))
			conv.Set(ctx, model.Variables{argv.Set: "{}"})
			return model.CallResponseOK, nil
		}

		return model.CallResponseError, err
	}

	serialized, err := session.Serialize()
	if err != nil {
		return model.CallResponseError, err
	}

	conv.Set(ctx, model.Variables{argv.Set: serialized})

	return model.CallResponseOK, nil
}
