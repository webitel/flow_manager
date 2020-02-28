package client

import (
	"github.com/webitel/engine/auth_manager/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type FlowClient interface {
	Name() string
	Close() error
	Ready() bool
}

const (
	ConnectionTimeout = 2 * time.Second
)

type flowConnection struct {
	name   string
	host   string
	client *grpc.ClientConn
	api    api.AuthClient
}

func NewAuthServiceConnection(name, url string) (FlowClient, error) {
	var err error
	connection := &flowConnection{
		name: name,
		host: url,
	}

	connection.client, err = grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(ConnectionTimeout))

	if err != nil {
		return nil, err
	}

	connection.api = api.NewAuthClient(connection.client)

	return connection, nil
}

func (c *flowConnection) Ready() bool {
	switch c.client.GetState() {
	case connectivity.Idle, connectivity.Ready:
		return true
	}
	return false
}

func (c *flowConnection) Name() string {
	return c.name
}

func (c *flowConnection) Close() error {
	err := c.client.Close()
	if err != nil {
		return ErrInternal
	}
	return nil
}
