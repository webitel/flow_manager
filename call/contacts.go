package call

import (
	"fmt"
	"github.com/webitel/engine/pkg/webitel_client"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"strconv"
)

func (r *Router) linkContact(call model.Call) {
	if c := call.GetContactId(); c != 0 {
		return
	}

	var dest string
	if call.Direction() == model.CallDirectionOutbound {
		dest = call.Destination()
	} else if call.From() != nil {
		dest = *call.From().GetNumber()
	} else {

		return
	}

	list, err := r.fm.SearchContactsNA(call.Context(), &webitel_client.SearchContactsRequestNA{
		DomainId: call.DomainId(),
		Qin:      []string{"phones"},
		Q:        dest,
		Size:     2,
		Fields:   []string{"id"},
	})

	if err != nil {
		wlog.Error(fmt.Sprintf("call %s, listContact error: %s", call.Id(), err.Error()))
		return
	}

	userIdStr, _ := call.Get("sip_h_X-Webitel-User-Id")
	userId, _ := strconv.Atoi(userIdStr)
	var contactId *int

	if len(list.Data) == 1 {
		call.Set(call.Context(), model.Variables{
			"wbt_contact_id": list.Data[0].Id,
		})

		cId, _ := strconv.Atoi(list.Data[0].Id)
		contactId = &cId
		r.fm.CallSetContactId(call.DomainId(), call.Id(), int64(cId))
	}

	if userId > 0 {
		n := model.Notification{
			DomainId:  call.DomainId(),
			Action:    "set_contact", // TODO
			CreatedAt: model.GetMillis(),
			ForUsers:  []int64{int64(userId)},
			Body: map[string]interface{}{
				"id":         call.Id(),
				"contact_id": contactId,
				"channel":    model.CallExchange,
			},
		}

		r.fm.UserNotification(n)
	}
}
