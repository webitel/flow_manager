package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type SetVarArgs struct {
	SetVar string `json:"setVar"`
}

func (r *router) makeCall(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.OutboundCallRequest
	var err *model.AppError
	var set SetVarArgs
	var id string

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err = scope.Decode(args, &set); err != nil {
		return nil, err
	}

	argv.DomainId = conn.DomainId()

	if argv.From == nil || (argv.From.Id == 0 && argv.From.Extension == "") {
		return nil, ErrorRequiredParameter("makeCall", "from")
	}
	if argv.Destination == "" {
		return nil, ErrorRequiredParameter("makeCall", "destination")
	}

	id, err = r.fm.MakeCall(ctx, argv)
	if err != nil {
		return model.CallResponseError, err
	}

	if set.SetVar != "" {
		return conn.Set(ctx, model.Variables{
			set.SetVar: id,
		})
	}

	return model.CallResponseOK, nil
}
