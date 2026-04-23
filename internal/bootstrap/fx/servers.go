package bsfx

import (
	"crypto/tls"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/app"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/flow_manager/providers/channel"
	"github.com/webitel/flow_manager/providers/email"
	"github.com/webitel/flow_manager/providers/fs"
	fmgrpc "github.com/webitel/flow_manager/providers/grpc"
	"github.com/webitel/flow_manager/providers/im"
	"github.com/webitel/flow_manager/store"
)

func NewCallbackResolver() *app.CallbackResolver {
	return app.NewCallbackResolver()
}

func NewTLSConfig(cfg *model.Config) (*tls.Config, error) {
	return bscfg.LoadTLSCreds(cfg.Tls)
}

func NewGrpcServer(cfg *model.Config, id AppID, cm *fmgrpc.ChatManager, cb *app.CallbackResolver) *fmgrpc.Server {
	return fmgrpc.NewServer(&fmgrpc.Config{
		Host:     cfg.Grpc.Host,
		Port:     cfg.Grpc.Port,
		NodeName: string(id),
	}, cm, cb)
}

func NewEslServer(cfg *model.Config) model.Server {
	return fs.NewServer(&fs.Config{
		Host:           cfg.Esl.Host,
		Port:           cfg.Esl.Port,
		RecordResample: cfg.Record.Sample,
	})
}

func NewMailServer(storage *app.StorageClient, s store.Store, cfg *model.Config) model.Server {
	return email.New(storage, s.Email(), cfg.DebugImap)
}

func NewChannelServer(eventQueue mq.MQ) model.Server {
	return channel.New(eventQueue.ConsumeExec())
}

func NewImServer(id AppID, cfg *model.Config, eventQueue mq.MQ, log *wlog.Logger, tlsCfg *tls.Config, s store.Store) model.Server {
	return im.NewServer(
		string(id),
		cfg.DiscoverySettings.Url,
		eventQueue.ConsumeIM(),
		log,
		tlsCfg,
		s.Session(),
	)
}
