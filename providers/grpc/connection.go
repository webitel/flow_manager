package grpc

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type Connection struct {
	id        string
	nodeId    string
	variables map[string]string
	stop      chan struct{}
	ctx       context.Context

	result  chan interface{}
	request interface{}

	event chan interface{}
}

func NewConnection(ctx context.Context, variables map[string]string) model.Connection {
	return newConnection(ctx, variables)
}

func newConnection(ctx context.Context, variables map[string]string) *Connection {
	return &Connection{
		ctx:       ctx,
		variables: variables,
		stop:      make(chan struct{}),
		result:    make(chan interface{}),
	}
}

func (c *Connection) ParseText(text string) string {
	return "FIXME"
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) NodeId() string {
	return c.nodeId
}

func (c *Connection) Execute(ctx context.Context, name string, args interface{}) (model.Response, *model.AppError) {
	return model.CallResponseOK, nil
}

func (c *Connection) Close() *model.AppError {
	c.ctx.Done()
	return nil
}

func (c *Connection) DomainId() int64 {
	return 0
}

func (c Connection) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func (c *Connection) Set(vars model.Variables) (model.Response, *model.AppError) {
	return model.CallResponseOK, nil
}

func (c *Connection) Get(key string) (string, bool) {
	return "", false
}

//fixme
func test() {
	a := func(c model.GRPCConnection) {}
	a(&Connection{})
}
