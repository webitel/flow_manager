package contacts

import (
	"context"

	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/model"
)

type Client interface {
	Create(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError)
	Locate(ctx context.Context, token string, req *contacts.LocateContactRequest) (*contacts.Contact, *model.AppError)
	Search(ctx context.Context, token string, req *contacts.SearchContactsRequest) (*contacts.ContactList, *model.AppError)
	SearchNA(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, *model.AppError)
	Update(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, *model.AppError)

	MergeVariables(ctx context.Context, token string, req *contacts.MergeVariablesRequest) (*contacts.VariableList, *model.AppError)
	MergePhones(ctx context.Context, token string, req *contacts.MergePhonesRequest) (*contacts.PhoneList, *model.AppError)
}
