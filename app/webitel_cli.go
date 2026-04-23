package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) SearchContactsNA1(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError) {
	c, err := fm.contacts.Api.SearchContactsNA(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}
