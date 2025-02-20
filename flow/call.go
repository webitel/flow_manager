package flow

import (
	eng "buf.build/gen/go/webitel/engine/protocolbuffers/go"
	"context"
	"github.com/webitel/flow_manager/model"
)

type MakeCallArgs struct {
	eng.CreateCallRequest
}

func (r *router) makeCall(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv MakeCallArgs
	var err *model.AppError

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	argv.DomainId = conn.DomainId()

	if argv.From == nil || argv.From.Id == 0 {
		return nil, ErrorRequiredParameter("makeCall", "from")
		// err
	}
	if argv.Destination == "" {
		return nil, ErrorRequiredParameter("makeCall", "destination")
		// err
	}
	err = r.fm.MakeCall(ctx, &argv.CreateCallRequest)
	if err != nil {
		return model.CallResponseError, err
	}
	return model.CallResponseOK, nil
}
