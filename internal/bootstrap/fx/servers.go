package bsfx

import (
	"crypto/tls"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/adapters/inbound/channel"
	"github.com/webitel/flow_manager/internal/adapters/inbound/email"
	"github.com/webitel/flow_manager/internal/adapters/inbound/fs"
	"github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	"github.com/webitel/flow_manager/internal/adapters/inbound/im"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	bootstrapServers "github.com/webitel/flow_manager/internal/bootstrap/servers"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/internal/usecase/callback"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

func NewCallbackResolver() *callback.Resolver {
	return callback.New()
}

func NewTLSConfig(cfg *model.Config) (*tls.Config, error) {
	return bscfg.LoadTLSCreds(cfg.Tls)
}

func NewChatManager() *grpc.ChatManager {
	return grpc.NewChatManager()
}

// NewServers constructs all protocol-level servers. Grouping them into a single
// bootstrapServers.Servers value avoids the fx same-type ambiguity for multiple
// model.Server providers.
func NewServers(
	cfg *model.Config,
	id AppID,
	cm *grpc.ChatManager,
	cb *callback.Resolver,
	storage domstorage.Client,
	s store.Store,
	eventQueue ports.EventBus,
	log *wlog.Logger,
	tlsCfg *tls.Config,
) bootstrapServers.Servers {
	grpcSrv := grpc.NewServer(&grpc.Config{
		Host:     cfg.Grpc.Host,
		Port:     cfg.Grpc.Port,
		NodeName: string(id),
	}, cm, cb)

	return bootstrapServers.Servers{
		Grpc:    grpcSrv,
		Esl:     fs.NewServer(&fs.Config{Host: cfg.Esl.Host, Port: cfg.Esl.Port, RecordResample: cfg.Record.Sample}),
		Mail:    email.New(storage, s.Email(), cfg.DebugImap),
		Channel: channel.New(eventQueue.ConsumeExec()),
		Im:      im.NewServer(string(id), cfg.DiscoverySettings.Url, eventQueue.ConsumeIM(), log, tlsCfg, s.Session()),
		// Http is created inside NewFlowManager (needs *FlowManager as the App interface)
	}
}
