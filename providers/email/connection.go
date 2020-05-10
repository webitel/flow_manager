package email

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"sync"
)

type connection struct {
	id        string
	profile   *Profile
	email     *model.Email
	variables model.Variables
	sync.RWMutex
}

func NewConnection(profile *Profile, email *model.Email) *connection {
	c := &connection{
		id:        email.MessageId,
		profile:   profile,
		email:     email,
		variables: make(map[string]interface{}),
	}

	c.variables["from"] = fmt.Sprintf("%v", email.From)
	c.variables["body"] = fmt.Sprintf("%v", string(email.Body))

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

func (c *connection) Set(vars model.Variables) (model.Response, *model.AppError) {
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

func (c *connection) Close() *model.AppError {
	return nil
}
func (c *connection) DomainId() int64 {
	return 0
}

//fixme
func test() {
	a := func(c model.EmailConnection) {}
	a(&connection{})
}
