package contacts

import (
	"context"
	"net/http"
	"sync"

	"github.com/webitel/wlog"

	gen "github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/infra/grpcdial"
	"github.com/webitel/flow_manager/model"
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

func (c *Client) Locate(ctx context.Context, token string, req *gen.LocateContactRequest) (*gen.Contact, *model.AppError) {
	res, err := c.Contacts().LocateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "LocateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) Create(ctx context.Context, token string, req *gen.InputContactRequest) (*gen.Contact, *model.AppError) {
	res, err := c.Contacts().CreateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "CreateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) Search(ctx context.Context, token string, req *gen.SearchContactsRequest) (*gen.ContactList, *model.AppError) {
	res, err := c.Contacts().SearchContacts(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "SearchContacts", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) SearchNA(ctx context.Context, req *gen.SearchContactsNARequest) (*gen.ContactList, *model.AppError) {
	res, err := c.Contacts().SearchContactsNA(ctx, req)
	if err != nil {
		return nil, model.NewAppError("contacts", "SearchContactsNA", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) Update(ctx context.Context, token string, req *gen.InputContactRequest) (*gen.Contact, *model.AppError) {
	res, err := c.Contacts().UpdateContact(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "UpdateContact", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) MergeVariables(ctx context.Context, token string, req *gen.MergeVariablesRequest) (*gen.VariableList, *model.AppError) {
	res, err := c.Variables().MergeVariables(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "MergeContactVariables", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) MergePhones(ctx context.Context, token string, req *gen.MergePhonesRequest) (*gen.PhoneList, *model.AppError) {
	res, err := c.Phone().MergePhones(c.contacts.WithToken(ctx, token), req)
	if err != nil {
		return nil, model.NewAppError("contacts", "MergeContactPhones", nil, err.Error(), http.StatusInternalServerError)
	}

	return res, nil
}

func (c *Client) Contacts() gen.ContactsClient {
	return c.contacts.API
}

func (c *Client) Phone() gen.PhonesClient {
	return c.phone.API
}

func (c *Client) Variables() gen.VariablesClient {
	return c.variables.API
}

func (c *Client) Stop() {}
