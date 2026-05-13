package contacts

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	contacts2 "github.com/webitel/flow_manager/api/gen/contacts"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const serviceName = "go.webitel.app"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	contacts   *grpcdial.Client[contacts2.ContactsClient]
	phone      *grpcdial.Client[contacts2.PhonesClient]
	variables  *grpcdial.Client[contacts2.VariablesClient]
}

func New(consulAddr string) *Client {
	return &Client{consulAddr: consulAddr}
}

func (c *Client) Start() error {
	wlog.Debug("starting " + serviceName + " client")
	var err error
	c.startOnce.Do(func() {
		c.contacts, err = grpcdial.NewClient(c.consulAddr, serviceName, contacts2.NewContactsClient)
		if err != nil {
			return
		}
		c.phone, err = grpcdial.NewClient(c.consulAddr, serviceName, contacts2.NewPhonesClient)
		if err != nil {
			return
		}
		c.variables, err = grpcdial.NewClient(c.consulAddr, serviceName, contacts2.NewVariablesClient)
		if err != nil {
			return
		}
	})
	return err
}

func (c *Client) Locate(ctx context.Context, token string, req *contacts2.LocateContactRequest) (*contacts2.Contact, error) {
	res, err := c.contacts.API.LocateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Locate: %w", err)
	}
	return res, nil
}

func (c *Client) Create(ctx context.Context, token string, req *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	res, err := c.contacts.API.CreateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Create: %w", err)
	}
	return res, nil
}

func (c *Client) Search(ctx context.Context, token string, req *contacts2.SearchContactsRequest) (*contacts2.ContactList, error) {
	res, err := c.contacts.API.SearchContacts(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Search: %w", err)
	}
	return res, nil
}

func (c *Client) SearchNA(ctx context.Context, req *contacts2.SearchContactsNARequest) (*contacts2.ContactList, error) {
	res, err := c.contacts.API.SearchContactsNA(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("contacts.SearchNA: %w", err)
	}
	return res, nil
}

func (c *Client) Update(ctx context.Context, token string, req *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	res, err := c.contacts.API.UpdateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Update: %w", err)
	}
	return res, nil
}

func (c *Client) MergeVariables(ctx context.Context, token string, req *contacts2.MergeVariablesRequest) (*contacts2.VariableList, error) {
	res, err := c.variables.API.MergeVariables(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.MergeVariables: %w", err)
	}
	return res, nil
}

func (c *Client) MergePhones(ctx context.Context, token string, req *contacts2.MergePhonesRequest) (*contacts2.PhoneList, error) {
	res, err := c.phone.API.MergePhones(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.MergePhones: %w", err)
	}
	return res, nil
}

func (c *Client) Stop() {}
