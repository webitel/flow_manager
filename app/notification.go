package app

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"strconv"
)

func (fm *FlowManager) NotificationMissedCalls(call model.MissedCall) {
	n := model.Notification{
		DomainId:  call.DomainId,
		Action:    refreshMissedNotification,
		CreatedAt: model.GetMillis(),
		ForUsers:  []int64{call.UserId},
		Body: map[string]interface{}{
			"call_id": call.Id,
		},
	}

	fm.UserNotification(n)
}

func (fm *FlowManager) UserNotification(n model.Notification) {
	err := fm.eventQueue.SendJSON("engine", "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson())
	if err != nil {
		wlog.Error(err.Error())
	}
}
