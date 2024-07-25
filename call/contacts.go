package call

import (
	"fmt"
	"github.com/webitel/engine/pkg/webitel_client"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (r *Router) linkContact(call model.Call) {
	if c, _ := call.Get("wbt_contact_id"); c != "" {
		return
	}

	list, err := r.fm.SearchContactsNA(call.Context(), &webitel_client.SearchContactsRequestNA{})

	if err != nil {
		wlog.Error(fmt.Sprintf("call %s, listContact error: %s", call.Id(), err.Error()))
		return
	}

	if len(list.Data) == 1 {

	}
}
