package bsfx

import (
	"context"

	"github.com/webitel/engine/pkg/presign"
	"go.uber.org/fx"

	fmgrpc "github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	bsruntime "github.com/webitel/flow_manager/internal/bootstrap/runtime"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	clusterPkg "github.com/webitel/flow_manager/internal/bootstrap/cluster"
	bootstrapServers "github.com/webitel/flow_manager/internal/bootstrap/servers"
)

// RegisterInfraHooks registers the startup/shutdown sequence that requires
// ordering guarantees:
//  1. Bind all protocol servers (so grpcServer.Host()/Port() are valid).
//  2. Start cluster registration with the bound gRPC address.
//  3. Connect ChatManager and gRPC server to the cluster discovery.
//  4. Wire presign certificate into SchemaAdapter (optional).
//  5. Pre-warm the timezone cache.
func RegisterInfraHooks(
	lc fx.Lifecycle,
	cfg *bscfg.Config,
	id AppID,
	srvs bootstrapServers.Servers,
	chatMgr *fmgrpc.ChatManager,
	fm *bsruntime.FlowManager,
) {
	var cluster *clusterPkg.Cluster

	// Step 1–3: servers → cluster → chatManager → gRPC cluster wiring.
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if err := srvs.Register(); err != nil {
				return err
			}
			cluster = clusterPkg.New(
				string(id),
				cfg.DiscoverySettings.Url,
				srvs.Grpc.Host(),
				srvs.Grpc.Port(),
			)
			if err := cluster.Start(); err != nil {
				return err
			}
			if err := chatMgr.Start(cluster.Discovery); err != nil {
				return err
			}
			return srvs.Grpc.Cluster(cluster.Discovery)
		},
		OnStop: func(_ context.Context) error {
			chatMgr.Stop()
			if cluster != nil {
				cluster.Stop()
			}
			srvs.Stop()
			return nil
		},
	})

	// Step 4: presign certificate (optional).
	if cfg.PreSignedCertificateLocation != "" {
		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				cert, err := presign.NewPreSigned(cfg.PreSignedCertificateLocation)
				if err != nil {
					return err
				}
				fm.SetCert(cert)
				return nil
			},
		})
	}

	// Step 5: timezone cache warm-up.
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return fm.InitCacheTimezones()
		},
	})
}
