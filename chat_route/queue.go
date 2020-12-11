package chat_route

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
	"github.com/webitel/wlog"
)

type QueueJoinArg struct {
	Priority int32               `json:"priority"`
	Bucket   *model.SearchEntity `json:"bucket"`
	Queue    *model.SearchEntity `json:"queue"`
	//Timers              []TimerArgs         `json:"timers"`
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var q QueueJoinArg

	if err := r.Decode(scope, args, &q); err != nil {
		return nil, err
	}

	var wCancel context.CancelFunc
	var wCtx context.Context
	wCtx, wCancel = context.WithCancel(ctx)

	defer func() {
		if wCancel != nil {
			wCancel()
			wCancel = nil
		}
	}()

	ctx2 := context.Background()
	res, err := r.fm.JoinChatToInboundQueue(ctx2, &cc.ChatJoinToQueueRequest{
		Priority:       q.Priority,
		DomainId:       conv.DomainId(),
		ConversationId: conv.Id(),
	})

	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseOK, nil
	}

	fmt.Println(res, wCtx)

	return model.CallResponseOK, nil
}
