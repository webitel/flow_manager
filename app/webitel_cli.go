package app

import (
	"context"
	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (fm *FlowManager) CreateContact(token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError) {
	ctx := fm.contacts.WithToken(context.Background(), token)
	c, err := fm.contacts.Api.CreateContact(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "CreateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) LocateContact(token string, req *contacts.LocateContactRequest) (*contacts.Contact, *model.AppError) {
	ctx := fm.contacts.WithToken(context.Background(), token)
	c, err := fm.contacts.Api.LocateContact(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "LocateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) UpdateContact(token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError) {
	ctx := fm.contacts.WithToken(context.Background(), token)
	c, err := fm.contacts.Api.UpdateContact(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "UpdateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) SearchContacts(token string, req *contacts.SearchContactsRequest) (*contacts.ContactList, *model.AppError) {
	ctx := fm.contacts.WithToken(context.Background(), token)
	c, err := fm.contacts.Api.SearchContacts(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) SearchContactsNA(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError) {
	c, err := fm.contacts.Api.SearchContactsNA(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}
