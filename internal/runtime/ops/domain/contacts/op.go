// Package contacts provides native ops for the Webitel Contacts service.
package contacts

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/webitel/flow_manager/gen/contacts"
	domcontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/model"
)

// LinkDeps is the subset of RouterDeps that linkContact needs.
type LinkDeps interface {
	CallSetContactId(domainId int64, callId string, contactId int64) error
	ContactLinkToChat(ctx context.Context, conversationId string, contactId string) error
	MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) error
}

// Register adds all contacts ops to reg.
func Register(reg *ops.Registry, client domcontacts.Client, link LinkDeps) {
	reg.Register("getContact", &getContactOp{client})
	reg.Register("findContact", &findContactOp{client})
	reg.Register("addContact", &addContactOp{client})
	reg.Register("updateContact", &updateContactOp{client})
	reg.Register("mergeContactPhones", &mergeContactPhonesOp{client})
	reg.Register("mergeContactVariables", &mergeContactVariablesOp{client})
	reg.Register("linkContact", &linkContactOp{link})
}

func marshalToVar(setVar string, res any) (ops.OpOutput, error) {
	b, err := json.Marshal(res)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("contacts: marshal result: %w", err)
	}
	str := string(b)
	if str == "{}" || str == "[]" || str == `""` {
		str = ""
	}
	return ops.OpOutput{SetVars: map[string]string{setVar: str}}, nil
}

func requireTokenAndSetVar(opName, token, setVar string) error {
	if token == "" {
		return fmt.Errorf("%s: token is required", opName)
	}
	if setVar == "" {
		return fmt.Errorf("%s: setVar is required", opName)
	}
	return nil
}

// ── getContact ────────────────────────────────────────────────────────────────

type getContactOp struct{ client domcontacts.Client }

func (o *getContactOp) Kind() ops.OpKind { return ops.OpKindSync }

type getContactArgs struct {
	pb.LocateContactRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
	Id     string `json:"id"`
}

func (o *getContactOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getContactArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("getContact", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.LocateContactRequest.Etag == "" {
		argv.LocateContactRequest.Etag = argv.Id
	}
	if argv.LocateContactRequest.Etag == "" {
		return ops.OpOutput{}, fmt.Errorf("getContact: etag is required")
	}
	res, err := o.client.Locate(ctx, argv.Token, &argv.LocateContactRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("getContact: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── findContact ───────────────────────────────────────────────────────────────

type findContactOp struct{ client domcontacts.Client }

func (o *findContactOp) Kind() ops.OpKind { return ops.OpKindSync }

type findContactArgs struct {
	pb.SearchContactsRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *findContactOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv findContactArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("findContact", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.Search(ctx, argv.Token, &argv.SearchContactsRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("findContact: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── addContact ────────────────────────────────────────────────────────────────

type addContactOp struct{ client domcontacts.Client }

func (o *addContactOp) Kind() ops.OpKind { return ops.OpKindSync }

type addContactArgs struct {
	pb.InputContactRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *addContactOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv addContactArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("addContact", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.Create(ctx, argv.Token, &argv.InputContactRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("addContact: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── updateContact ─────────────────────────────────────────────────────────────

type updateContactOp struct{ client domcontacts.Client }

func (o *updateContactOp) Kind() ops.OpKind { return ops.OpKindSync }

type updateContactArgs struct {
	pb.InputContactRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *updateContactOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv updateContactArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("updateContact", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.Update(ctx, argv.Token, &argv.InputContactRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("updateContact: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── mergeContactPhones ────────────────────────────────────────────────────────

type mergeContactPhonesOp struct{ client domcontacts.Client }

func (o *mergeContactPhonesOp) Kind() ops.OpKind { return ops.OpKindSync }

type mergeContactPhonesArgs struct {
	pb.MergePhonesRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *mergeContactPhonesOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv mergeContactPhonesArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("mergeContactPhones", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.MergePhones(ctx, argv.Token, &argv.MergePhonesRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("mergeContactPhones: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── mergeContactVariables ─────────────────────────────────────────────────────

type mergeContactVariablesOp struct{ client domcontacts.Client }

func (o *mergeContactVariablesOp) Kind() ops.OpKind { return ops.OpKindSync }

type mergeContactVariablesArgs struct {
	pb.MergeVariablesRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *mergeContactVariablesOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv mergeContactVariablesArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("mergeContactVariables", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.MergeVariables(ctx, argv.Token, &argv.MergeVariablesRequest)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("mergeContactVariables: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── linkContact ───────────────────────────────────────────────────────────────

type linkContactOp struct{ deps LinkDeps }

func (o *linkContactOp) Kind() ops.OpKind { return ops.OpKindSync }

type linkContactArgs struct {
	SessionId  string  `json:"sessionId"`
	ContactId  int64   `json:"contactId"`
	ContactIds []int64 `json:"contactIds"`
	Channel    string  `json:"channel"`
}

func (o *linkContactOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv linkContactArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	conn := connctx.ConnectionFromContext(ctx)
	if conn == nil {
		return ops.OpOutput{}, fmt.Errorf("linkContact: no connection in context")
	}

	contactIds := argv.ContactIds
	if argv.ContactId != 0 {
		contactIds = []int64{argv.ContactId}
	}
	if len(contactIds) == 0 {
		return ops.OpOutput{}, fmt.Errorf("linkContact: Contact is required")
	}

	sessionId := argv.SessionId
	if sessionId == "" {
		sessionId = conn.Id()
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

	var linkErr error
	switch channel {
	case model.ConnectionTypeCall:
		linkErr = o.deps.CallSetContactId(conn.DomainId(), sessionId, contactIds[0])
	case model.ConnectionTypeChat:
		linkErr = o.deps.ContactLinkToChat(ctx, sessionId, fmt.Sprintf("%v", contactIds[0]))
	case model.ConnectionTypeEmail:
		linkErr = o.deps.MailSetContacts(ctx, conn.DomainId(), sessionId, contactIds)
	default:
		return ops.OpOutput{}, fmt.Errorf("linkContact: unsupported channel type %v", channel)
	}

	if linkErr != nil {
		return ops.OpOutput{}, fmt.Errorf("linkContact: %s", linkErr.Error())
	}

	return ops.OpOutput{
		SetVars: map[string]string{
			"wbt_contact_id": fmt.Sprintf("%d", contactIds[0]),
		},
	}, nil
}
