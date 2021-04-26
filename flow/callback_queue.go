package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type CallbackCommunication struct {
	Destination string
	TypeId      int `json:"type_id"`
}

type CallbackQueueArgs struct {
	Name          string
	Variables     map[string]string
	QueueId       int `json:"queue_id"`
	HoldSec       int `json:"holdSec"`
	Communication CallbackCommunication
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
	var member CallbackQueueArgs
	if err := scope.Decode(args, &member); err != nil {
		return nil, err
	}

	if err := r.fm.AddMemberToQueueQueue(c.DomainId(), member.QueueId, member.Communication.Destination, member.Name, member.Communication.TypeId,
		member.HoldSec, member.Variables); err != nil {
		return nil, err
	}

	return ResponseOK, nil
}
