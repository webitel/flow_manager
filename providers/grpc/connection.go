package grpc

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"github.com/webitel/wlog"
	"io"
)

type connection struct {
	id        string
	nodeId    string
	variables map[string]string
	queue     map[int]interface{}
	stream    flow.FlowService_RouteFlowServer
}

func NewConnection(stream flow.FlowService_RouteFlowServer) model.Connection {
	return &connection{
		variables: make(map[string]string),
		stream:    stream,
	}
}

func (c *connection) ParseText(text string) string {
	return "FIXME"
}

func (c *connection) Id() string {
	return c.id
}

func (c *connection) NodeId() string {
	return c.nodeId
}

func (c *connection) Execute(string, interface{}) (model.Response, *model.AppError) {
	return nil, nil
}

func (c *connection) Close() *model.AppError {
	return nil
}

func (c connection) Type() model.ConnectionType {
	return model.ConnectionTypeGrpc
}

func (s *server) RouteFlow(stream flow.FlowService_RouteFlowServer) error {
	wlog.Debug(fmt.Sprintf("receive new grpc connection "))
	defer wlog.Debug(fmt.Sprintf("close grpc connection "))
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

func (c *connection) Set(vars model.Variables) (model.Response, *model.AppError) {
	return nil, nil
}

func (c *connection) Get(key string) (string, bool) {
	return "", false
}
