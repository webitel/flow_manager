package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type LastBridgedFilter struct {
	Dialer   *string `json:"dialer"`
	Inbound  *string `json:"inbound"`
	Outbound *string `json:"outbound"`
	QueueIds []int   `json:"queue_ids"`
}

type LastBridged struct {
	Calls  *LastBridgedFilter // deprecated
	Filter LastBridgedFilter
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

	if argv.Calls != nil {
		argv.Filter = *argv.Calls
	}

	if c.Type() == model.ConnectionTypeChat {
		//argv.Calls.Dialer, argv.Calls.Inbound, argv.Calls.Outbound,
		lastBridged, err = r.fm.LastBridgedChat(c.DomainId(), argv.Number, argv.Hours, argv.Filter.QueueIds, argv.Set)
	} else {
		lastBridged, err = r.fm.LastBridgedCall(c.DomainId(), argv.Number, argv.Hours, argv.Filter.Dialer, argv.Filter.Inbound, argv.Filter.Outbound, argv.Filter.QueueIds, argv.Set)
	}

	if err != nil {
		return nil, err
	}

	return c.Set(ctx, lastBridged)
}
