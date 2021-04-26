package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type LastBridged struct {
	Calls struct {
		Dialer   *string `json:"dialer"`
		Inbound  *string `json:"inbound"`
		Outbound *string `json:"outbound"`
	}
	Hours  string `json:"hours"`
	Number string `json:"number"`
	SetVar string `json:"setVar"`
}

func (r *router) lastBridged(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = LastBridged{}
	var extension string

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return nil, ErrorRequiredParameter("lastBridged", "setVar")
	}

	extension, err = r.fm.LastBridgedExtension(c.DomainId(), argv.Number, argv.Hours, argv.Calls.Dialer, argv.Calls.Inbound, argv.Calls.Outbound)
	if err != nil {
		return nil, err
	}

	return c.Set(ctx, model.Variables{
		argv.SetVar: extension,
	})
}
