package app

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
	"strconv"
)

const (
	engineExchange   = "engine"
	actionOpenLink   = "open_link"
	descTrackAppName = "desc_track"
)

func (fm *FlowManager) OpenLink(domainId int64, sockId string, userId int64, message string, url string) *model.AppError {
	var sockSession *model.SocketSession
	var err *model.AppError

	if sockId == "" {
		sockSession, err = fm.Store.SocketSession().Get(userId, domainId, descTrackAppName)
		if err != nil {
			return model.NewAppError("open_link", "store.open_link.error", nil, err.Error(), http.StatusInternalServerError)
		}
		sockId = sockSession.ID
	}

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

	err = fm.eventQueue.SendJSON(engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson())
	if err != nil {
		wlog.Error(err.Error())
	}

	return err
}
