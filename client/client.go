package client

import (
	"context"
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"google.golang.org/grpc"
	"time"
)

type Client struct {
	client    flow.FlowServiceClient
	variables map[string]string //Client variables (FS Global ?)
}

type Connection struct {
	conn flow.FlowService_RouteFlowClient
}

type Plugin interface {
	Execute(appId string, args interface{}) error
	Close() error
}

func init2() {
	cli, err := NewClient("10.10.10.25:8043")
	if err != nil {
		panic(err.Error())
	}
	go func(cli *Client) {
		for {
			time.Sleep(time.Second * 3)
			c, _ := cli.RouteFlow(context.Background())
			c.Close()
		}
	}(cli)
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	cli := &Client{
		client:    flow.NewFlowServiceClient(conn),
		variables: make(map[string]string),
	}
	return cli, nil
}

func (c *Client) RouteFlow(ctx context.Context) (*Connection, error) {
	conn, err := c.client.RouteFlow(ctx)
	if err != nil {
		return nil, err
	}

	out := &Connection{
		conn: conn,
	}

	return out, nil
}

func (c *Connection) Execute() error {
	return c.conn.Send(&flow.RouteFlowRequest{
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	})
}

func (c *Connection) Close() error {
	return c.conn.CloseSend()
}
