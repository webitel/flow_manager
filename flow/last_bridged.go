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
		QueueIds []int   `json:"queue_ids"`
	}
	Hours  string `json:"hours"`
	Number string `json:"number"`
	Set    model.Variables
}

func (r *router) lastBridged(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = LastBridged{}
	var lastBridged model.Variables

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if len(argv.Set) == 0 {
		return nil, ErrorRequiredParameter("lastBridged", "setVar")
	}

	lastBridged, err = r.fm.LastBridged(c.DomainId(), argv.Number, argv.Hours, argv.Calls.Dialer, argv.Calls.Inbound, argv.Calls.Outbound, argv.Calls.QueueIds, argv.Set)
	if err != nil {
		return nil, err
	}

	return c.Set(ctx, lastBridged)
}
