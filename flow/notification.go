package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

const (
	notificationAction = "show_message"
)

type NotificationArgs struct {
	UserIds []int64 `json:"userIds"`
	Message string  `json:"message"`
	Timeout int     `json:"timeout"`
	Type    string  `json:"type"`
}

func (r *router) notification(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = NotificationArgs{}
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	n := model.Notification{
		DomainId:  conn.DomainId(),
		Action:    notificationAction,
		CreatedAt: model.GetMillis(),
		ForUsers:  argv.UserIds,
		Body: map[string]interface{}{
			"message": argv.Message,
			"timeout": argv.Timeout,
			"type":    argv.Type,
		},
	}

	r.fm.UserNotification(n)
	return model.CallResponseOK, nil
}
