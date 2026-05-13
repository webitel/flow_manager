package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/webitel/wlog"

	"github.com/tidwall/gjson"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

type Connection struct {
	id        string
	ctx       context.Context
	domainId  int64
	schemaId  int
	variables map[string]string
	sync.RWMutex
	log *wlog.Logger
}

func newConnection(c flow.ChannelExec) flow.Connection {
	id := utils.NewId()
	conn := &Connection{
		id:        id,
		ctx:       context.Background(),
		domainId:  c.DomainId,
		variables: toVariables(c.Variables),
		schemaId:  c.SchemaId,
		RWMutex:   sync.RWMutex{},
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "channel"),
			wlog.String("channel_id", id),
			wlog.Int64("domain_id", c.DomainId),
			wlog.Int("schema_id", c.SchemaId),
		),
	}
	if conn.variables == nil {
		conn.variables = make(map[string]string)
	}
	return conn
}

func (c *Connection) Type() flow.ConnectionType {
	return flow.ConnectionTypeChannel
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
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

func (c *Connection) Set(ctx context.Context, vars flow.Variables) (flow.Response, error) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return call.CallResponseOK, nil
}

func (c *Connection) ParseText(text string, ops ...flow.ParseOption) string {
	return flow.ParseText(c, text, ops...)
}

func (c *Connection) Close() error {
	return nil
}

func (c *Connection) Variables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}

func toVariables(in map[string]json.RawMessage) map[string]string {
	vars := make(map[string]string)

	for k, v := range in {
		vars[k] = string(v)
	}

	return vars
}

// OnInboundMessage satisfies sessionmgr.Connection. Channel connections are
// ephemeral and never receive inbound messages after flow start.
func (c *Connection) OnInboundMessage(_ func(string)) (unregister func()) {
	return func() {}
}
