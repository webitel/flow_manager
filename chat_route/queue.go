package chat_route

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
	"github.com/webitel/wlog"
	"io"
)

/*
 {
	"joinQueue": {
		"bridged": [
			{
				"sendText": "на связи!!"
			}
		],
		"missed": [
			{
				"sendText": "пока, я мизантроп"
			}
		],
		"offering": [
			{
				"sendText": "Хело, май нейм ${cc_agent_name}"
			}
		],
		"priority": 100,
		"queue": {
			"id": 223
		},
		"timers": [
			{
				"actions": [
					{
						"sendText": "ще черга!!!!!"
					}
				],
				"interval": 5,
				"tries": 20
			}
		]
	}
}
*/

type Queue struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type QueueJoinArg struct {
	Priority int32            `json:"priority"`
	BucketId int32            `json:"bucket_id"` // TODO
	Queue    Queue            `json:"queue"`
	Timers   []flow.TimerArgs `json:"timers"`
	Offering []interface{}    `json:"offering"`
	Missed   []interface{}    `json:"missed"`
	Bridged  []interface{}    `json:"bridged"`
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg
	var wCancel context.CancelFunc
	var wCtx context.Context

	wCtx, wCancel = context.WithCancel(ctx)

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	if len(q.Timers) > 0 {
		for k, t := range q.Timers {
			t.Name = fmt.Sprintf("queue-timer-%d", k)
			go scope.Timer(wCtx, t, r)
		}
	}

	defer func() {
		if wCancel != nil {
			wCancel()
			wCancel = nil
		}
	}()

	//ctx2 := context.Background()
	res, err := r.fm.JoinChatToInboundQueue(ctx, &cc.ChatJoinToQueueRequest{
		ConversationId: conv.Id(),
		Queue: &cc.ChatJoinToQueueRequest_Queue{
			Id:   int32(q.Queue.Id),
			Name: q.Queue.Name,
		},
		Priority:  q.Priority,
		BucketId:  q.BucketId,
		Variables: conv.DumpExportVariables(),
		DomainId:  conv.DomainId(),
	})

	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseOK, nil
	}

	// TODO bug close stream channel
	for {
		var msg cc.QueueEvent
		err = res.RecvMsg(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			wlog.Error(err.Error())
			return model.CallResponseError, nil
		}

		switch e := msg.Data.(type) {
		case *cc.QueueEvent_Offering:
			conv.Set(ctx, model.Variables{
				"cc_agent_name": e.Offering.AgentName,
				"cc_agent_id":   e.Offering.AgentId,
			})
			if len(q.Offering) > 0 {
				go flow.Route(wCtx, scope.Fork("queue-offering", flow.ArrInterfaceToArrayApplication(q.Offering)), r)
			}
		case *cc.QueueEvent_Missed:
			conv.Set(ctx, model.Variables{
				"cc_agent_name": "",
				"cc_agent_id":   "",
			})
			if len(q.Missed) > 0 {
				go flow.Route(wCtx, scope.Fork("queue-missed", flow.ArrInterfaceToArrayApplication(q.Missed)), r)
			}

		case *cc.QueueEvent_Bridged:
			if len(q.Bridged) > 0 {
				flow.Route(wCtx, scope.Fork("queue-bridged", flow.ArrInterfaceToArrayApplication(q.Bridged)), r)
			}
			if wCancel != nil {
				wCancel()
				wCancel = nil
			}

		case *cc.QueueEvent_Leaving:
			conv.Set(ctx, model.Variables{
				"cc_result": msg.Data.(*cc.QueueEvent_Leaving).Leaving.Result,
			})
			break
		}
	}

	return model.CallResponseOK, nil
}
