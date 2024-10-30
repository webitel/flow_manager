package chat_route

import (
	"context"
	"strings"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ReceiveMessage struct {
	Timeout        int
	MessageTimeout int `json:"messageTimeout"`
	Delimiter      string
	Set            string
}

func (r *Router) recvMessage(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv ReceiveMessage
	delimiter := " "

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Set == "" {
		return model.CallResponseOK, nil
	}

	if len(argv.Delimiter) != 0 {
		delimiter = argv.Delimiter
	}

	msgs, err := conv.ReceiveMessage(ctx, argv.Set, argv.Timeout, argv.MessageTimeout)
	if err != nil {
		conv.Set(ctx, model.Variables{
			argv.Set: "",
		})
		return nil, err
	}

	if scope.CountTriggers() > 0 {
		for _, m := range msgs {
			commandName := flow.TriggerCommandsName(m)
			if scope.HasTrigger(commandName) {
				err = scope.TriggerScopeAsync(ctx, commandName, r)
				if err != nil {
					wlog.Error(err.Error())
				}
				// TODO
				_, err = conv.Set(ctx, model.Variables{
					argv.Set: m,
				})
				if err != nil {
					wlog.Error(err.Error())
				}

				return r.recvMessage(ctx, scope, conv, args)
			}
		}
	}

	return conv.Set(ctx, model.Variables{
		argv.Set: strings.Join(msgs, delimiter),
	})
}
