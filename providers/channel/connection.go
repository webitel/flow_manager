package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/tidwall/gjson"

	"github.com/webitel/flow_manager/model"
)

var compileVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

type Connection struct {
	id        string
	ctx       context.Context
	domainId  int64
	schemaId  int
	variables map[string]string
	sync.RWMutex
}

func newConnection(c model.ChannelExec) model.Connection {
	conn := &Connection{
		id:        model.NewId(),
		ctx:       context.Background(),
		domainId:  c.DomainId,
		variables: toVariables(c.Variables),
		schemaId:  c.SchemaId,
		RWMutex:   sync.RWMutex{},
	}
	if conn.variables == nil {
		conn.variables = make(map[string]string)
	}
	return conn
}

func (c Connection) Type() model.ConnectionType {
	return model.ConnectionTypeChannel
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) SchemaId() int {
	return c.schemaId
}

func (c *Connection) NodeId() string {
	return ""
}

func (c *Connection) DomainId() int64 {
	return c.domainId
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(key, ".")
	if idx > 0 {
		nameRoot := key[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(v), key[idx+1:]).String(), true
		}
	}
	v, ok := c.variables[key]
	return v, ok
}

func (c *Connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return model.CallResponseOK, nil
}

func (c *Connection) ParseText(text string) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		return
	})

	return text
}

func (c *Connection) Close() *model.AppError {
	return nil
}

func (c *Connection) Variables() map[string]string {
	return c.variables
}

func toVariables(in map[string]json.RawMessage) map[string]string {
	vars := make(map[string]string)

	for k, v := range in {
		vars[k] = string(v)
	}

	return vars
}
