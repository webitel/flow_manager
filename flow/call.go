package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

func (r *router) makeCall(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.OutboundCallRequest
	var err *model.AppError

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	argv.DomainId = conn.DomainId()

	if argv.From == nil || (argv.From.Id == 0 && argv.From.Extension == "") {
		return nil, ErrorRequiredParameter("makeCall", "from")
	}
	if argv.Destination == "" {
		return nil, ErrorRequiredParameter("makeCall", "destination")
	}
	err = r.fm.MakeCall(ctx, argv)
	if err != nil {
		return model.CallResponseError, err
	}
	return model.CallResponseOK, nil
}
