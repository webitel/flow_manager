package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/engine/pkg/webitel_client"
	"github.com/webitel/flow_manager/model"
)

type GetContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	webitel_client.LocateContactRequest
}

type FindContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	webitel_client.SearchContactsRequest
}

type AddContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	webitel_client.InputContactRequest
}

type UpdateContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	webitel_client.InputContactRequest
}

type LinkContactArgv struct {
	SessionId string `json:"sessionId"`
	ContactId int64  `json:"contactId"`
}

func (r *router) getContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *GetContactRequest
	var err *model.AppError
	var res *webitel_client.Contact

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.LocateContactRequest); err != nil {
		return nil, err
	}

	if argv.LocateContactRequest.Id == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "id")
	}

	res, err = r.fm.LocateContact(argv.Token, &argv.LocateContactRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}

func (r *router) findContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *FindContactRequest
	var err *model.AppError
	var res *webitel_client.ContactList

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.SearchContactsRequest); err != nil {
		return nil, err
	}

	res, err = r.fm.SearchContacts(argv.Token, &argv.SearchContactsRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}

func (r *router) linkContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *LinkContactArgv
	var err *model.AppError
	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.ContactId == 0 {
		return model.CallResponseError, model.ErrorRequiredParameter("linkContact", "ContactId")
	}

	switch conn.Type() {
	case model.ConnectionTypeCall:
		err = r.fm.SetContactId(conn.DomainId(), conn.Id(), argv.ContactId)
	default:
		return model.CallResponseError, model.NewAppError("flow", "flow.todo", nil, "", http.StatusInternalServerError)
	}

	return conn.Set(ctx, model.Variables{
		"wbt_contact_id": fmt.Sprintf("%d", argv.ContactId),
	})
}

func (r *router) updateContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *UpdateContactRequest
	var err *model.AppError
	var res *webitel_client.Contact

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.InputContactRequest); err != nil {
		return nil, err
	}

	res, err = r.fm.UpdateContact(argv.Token, &argv.InputContactRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}

func (r *router) addContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *AddContactRequest
	var err *model.AppError
	var res *webitel_client.Contact

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.InputContactRequest); err != nil {
		return nil, err
	}

	res, err = r.fm.CreateContact(argv.Token, &argv.InputContactRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}
