package client

import (
	"github.com/webitel/flow_manager/providers/grpc/flow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type fConnection struct {
	name   string
	host   string
	client *grpc.ClientConn

	queue flow.FlowServiceClient
}

func NewFlowConnection(name, url string) (*fConnection, error) {
	var err error
	connection := &fConnection{
		name: name,
		host: url,
	}

	connection.client, err = grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))

	if err != nil {
		return nil, err
	}

	connection.queue = flow.NewFlowServiceClient(connection.client)

	return connection, nil
}

func (conn *fConnection) Ready() bool {
	switch conn.client.GetState() {
	case connectivity.Idle, connectivity.Ready:
		return true
	}
	return false
}

func (conn *fConnection) Name() string {
	return conn.name
}

func (conn *fConnection) Close() error {
	err := conn.client.Close()
	if err != nil {
		return ErrInternal
	}
	return nil
}
