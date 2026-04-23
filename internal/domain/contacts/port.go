package contacts

import (
	"context"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

type Client interface {
	CreateContact(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError)
	LocateContact(ctx context.Context, token string, req *contacts.LocateContactRequest) (*contacts.Contact, *model.AppError)
	SearchContacts(ctx context.Context, token string, req *contacts.SearchContactsRequest) (*contacts.ContactList, *model.AppError)
	SearchContactsNA(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError)
	UpdateContact(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError)

	MergeContactVariables(ctx context.Context, token string, req *contacts.MergeVariablesRequest) (*contacts.VariableList, *model.AppError)
	MergeContactPhones(ctx context.Context, token string, req *contacts.MergePhonesRequest) (*contacts.PhoneList, *model.AppError)
}
