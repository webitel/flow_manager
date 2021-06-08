package flow

import (
	"context"
	"fmt"
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
	Set    map[string]string
}

func (r *router) lastBridged(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = LastBridged{}
	var lastBridged *model.LastBridged

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if len(argv.Set) == 0 {
		return nil, ErrorRequiredParameter("lastBridged", "setVar")
	}

	lastBridged, err = r.fm.LastBridgedExtension(c.DomainId(), argv.Number, argv.Hours, argv.Calls.Dialer, argv.Calls.Inbound, argv.Calls.Outbound, argv.Calls.QueueIds)
	if err != nil {
		return nil, err
	}

	vars := make(model.Variables)

	for k, v := range argv.Set {
		switch v {
		case "extension":
			vars[k] = lastBridged.Extension
		case "agent_id":
			if lastBridged.AgentId != nil {
				vars[k] = fmt.Sprintf("%d", *lastBridged.AgentId)
			}
		case "queue_id":
			if lastBridged.QueueId != nil {
				vars[k] = fmt.Sprintf("%d", *lastBridged.QueueId)
			}
		}
	}

	if len(vars) == 0 {
		return model.CallResponseOK, nil
	}

	return c.Set(ctx, vars)
}
