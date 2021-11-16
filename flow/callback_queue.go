package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type CallbackQueueArgs struct {
	QueueId int `json:"queue_id"`
	HoldSec int `json:"holdSec"`
}

/*
   {
       "callbackQueue": {
           "communication": {
               "destination": "${caller_id_number}",
               "type_id": 1
           },
           "holdSec": "60",
           "name": "${caller_id_number}",
           "queue_id": 1,
           "variables": {
               "aaa": "2321321"
           }
       }
   }
*/

func (r *router) callbackQueue(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var params CallbackQueueArgs
	if err := scope.Decode(args, &params); err != nil {
		return nil, err
	}
	var member model.CallbackMember
	if err := scope.Decode(args, &member); err != nil {
		return nil, err
	}

	if err := r.fm.CreateMember(c.DomainId(), params.QueueId, params.HoldSec, &member); err != nil {
		return nil, err
	}

	return ResponseOK, nil
}
