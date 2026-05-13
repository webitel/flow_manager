package contacts

import (
	"context"

	contacts2 "github.com/webitel/flow_manager/api/gen/contacts"
)

type Client interface {
	Create(ctx context.Context, token string, req *contacts2.InputContactRequest) (*contacts2.Contact, error)
	Locate(ctx context.Context, token string, req *contacts2.LocateContactRequest) (*contacts2.Contact, error)
	Search(ctx context.Context, token string, req *contacts2.SearchContactsRequest) (*contacts2.ContactList, error)
	SearchNA(ctx context.Context, req *contacts2.SearchContactsNARequest) (*contacts2.ContactList, error)
	Update(ctx context.Context, token string, req *contacts2.InputContactRequest) (*contacts2.Contact, error)

	MergeVariables(ctx context.Context, token string, req *contacts2.MergeVariablesRequest) (*contacts2.VariableList, error)
	MergePhones(ctx context.Context, token string, req *contacts2.MergePhonesRequest) (*contacts2.PhoneList, error)
}
