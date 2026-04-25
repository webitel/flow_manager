package app

import (
	"context"
	"net/http"
	"strconv"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

const (
	engineExchange   = "engine"
	actionOpenLink   = "open_link"
	descTrackAppName = "desc_track"
)

func (fm *FlowManager) OpenLink(domainId int64, sockId string, userId int64, message string, url string) *model.AppError {
	if sockId == "" {
		sockSession, storeErr := fm.Store.SocketSession().Get(userId, domainId, descTrackAppName)
		if storeErr != nil {
			return model.NewAppError("open_link", "store.open_link.error", nil, storeErr.Error(), http.StatusInternalServerError)
		}
		sockId = sockSession.ID
	}

	var err *model.AppError

	n := model.Notification{
		DomainId:  domainId,
		Action:    actionOpenLink,
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{userId},
		SockID:    sockId,
		Body: map[string]interface{}{
			"url":     url,
			"message": message,
		},
	}

	if pubErr := fm.eventQueue.Publish(context.Background(), engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson()); pubErr != nil {
		wlog.Error(pubErr.Error())
		err = model.NewAppError("open_link", "mq.publish.err", nil, pubErr.Error(), http.StatusInternalServerError)
	}

	return err
}
