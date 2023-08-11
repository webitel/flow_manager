package flow

import (
	"context"

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
	panic("TODO")
}

func (r *router) updateContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	panic("TODO")
}

func (r *router) addContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	panic("TODO")
}
