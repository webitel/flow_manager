package main

import (
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type Client struct {
	client    flow.FlowServiceClient
	variables map[string]string //Client variables (FS Global ?)
}

type Connection struct {
	sync.RWMutex
	queue map[int]chan interface{}
	seq   int
}

func main() {
	_, err := NewClient("10.10.10.25:8043", 1, 2)
	if err != nil {
		panic(err.Error())
	}
	for {
		time.Sleep(time.Millisecond * 300)
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

func (c *Connection) Execute() error {
	return nil
}

func (c *Connection) Close() error {
	return nil
}
