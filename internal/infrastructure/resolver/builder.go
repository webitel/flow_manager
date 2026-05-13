package resolver

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"

	"github.com/webitel/wlog"
)

// schemeName for the urls
// All target URLs like 'consul://.../...' will be resolved by this resolver
const schemeName = "wbt"

// builder implements resolver.Builder and use for constructing all consul resolvers
type builder struct {
	log *wlog.Logger
}

var consulClients sync.Map

func (b *builder) Build(url resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	dsn := strings.Join([]string{schemeName + ":/", url.URL.Host + url.URL.Path + "?" + url.URL.RawQuery}, "/")

	tgt, err := parseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("wrong consul URL: %w", err)
	}

	cfg := tgt.consulConfig()

	var cli *api.Client

	if c, ok := consulClients.Load(cfg.Address); ok {
		cli = c.(*api.Client)
	} else {
		cli, err = api.NewClient(cfg)
		consulClients.Store(cfg.Address, cli)
	}

	if err != nil {
		return nil, fmt.Errorf("couldn't connect to the Consul API: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	pipe := make(chan []serviceMeta)
	go watchConsulService(ctx, cli.Health(), tgt, pipe)
	go populateEndpoints(ctx, cc, pipe)

	return &resolvr{cancelFunc: cancel}, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return schemeName
}
