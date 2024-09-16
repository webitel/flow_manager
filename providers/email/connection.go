package email

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/webitel/wlog"
	"strings"
	"sync"

	"github.com/webitel/flow_manager/model"
)

type PKey struct {
	Id        int
	UpdatedAt int64
	FlowId    int
	DomainId  int64
}

type connection struct {
	id        string
	srv       *MailServer
	email     *model.Email
	variables model.Variables
	ctx       context.Context
	pkey      PKey
	sync.RWMutex
	log *wlog.Logger
}

func NewConnection(srv *MailServer, pkey PKey, email *model.Email) *connection {
	c := &connection{
		id:        email.MessageId,
		srv:       srv,
		pkey:      pkey,
		email:     email,
		variables: make(map[string]interface{}),
		ctx:       context.Background(),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("message_id", email.MessageId),
			wlog.Int64("email_id", email.Id),
			wlog.Any("from", email.From),
		),
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

func (c *connection) Log() *wlog.Logger {
	return c.log
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

func (c *connection) Get(name string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(name, ".")
	if idx > 0 {
		nameRoot := name[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(fmt.Sprintf("%v", v)), name[idx+1:]).String(), true
		}
	}
	v, ok := c.variables[name]
	return fmt.Sprintf("%v", v), ok
}

func (c *connection) GetProfile() (*Profile, *model.AppError) {
	return c.srv.GetProfile(c.pkey.Id, c.pkey.UpdatedAt)
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
	return c.pkey.FlowId
}

func (c *connection) Close() *model.AppError {
	return nil
}

func (c *connection) DomainId() int64 {
	return c.pkey.DomainId
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

// fixme
func test() {
	a := func(c model.EmailConnection) {}
	a(&connection{})
}
