package email

import (
	"fmt"
	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
	"strconv"
	"time"
)

func (r *Router) linkContact(conn model.EmailConnection) {
	email := conn.Email()
	if email == nil || len(email.From) == 0 {
		return
	}

	list, err := r.fm.SearchContactsNA(conn.Context(), &contacts.SearchContactsNARequest{
		DomainId: conn.DomainId(),
		Qin:      []string{"emails"},
		Q:        email.From[0],
		Size:     2,
		Fields:   []string{"id"},
	})

	if err != nil {
		conn.Log().Error("listContact error:" + err.Error())
		return
	}
	now := time.Now()
	defer func() {
		conn.Log().Debug("linkContact took: " + time.Since(now).String())
	}()
	if len(list.Data) == 1 {
		conn.Set(conn.Context(), model.Variables{
			"wbt_contact_id": list.Data[0].Id,
		})

		cId, _ := strconv.Atoi(list.Data[0].Id)
		err = r.fm.MailSetContacts(conn.Context(), conn.DomainId(), conn.Id(), []int64{int64(cId)})
		if err != nil {
			conn.Log().Error("mailSetContacts error:" + err.Error())
		}
	} else {
		conn.Log().Debug(fmt.Sprintf("skip link contact, find contacts %d", len(list.Data)))
	}
}
