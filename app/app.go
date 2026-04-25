package app

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/engine/pkg/presign"
	"github.com/webitel/wlog"

	_ "github.com/webitel/flow_manager/infra/resolver"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	cases "github.com/webitel/flow_manager/internal/adapters/outbound/cases"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
	fmgrpc "github.com/webitel/flow_manager/providers/grpc"
	fmhttp "github.com/webitel/flow_manager/providers/http"
	"github.com/webitel/flow_manager/store"

	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"
)

var _ ports.RouterDeps = (*FlowManager)(nil)

type FlowManager struct {
	log            *wlog.Logger
	id             string
	config         *model.Config
	cluster        *cluster
	Store          store.Store
	ExternalStore  *cache.ExternalStoreManager
	checkpointRepo session.Repository

	grpcServer    *fmgrpc.Server
	mailServer    model.Server
	eslServer     model.Server
	channelServer model.Server
	httpServer    model.Server
	imServer      model.Server

	schemaCache model.ObjectCache
	chatManager *fmgrpc.ChatManager
	storage     domstorage.Client
	cases       *cases.Api

	timezoneList map[int]*time.Location
	cc           domcc.CCManager

	stop    chan struct{}
	stopped chan struct{}

	eventQueue ports.EventBus

	CallRouter    model.Router
	GRPCRouter    model.Router
	EmailRouter   model.Router
	ChatRouter    model.Router
	FormRouter    model.Router
	ChannelRouter model.Router
	WebHookRouter model.Router
	IMRouter      model.Router

	callWatcher *callWatcher
	cert        presign.PreSign
	listWatcher *listWatcher

	cacheStore cache.CacheStores

	AiBots *aibridge.Client

	ctx context.Context
	cbr *CallbackResolver
}

func NewFlowManager(
	cfg *model.Config,
	log *wlog.Logger,
	st store.Store,
	checkpointRepo session.Repository,
	cacheStores cache.CacheStores,
	storage domstorage.Client,
	casesClient *cases.Api,
	aiBots *aibridge.Client,
	srvs Servers,
	chatMgr *fmgrpc.ChatManager,
	ccMgr domcc.CCManager,
	eventQueue ports.EventBus,
	cb *CallbackResolver,
) (*FlowManager, error) {
	fm := &FlowManager{
		log:            log,
		id:             fmt.Sprintf("%s-%s", model.AppServiceName, cfg.Id),
		config:         cfg,
		Store:          st,
		checkpointRepo: checkpointRepo,
		cacheStore:     cacheStores,
		storage:        storage,
		cases:          casesClient,
		AiBots:         aiBots,
		chatManager:    chatMgr,
		cc:             ccMgr,
		eventQueue:     eventQueue,
		grpcServer:     srvs.Grpc,
		eslServer:      srvs.Esl,
		mailServer:     srvs.Mail,
		channelServer:  srvs.Channel,
		imServer:       srvs.Im,
		httpServer:     srvs.Http,
		schemaCache:    model.NewLruWithParams(model.SchemaCacheSize, "schema", model.SchemaCacheExpire, ""),
		stop:           make(chan struct{}),
		stopped:        make(chan struct{}),
		ctx:            context.Background(),
		cbr:            cb,
	}

	if cfg.ExternalSql {
		fm.ExternalStore = cache.NewExternalStoreManager()
	}

	wlog.Info(fmt.Sprintf("version: %s", Version()))
	wlog.Info("server is initializing...")

	fm.callWatcher = NewCallWatcher(fm)
	fm.listWatcher = NewListWatcher(fm)
	fm.cluster = NewCluster(fm)

	if err := fm.cluster.Start(); err != nil {
		return nil, err
	}
	if err := fm.chatManager.Start(fm.cluster.discovery); err != nil {
		return nil, err
	}
	if err := srvs.Grpc.Cluster(fm.cluster.discovery); err != nil {
		return nil, err
	}
	if len(cfg.WebHook.Addr) > 1 {
		fm.httpServer = fmhttp.NewServer(fm, cfg.WebHook.Addr)
	}
	if err := fm.RegisterServers(); err != nil {
		return nil, err
	}

	if cfg.PreSignedCertificateLocation != "" {
		cert, err := presign.NewPreSigned(cfg.PreSignedCertificateLocation)
		if err != nil {
			return nil, err
		}
		fm.cert = cert
	}

	if err := fm.InitCacheTimezones(); err != nil {
		return nil, err
	}

	return fm, nil
}

func (f *FlowManager) Shutdown() {
	wlog.Info("stopping Server...")
	if f.cluster != nil {
		f.cluster.Stop()
	}

	if f.callWatcher != nil {
		f.callWatcher.Stop()
	}

	if f.listWatcher != nil {
		f.listWatcher.Stop()
	}

	if f.cc != nil {
		f.cc.Stop()
	}

	if f.chatManager != nil {
		f.chatManager.Stop()
	}

	if f.AiBots != nil {
		f.AiBots.Stop()
	}

	close(f.stop)
	<-f.stopped
	f.StopServers()
}

func (f *FlowManager) Log() *wlog.Logger {
	return f.log
}

func (f *FlowManager) AppID() string {
	return f.id
}

func (f *FlowManager) CheckpointRepo() session.Repository {
	return f.checkpointRepo
}

func (f *FlowManager) Callback() *CallbackResolver {
	return f.cbr
}

func (f *FlowManager) GetStore() store.Store {
	return f.Store
}

func (f *FlowManager) GetAiBots() *aibridge.Client {
	return f.AiBots
}
