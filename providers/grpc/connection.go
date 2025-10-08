package grpc

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/webitel/wlog"

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

	result          chan interface{}
	cancel          context.CancelFunc
	exportVariables []string
	scope           model.Scope

	request interface{}

	event chan interface{}

	sync.RWMutex

	log *wlog.Logger
}

func newConnection(id string, domainId int64, flowId int, ctx context.Context, variables map[string]string, timeout time.Duration) *Connection {
	c := &Connection{
		id:        id,
		domainId:  domainId,
		schemaId:  flowId,
		variables: variables,
		stop:      make(chan struct{}),
		result:    make(chan interface{}),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "grpc"),
			wlog.String("id", id),
			wlog.Int64("domain_d", domainId),
			wlog.Int("schema_id", flowId),
		),
	}

	if timeout == 0 {
		timeout = timeoutFlowSchema
	}
	c.ctx, c.cancel = context.WithTimeout(ctx, timeout)

	return c
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
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

func (c *Connection) Type() model.ConnectionType {
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
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}

func (c *Connection) Scope() model.Scope {
	return c.scope
}

func (c *Connection) Export(ctx context.Context, vars []string) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()
	for _, v := range vars {
		if v == "" {
			continue
		}
		c.exportVariables = append(c.exportVariables, v)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) DumpExportVariables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	var res map[string]string
	if len(c.exportVariables) > 0 {
		res = make(map[string]string)
		for _, v := range c.exportVariables {
			res[v], _ = c.Get(v)
		}
	}
	return res
}

// fixme
func test() {
	a := func(c model.GRPCConnection) {}
	a(&Connection{})
}
