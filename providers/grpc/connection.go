package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"github.com/webitel/wlog"
	"io"
)

type Connection struct {
	id        string
	nodeId    string
	req       *flow.RouteFlowRequest_Request_
	variables map[string]string
	queue     map[int]interface{}
	stream    flow.FlowService_RouteFlowServer
}

func NewConnection(stream flow.FlowService_RouteFlowServer) model.Connection {
	return NewConnection(stream)
}

func newConnection(stream flow.FlowService_RouteFlowServer) *Connection {
	return &Connection{
		variables: make(map[string]string),
		stream:    stream,
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
	fmt.Println(name, args)
	return model.CallResponseOK, nil
}

func (c *Connection) Close() *model.AppError {
	c.stream.Send(&flow.RouteFlowResponse{
		Resp: &flow.RouteFlowResponse_Error{Error: &flow.ErrorStatus{
			Message: "close",
		}},
	})
	return nil
}

func (c *Connection) SchemaId() int {
	return int(c.req.Request.SchemaId)
}

func (c *Connection) SchemaUpdatedAt() int64 {
	return c.req.Request.SchemaUpdatedAt
}

func (c *Connection) DomainId() int {
	return int(c.req.Request.DomainId)
}

func (c Connection) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func (c *Connection) connect(stream flow.FlowService_RouteFlowServer) error {
	in, err := stream.Recv()
	if err != nil {
		return err
	}
	if req, ok := in.GetRouteRequest().(*flow.RouteFlowRequest_Request_); ok {
		c.req = req
	}

	return nil
}

func (s *server) RouteFlow(stream flow.FlowService_RouteFlowServer) error {
	wlog.Debug(fmt.Sprintf("receive new grpc connection "))
	defer wlog.Debug(fmt.Sprintf("close grpc connection "))

	conn := newConnection(stream)

	if err := conn.connect(stream); err != nil {
		return err
	}

	s.consume <- conn

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		fmt.Println(in)
	}

	return nil
}

func (s *server) CallCenter(stream flow.FlowService_CallCenterServer) error {
	return nil
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
