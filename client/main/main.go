package main

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type Client struct {
	client    flow.FlowServiceClient
	conn      flow.FlowService_RouteFlowClient
	variables map[string]string //Client variables (FS Global ?)
}

type Connection struct {
	sync.RWMutex
	queue map[int]chan interface{}
	seq   int
	conn  flow.FlowService_RouteFlowClient
}

type Plugin interface {
	Execute(appId string, args interface{}) error
	Close() error
}

func main() {
	cli, err := NewClient("10.10.10.25:8043", 1, 2)
	if err != nil {
		panic(err.Error())
	}
	for {
		time.Sleep(time.Millisecond * 300)
		c, e := cli.RouteFlow(context.Background())
		if e != nil {
			fmt.Println(e)
			return
		}
		c.Close()
		break
	}
}

func NewClient(addr string, domainId int64, schemaId int) (*Client, error) {
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

func (c *Connection) send(out chan interface{}, req *flow.RouteFlowRequest) error {
	c.Lock()
	c.seq++

	c.queue[c.seq] = out
	req.Seq = int32(c.seq)

	c.Unlock()
	return c.conn.Send(req)
}

func (c *Client) RouteFlow(ctx context.Context) (*Connection, error) {
	conn, err := c.client.RouteFlow(ctx)
	if err != nil {
		return nil, err
	}

	out := &Connection{
		conn:  conn,
		queue: make(map[int]chan interface{}),
	}

	err = conn.Send(&flow.RouteFlowRequest{
		RouteRequest: &flow.RouteFlowRequest_Request_{
			Request: &flow.RouteFlowRequest_Request{
				SchemaId: 2,
				DomainId: 1,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	var rec *flow.RouteFlowResponse
	rec, err = conn.Recv()
	if err != nil {
		return nil, err
	}

	fmt.Println(rec.Resp)

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
