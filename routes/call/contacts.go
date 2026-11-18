package call

import (
	"context"

	"github.com/webitel/flow_manager/contactid"
	"github.com/webitel/flow_manager/model"
)

// linkContact is the call auto-link (domain setting autolink_call_to_contact):
// before the schema runs, the caller's number is searched and a single
// verified contact is bound. The matches are also kept as the call's contact
// candidates.
func (r *Router) linkContact(call model.Call) {
	if c := call.GetContactId(); c != 0 {
		return
	}

	ctx, cancel := context.WithTimeout(call.Context(), autoLinkTimeout)
	defer cancel()

	_, bound, err := identifyContact(ctx, r.fm, call, contactid.Request{}, true)
	if err != nil {
		call.Log().Error("listContact error:" + err.Error())

		return
	}

	if !bound {
		// Nothing bound: the client still learns the lookup finished.
		notifyContact(r.fm, call, nil)
	}
}
