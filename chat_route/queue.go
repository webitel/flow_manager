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

type Queue struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type QueueJoinArg struct {
	Priority int32 `json:"priority"`
	BucketId int32 `json:"bucket_id"` // TODO
	Queue    Queue `json:"queue"`
	//Timers              []TimerArgs         `json:"timers"`
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	ctx2 := context.Background()
	res, err := r.fm.JoinChatToInboundQueue(ctx2, &cc.ChatJoinToQueueRequest{
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

		switch msg.Data.(type) {
		case *cc.QueueEvent_Bridged:
			fmt.Println("BRIDGED")

		case *cc.QueueEvent_Leaving:
			conv.Set(ctx, model.Variables{
				"cc_result": msg.Data.(*cc.QueueEvent_Leaving).Leaving.Result,
			})
			break
		}
	}

	return model.CallResponseOK, nil
}
