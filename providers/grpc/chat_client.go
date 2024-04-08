package grpc

import (
	"context"
	"time"

	"github.com/webitel/flow_manager/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"

	gogrpc "buf.build/gen/go/webitel/chat/grpc/go/_gogrpc"
)

var (
	ChatClientService = "webitel.chat.server"
	WatcherInterval   = 5 * 1000
)

type ChatClientConnection struct {
	id       string
	host     string
	client   *grpc.ClientConn
	api      gogrpc.ChatServiceClient
	messages gogrpc.MessagesClient
}

func NewChatClientConnection(id, url string) (*ChatClientConnection, error) {
	var err error
	connection := &ChatClientConnection{
		id:   id,
		host: url,
	}

	connection.client, err = grpc.Dial(
		url,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(time.Second*5),
		grpc.WithUnaryInterceptor(connection.UnaryClientInterceptor),
	)

	if err != nil {
		return nil, err
	}

	connection.api = gogrpc.NewChatServiceClient(connection.client)
	connection.messages = gogrpc.NewMessagesClient(connection.client)

	return connection, nil
}

func (cc ChatClientConnection) UnaryClientInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Create a new context with the token and make the first request
	serviceCtx := metadata.AppendToOutgoingContext(ctx, model.HeaderFromServiceName, model.AppServiceName)
	return invoker(serviceCtx, method, req, reply, conn, opts...)
}

func (cc *ChatClientConnection) Name() string {
	return cc.id
}

func (cc *ChatClientConnection) Ready() bool {
	switch cc.client.GetState() {
	case connectivity.Idle, connectivity.Ready:
		return true
	}
	return false
}

func (cc *ChatClientConnection) Close() error {
	return cc.client.Close()
}
