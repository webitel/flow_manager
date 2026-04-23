package contacts

import (
	"context"

	"github.com/webitel/flow_manager/gen/contacts"
)

type Client interface {
	Create(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, error)
	Locate(ctx context.Context, token string, req *contacts.LocateContactRequest) (*contacts.Contact, error)
	Search(ctx context.Context, token string, req *contacts.SearchContactsRequest) (*contacts.ContactList, error)
	SearchNA(ctx context.Context, req *contacts.SearchContactsNARequest) (*contacts.ContactList, error)
	Update(ctx context.Context, token string, req *contacts.InputContactRequest) (*contacts.Contact, error)

	MergeVariables(ctx context.Context, token string, req *contacts.MergeVariablesRequest) (*contacts.VariableList, error)
	MergePhones(ctx context.Context, token string, req *contacts.MergePhonesRequest) (*contacts.PhoneList, error)
}
