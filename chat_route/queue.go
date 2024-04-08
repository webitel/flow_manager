package chat_route

import (
	"context"
	"fmt"
	"io"

	cc "buf.build/gen/go/webitel/cc/protocolbuffers/go"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
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
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

type QueueJoinArg struct {
	Priority int32 `json:"priority"`
	Bucket   struct {
		Id int32 `json:"id"`
	} `json:"bucket"`
	Queue Queue `json:"queue"`
	Agent *struct {
		Id        *int32  `json:"id"`
		Extension *string `json:"extension"`
	}
	Timers   []flow.TimerArgs `json:"timers"`
	Offering []interface{}    `json:"offering"`
	Missed   []interface{}    `json:"missed"`
	Bridged  []interface{}    `json:"bridged"`
}

func (r *Router) cancelQueue(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	key := conv.GetQueueKey()
	if key == nil {
		//TODO NO QUEUE
		return model.CallResponseError, nil
	}

	err := r.fm.CancelAttempt(ctx, *key, "cancel")
	if err != nil {
		return nil, err
	}

	return conv.Set(ctx, model.Variables{
		"cc_cancel": fmt.Sprintf("%v", conv.SetQueue(nil)),
	})
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg
	var wCancel context.CancelFunc
	var wCtx context.Context
	var stickyAgentId int32

	wCtx, wCancel = context.WithCancel(ctx)

	if err := r.Decode(scope, args, &q); err != nil {
		wCancel()
		return nil, err
	}

	if q.Queue.Id == 0 && q.Queue.Name != "" {
		var err *model.AppError
		if q.Queue.Id, err = r.fm.FindQueueByName(conv.DomainId(), q.Queue.Name); err != nil {
			wCancel()
			return nil, err
		}
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

	if q.Agent != nil {
		if q.Agent.Extension != nil && q.Agent.Id == nil {
			q.Agent.Id, _ = r.fm.GetAgentIdByExtension(conv.DomainId(), *q.Agent.Extension)
		}

		if q.Agent.Id != nil {
			stickyAgentId = *q.Agent.Id
		}
	}

	res, err := r.fm.JoinChatToInboundQueue(ctx, &cc.ChatJoinToQueueRequest{
		ConversationId: conv.Id(),
		Queue: &cc.ChatJoinToQueueRequest_Queue{
			Id:   q.Queue.Id,
			Name: q.Queue.Name,
		},
		Priority:      q.Priority,
		BucketId:      q.Bucket.Id,
		Variables:     conv.DumpExportVariables(),
		DomainId:      conv.DomainId(),
		StickyAgentId: stickyAgentId,
	})

	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseOK, nil
	}

	defer conv.SetQueue(nil)

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
		case *cc.QueueEvent_Joined:
			conv.SetQueue(&model.InQueueKey{
				AttemptId: e.Joined.AttemptId,
				AppId:     e.Joined.AppId,
			})
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
			conv.SetQueue(nil)
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
