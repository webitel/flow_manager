package notification

import (
	"context"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

const notificationAction = "show_message"

// Deps is the subset of RouterDeps that the notification op needs.
type Deps interface {
	UserNotification(n model.Notification)
}

type notificationArgs struct {
	UserIds []int64 `json:"userIds"`
	Message string  `json:"message"`
	Timeout int     `json:"timeout"`
	Type    string  `json:"type"`
}

type notificationOp struct{ deps Deps }

func New(deps Deps) ops.Op { return &notificationOp{deps} }

func (o *notificationOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *notificationOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv notificationArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	o.deps.UserNotification(model.Notification{
		DomainId:  in.DomainID,
		Action:    notificationAction,
		CreatedAt: model.GetMillis(),
		ForUsers:  argv.UserIds,
		Body: map[string]interface{}{
			"message": argv.Message,
			"timeout": argv.Timeout,
			"type":    argv.Type,
		},
	})
	return ops.OpOutput{}, nil
}
