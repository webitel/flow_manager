package app

import (
	"context"
	"net/http"

	"github.com/webitel/engine/pkg/webitel_client"
	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) CreateContact(token string, req *webitel_client.InputContactRequest) (*webitel_client.Contact, *model.AppError) {
	c, err := fm.wbtCli.CreateContact(token, req)
	if err != nil {
		return nil, model.NewAppError("App", "CreateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) LocateContact(token string, req *webitel_client.LocateContactRequest) (*webitel_client.Contact, *model.AppError) {
	c, err := fm.wbtCli.LocateContact(token, req)
	if err != nil {
		return nil, model.NewAppError("App", "LocateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) UpdateContact(token string, req *webitel_client.InputContactRequest) (*webitel_client.Contact, *model.AppError) {
	c, err := fm.wbtCli.UpdateContact(token, req)
	if err != nil {
		return nil, model.NewAppError("App", "UpdateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) SearchContacts(token string, req *webitel_client.SearchContactsRequest) (*webitel_client.ContactList, *model.AppError) {
	c, err := fm.wbtCli.SearchContacts(token, req)
	if err != nil {
		return nil, model.NewAppError("App", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}

func (fm *FlowManager) SearchContactsNA(ctx context.Context, req *webitel_client.SearchContactsRequestNA) (*webitel_client.ContactList, *model.AppError) {
	c, err := fm.wbtCli.SearchContactsNA(ctx, req)
	if err != nil {
		return nil, model.NewAppError("App", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return c, nil
}
