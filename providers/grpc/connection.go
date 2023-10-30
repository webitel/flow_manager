package grpc

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/webitel/flow_manager/model"
)

type Connection struct {
	id        string
	nodeId    string
	domainId  int64
	schemaId  int
	variables map[string]string
	stop      chan struct{}
	ctx       context.Context

	result chan interface{}
	cancel context.CancelFunc

	request interface{}

	event chan interface{}

	sync.RWMutex
}

var compileVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

func newConnection(ctx context.Context, variables map[string]string, timeout time.Duration) *Connection {
	c := &Connection{
		variables: variables,
		stop:      make(chan struct{}),
		result:    make(chan interface{}),
	}

	if timeout == 0 {
		timeout = timeoutFlowSchema
	}
	c.ctx, c.cancel = context.WithTimeout(ctx, timeout)

	return c
}

func (c *Connection) Context() context.Context {
	return c.ctx
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

func (c *Connection) Result(result interface{}) {
	c.result <- result
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) NodeId() string {
	return c.nodeId
}

func (c *Connection) SchemaId() int {
	return c.schemaId
}

func (c *Connection) Close() *model.AppError {
	c.cancel()
	return nil
}

func (c *Connection) DomainId() int64 {
	return c.domainId
}

func (c Connection) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func (c *Connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return model.CallResponseOK, nil
}

func (c *Connection) Get(name string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(name, ".")
	if idx > 0 {
		nameRoot := name[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(v), name[idx+1:]).String(), true
		}
	}
	v, ok := c.variables[name]
	return v, ok
}

func (c *Connection) Variables() map[string]string {
	return c.variables
}

// fixme
func test() {
	a := func(c model.GRPCConnection) {}
	a(&Connection{})
}
