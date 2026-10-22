package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

const (
	notificationAction      = "show_message"
	defaultNotificationType = "info"
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

	body := map[string]any{
		"message": argv.Message,
		"timeout": argv.Timeout,
		"type":    argv.Type,
	}

	n := model.Notification{
		DomainId:  conn.DomainId(),
		Action:    notificationAction,
		CreatedAt: model.GetMillis(),
		ForUsers:  argv.UserIds,
		Body:      body,
	}

	un := &model.UserNotification{
		DomainId: conn.DomainId(),
		ForUsers: argv.UserIds,
		Type:     argv.Type,
		Message:  argv.Message,
	}
	if un.Type == "" {
		un.Type = defaultNotificationType
	}

	if err = r.fm.SaveUserNotification(ctx, un); err != nil {
		conn.Log().Error(err.Error())
	} else {
		n.Id = un.Id
		n.CreatedAt = un.CreatedAt
		body["id"] = un.Id
	}

	r.fm.UserNotification(n)
	return model.CallResponseOK, nil
}
