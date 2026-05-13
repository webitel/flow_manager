package grpcdial

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/flow_manager/internal/infrastructure/resolver"
)

const (
	TokenHeaderName = "x-webitel-access"
)

type Client[T any] struct {
	conn *grpc.ClientConn
	API  T
}

var conns sync.Map

func NewClient[T any](consulTarget, service string, api func(conn grpc.ClientConnInterface) T) (*Client[T], error) {
	var (
		conn *grpc.ClientConn
		err  error
	)

	dsn := fmt.Sprintf("wbt://%s/%s?wait=15s", consulTarget, service)

	if c, ok := conns.Load(dsn); ok {
		conn = c.(*grpc.ClientConn)
	} else {
		conn, err = grpc.NewClient(dsn,
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "wbt_round_robin"}`),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}

		conns.Store(dsn, conn)
	}

	return &Client[T]{
		conn: conn,
		API:  api(conn),
	}, nil
}

func (c *Client[T]) StaticHost(ctx context.Context, name string) context.Context {
	return StaticHost(ctx, name)
}

func (c *Client[T]) WithToken(ctx context.Context, token string) context.Context {
	return WithToken(ctx, token)
}

func StaticHost(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, resolver.StaticHostKey{}, resolver.StaticHost{Name: name})
}

func WithToken(ctx context.Context, token string) context.Context {
	header := metadata.New(map[string]string{TokenHeaderName: token})

	return metadata.NewOutgoingContext(ctx, header)
}

func (c *Client[T]) Close() error {
	return c.conn.Close()
}

func (c *Client[T]) Start() error {
	return nil
}

// NewClientWithOpts creates a Client without shared connection caching.
// Use this when custom DialOptions (e.g. TLS credentials) are required.
func NewClientWithOpts[T any](consulTarget, service string, api func(conn grpc.ClientConnInterface) T, opts ...grpc.DialOption) (*Client[T], error) {
	dsn := fmt.Sprintf("wbt://%s/%s?wait=15s", consulTarget, service)
	base := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "wbt_round_robin"}`),
	}
	conn, err := grpc.NewClient(dsn, append(base, opts...)...)
	if err != nil {
		return nil, err
	}
	return &Client[T]{
		conn: conn,
		API:  api(conn),
	}, nil
}
