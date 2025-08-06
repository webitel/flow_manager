package flow

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

const (
	actionOpenLink    = "open_link"
	descTrackAppName  = "desc_track"
	variableWbtSockId = "wbt_sock_id"
)

type OpenLinkArgs struct {
	UserId  int64  `json:"userId"`
	Message string `json:"message"`
	Url     string `json:"url"`
}

func (r *router) openLink(_ context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = OpenLinkArgs{}
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	var sockSession *model.SocketSession

	sockId, _ := conn.Get(variableWbtSockId)
	if sockId == "" {
		sockSession, err = r.fm.Store.SocketSession().Get(argv.UserId, conn.DomainId(), descTrackAppName)
		if err != nil {
			return nil, model.NewAppError("open_link", "store.open_link.error", nil, err.Error(), http.StatusInternalServerError)
		}
		sockId = sockSession.ID
	}

	n := model.Notification{
		DomainId:  conn.DomainId(),
		Action:    actionOpenLink,
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{argv.UserId},
		SockID:    sockId,
		Body: map[string]interface{}{
			"url":     argv.Url,
			"message": argv.Message,
		},
	}

	r.fm.OpenLink(n)
	return model.CallResponseOK, nil
}
