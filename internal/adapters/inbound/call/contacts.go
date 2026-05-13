package call

import (
	"strconv"

	"github.com/webitel/flow_manager/api/gen/contacts"
	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/notification"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

func (r *Router) linkContact(c calldomain.Call) {
	if ct := c.GetContactId(); ct != 0 {
		return
	}

	var dest string
	if c.Direction() == calldomain.CallDirectionOutbound {
		dest = c.Destination()
	} else if c.From() != nil {
		dest = *c.From().GetNumber()
	} else {
		return
	}

	list, err := r.contacts.SearchNA(c.Context(), &contacts.SearchContactsNARequest{
		DomainId: c.DomainId(),
		Qin:      []string{"phones"},
		Q:        dest,
		Size:     2,
		Fields:   []string{"id"},
	})
	if err != nil {
		c.Log().Error("listContact error:" + err.Error())
		return
	}

	userIdStr, _ := c.Get("sip_h_X-Webitel-User-Id")
	userId, _ := strconv.Atoi(userIdStr)
	var contactId *int

	if len(list.Data) == 1 {
		c.Set(c.Context(), flow.Variables{
			"wbt_contact_id": list.Data[0].Id,
		})

		cId, _ := strconv.Atoi(list.Data[0].Id)
		contactId = &cId
		r.fm.CallSetContactId(c.DomainId(), c.Id(), int64(cId))
	}

	if userId > 0 {
		n := notification.Notification{
			DomainId:  c.DomainId(),
			Action:    "set_contact", // TODO
			CreatedAt: utils.GetMillis(),
			ForUsers:  []int64{int64(userId)},
			Body: map[string]any{
				"id":         c.Id(),
				"contact_id": contactId,
				"channel":    calldomain.CallExchange,
			},
		}

		r.fm.UserNotification(n)
	}
}
