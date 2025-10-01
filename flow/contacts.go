package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

type GetContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	Id     string `json:"id"`
	contacts.LocateContactRequest
}

type FindContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	contacts.SearchContactsRequest
}

type AddContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	contacts.InputContactRequest
}

type UpdateContactRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	contacts.InputContactRequest
}

type LinkContactArgv struct {
	SessionId  string  `json:"sessionId"`
	ContactId  int64   `json:"contactId"`
	ContactIds []int64 `json:"contactIds"`
	Channel    string  `json:"channel"`
}

type MergeContactPhonesRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	contacts.MergePhonesRequest
}

type MergeContactVariablesRequest struct {
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	contacts.MergeVariablesRequest
}

func (r *router) mergeContactPhones(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *MergeContactPhonesRequest
	var err *model.AppError
	var res *contacts.PhoneList

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.MergePhonesRequest); err != nil {
		return nil, err
	}

	res, err = r.fm.MergeContactPhones(argv.Token, &argv.MergePhonesRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}

func (r *router) mergeContactVariables(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *MergeContactVariablesRequest
	var err *model.AppError
	var res *contacts.VariableList

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "setVar")
	}
	if argv.Token == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "token")
	}

	if err = scope.Decode(args, &argv.MergeVariablesRequest); err != nil {
		return nil, err
	}

	res, err = r.fm.MergeContactVariables(argv.Token, &argv.MergeVariablesRequest)
	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: model.ToJson(res),
	})
}

func (r *router) getContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *GetContactRequest
	var err *model.AppError
	var res *contacts.Contact

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
	if argv.LocateContactRequest.Etag == "" {
		argv.LocateContactRequest.Etag = argv.Id
	}

	if argv.LocateContactRequest.Etag == "" {
		return model.CallResponseError, model.ErrorRequiredParameter("getContact", "etag")
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
	var res *contacts.ContactList

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

	contacts := argv.ContactIds

	if argv.ContactId != 0 {
		contacts = []int64{argv.ContactId}
	}

	if len(contacts) == 0 {
		return model.CallResponseError, model.ErrorRequiredParameter("linkContact", "Contact")
	}

	if argv.SessionId == "" {
		argv.SessionId = conn.Id()
	}

	channel := conn.Type()

	switch argv.Channel {
	case "call":
		channel = model.ConnectionTypeCall
	case "email":
		channel = model.ConnectionTypeEmail
	case "chat":
		channel = model.ConnectionTypeChat
	}

	switch channel {
	case model.ConnectionTypeCall:
		err = r.fm.CallSetContactId(conn.DomainId(), argv.SessionId, contacts[0])
	case model.ConnectionTypeChat:
		err = r.fm.ContactLinkToChat(ctx, argv.SessionId, fmt.Sprintf("%v", contacts[0]))
	case model.ConnectionTypeEmail:
		err = r.fm.MailSetContacts(ctx, conn.DomainId(), argv.SessionId, contacts)

	default:
		return model.CallResponseError, model.NewAppError("flow", "flow.todo", nil, "", http.StatusInternalServerError)
	}

	if err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		"wbt_contact_id": fmt.Sprintf("%d", contacts[0]), // TODO
	})
}

func (r *router) updateContact(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv *UpdateContactRequest
	var err *model.AppError
	var res *contacts.Contact

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
	var res *contacts.Contact

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
