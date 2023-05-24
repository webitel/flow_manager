package email

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/webitel/flow_manager/model"
)

type connection struct {
	id        string
	profile   *Profile
	email     *model.Email
	variables model.Variables
	ctx       context.Context
	sync.RWMutex
}

func NewConnection(profile *Profile, email *model.Email) *connection {
	c := &connection{
		id:        email.MessageId,
		profile:   profile,
		email:     email,
		variables: make(map[string]interface{}),
		ctx:       context.Background(),
	}

	c.variables["message_id"] = email.MessageId
	c.variables["reply_to"] = strings.Join(email.ReplyTo, ",")
	c.variables["from"] = strings.Join(email.From, ",")
	c.variables["cc"] = strings.Join(email.CC, ",")
	c.variables["sender"] = strings.Join(email.Sender, ",")
	c.variables["in_reply_to"] = email.InReplyTo
	c.variables["body"] = string(email.Body)
	if email.HtmlBody != nil {
		c.variables["body_html"] = string(email.HtmlBody)
	}
	c.variables["subject"] = fmt.Sprintf("%v", email.Subject)
	c.variables["id"] = fmt.Sprintf("%d", email.Id)

	if len(email.Attachments) != 0 {
		if attachments, err := json.Marshal(email.Attachments); err == nil {
			c.variables["attachments"] = string(attachments)
		}
	}

	return c
}

func (c *connection) Email() *model.Email {
	return c.email
}

func (c *connection) Type() model.ConnectionType {
	return model.ConnectionTypeEmail
}

func (c *connection) Id() string {
	return c.id
}

func (c *connection) NodeId() string {
	//TODO PROFILE NAME
	return c.id
}

func (c *connection) Get(key string) (string, bool) {
	v, ok := c.variables[key]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", v), true
}

func (c *connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()
	for k, v := range vars {
		c.variables[k] = v
	}

	return model.CallResponseOK, nil //TODO
}

func (c *connection) ParseText(text string) string {
	//TODO
	return text
}

func (c *connection) SchemaId() int {
	return c.profile.flowId
}

func (c *connection) Close() *model.AppError {
	return nil
}

func (c *connection) DomainId() int64 {
	return c.profile.DomainId
}

func (c *connection) Context() context.Context {
	return c.ctx
}

func (c *connection) Variables() map[string]string {
	vars := make(map[string]string)
	for k, v := range c.variables {
		vars[k] = fmt.Sprintf("%v", v)

	}
	return vars
}

//fixme
func test() {
	a := func(c model.EmailConnection) {}
	a(&connection{})
}
