package call

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/webitel/wlog"

	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

// GrpcConnection is the transport-level connection for gRPC-driven call flows.
type GrpcConnection struct {
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
	scope           flow.Scope

	request interface{}

	event chan interface{}

	sync.RWMutex

	log *wlog.Logger
}

func newGrpcConnection(id string, domainId int64, flowId int, ctx context.Context, variables map[string]string, timeout time.Duration) *GrpcConnection {
	c := &GrpcConnection{
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
		timeout = timeoutGrpcSchema
	}
	c.ctx, c.cancel = context.WithTimeout(ctx, timeout)

	return c
}

func (c *GrpcConnection) Log() *wlog.Logger {
	return c.log
}

func (c *GrpcConnection) Context() context.Context {
	return c.ctx
}

func (c *GrpcConnection) ParseText(text string, ops ...flow.ParseOption) string {
	return flow.ParseText(c, text, ops...)
}

func (c *GrpcConnection) Result(result interface{}) {
	c.result <- result
}

func (c *GrpcConnection) Id() string {
	return c.id
}

func (c *GrpcConnection) NodeId() string {
	return c.nodeId
}

func (c *GrpcConnection) SchemaId() int {
	return c.schemaId
}

func (c *GrpcConnection) Close() error {
	c.cancel()
	return nil
}

func (c *GrpcConnection) DomainId() int64 {
	return c.domainId
}

func (c *GrpcConnection) Type() flow.ConnectionType {
	return flow.ConnectionTypeGrpc
}

func (c *GrpcConnection) Set(ctx context.Context, vars flow.Variables) (flow.Response, error) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v)
	}

	return calldomain.CallResponseOK, nil
}

func (c *GrpcConnection) Get(name string) (string, bool) {
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

func (c *GrpcConnection) Variables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}

func (c *GrpcConnection) Scope() flow.Scope {
	return c.scope
}

func (c *GrpcConnection) Export(ctx context.Context, vars []string) (flow.Response, error) {
	c.Lock()
	defer c.Unlock()
	for _, v := range vars {
		if v == "" {
			continue
		}
		c.exportVariables = append(c.exportVariables, v)
	}

	return calldomain.CallResponseOK, nil
}

func (c *GrpcConnection) DumpExportVariables() map[string]string {
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

// OnInboundMessage satisfies sessionmgr.Connection. GRPC connections are
// ephemeral and never receive inbound messages after flow start.
func (c *GrpcConnection) OnInboundMessage(_ func(string)) (unregister func()) {
	return func() {}
}
