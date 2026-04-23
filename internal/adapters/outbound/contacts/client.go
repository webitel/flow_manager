package contacts

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	gen "github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const serviceName = "go.webitel.app"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	contacts   *grpcdial.Client[gen.ContactsClient]
	phone      *grpcdial.Client[gen.PhonesClient]
	variables  *grpcdial.Client[gen.VariablesClient]
}

func New(consulAddr string) *Client {
	return &Client{consulAddr: consulAddr}
}

func (c *Client) Start() error {
	wlog.Debug("starting " + serviceName + " client")
	var err error
	c.startOnce.Do(func() {
		c.contacts, err = grpcdial.NewClient(c.consulAddr, serviceName, gen.NewContactsClient)
		if err != nil {
			return
		}
		c.phone, err = grpcdial.NewClient(c.consulAddr, serviceName, gen.NewPhonesClient)
		if err != nil {
			return
		}
		c.variables, err = grpcdial.NewClient(c.consulAddr, serviceName, gen.NewVariablesClient)
		if err != nil {
			return
		}
	})
	return err
}

func (c *Client) Locate(ctx context.Context, token string, req *gen.LocateContactRequest) (*gen.Contact, error) {
	res, err := c.contacts.API.LocateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Locate: %w", err)
	}
	return res, nil
}

func (c *Client) Create(ctx context.Context, token string, req *gen.InputContactRequest) (*gen.Contact, error) {
	res, err := c.contacts.API.CreateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Create: %w", err)
	}
	return res, nil
}

func (c *Client) Search(ctx context.Context, token string, req *gen.SearchContactsRequest) (*gen.ContactList, error) {
	res, err := c.contacts.API.SearchContacts(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Search: %w", err)
	}
	return res, nil
}

func (c *Client) SearchNA(ctx context.Context, req *gen.SearchContactsNARequest) (*gen.ContactList, error) {
	res, err := c.contacts.API.SearchContactsNA(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("contacts.SearchNA: %w", err)
	}
	return res, nil
}

func (c *Client) Update(ctx context.Context, token string, req *gen.InputContactRequest) (*gen.Contact, error) {
	res, err := c.contacts.API.UpdateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.Update: %w", err)
	}
	return res, nil
}

func (c *Client) MergeVariables(ctx context.Context, token string, req *gen.MergeVariablesRequest) (*gen.VariableList, error) {
	res, err := c.variables.API.MergeVariables(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.MergeVariables: %w", err)
	}
	return res, nil
}

func (c *Client) MergePhones(ctx context.Context, token string, req *gen.MergePhonesRequest) (*gen.PhoneList, error) {
	res, err := c.phone.API.MergePhones(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, fmt.Errorf("contacts.MergePhones: %w", err)
	}
	return res, nil
}

func (c *Client) Stop() {}
