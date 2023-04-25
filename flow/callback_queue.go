package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type CallbackQueueArgs struct {
	QueueId int `json:"queue_id"`
	HoldSec int `json:"holdSec"`
}

func (r *router) callbackQueue(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var params CallbackQueueArgs
	if err := scope.Decode(args, &params); err != nil {
		return nil, err
	}
	var member model.CallbackMember
	if err := scope.Decode(args, &member); err != nil {
		return nil, err
	}

	// todo deprecated queue_id
	if member.Queue.Id != nil {
		params.QueueId = *member.Queue.Id
	}

	//todo deprecated TypeId
	if member.Communication.TypeId != nil && member.Communication.Type.Id == nil {
		member.Communication.Type.Id = member.Communication.TypeId
	}

	if err := r.fm.CreateMember(c.DomainId(), params.QueueId, params.HoldSec, &member); err != nil {
		return nil, err
	}

	return ResponseOK, nil
}
