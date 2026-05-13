package runtime

import (
	"fmt"

	"github.com/webitel/engine/pkg/presign"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	cacheAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/cache_adapter"
	cases "github.com/webitel/flow_manager/internal/adapters/outbound/cases"
	ccAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/cc"
	chatAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/chat"
	eventAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/event"
	schemaAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/schema"
	fileAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/storage"
	storeAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/store_adapter"
	clusterPkg "github.com/webitel/flow_manager/internal/bootstrap/cluster"
	
	bootstrapServers "github.com/webitel/flow_manager/internal/bootstrap/servers"
	bsversion "github.com/webitel/flow_manager/internal/bootstrap/version"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	_ "github.com/webitel/flow_manager/internal/infrastructure/resolver"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/internal/usecase/callback"
	callWatcherPkg "github.com/webitel/flow_manager/internal/workers/call_watcher"
	listWatcher "github.com/webitel/flow_manager/internal/workers/list_watcher"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/internal/storage"

	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"
)

type FlowManager struct {
	*storeAdapter.Adapter
	*cacheAdapter.CacheAdapter
	*ccAdapter.FMAdapter
	*schemaAdapter.SchemaAdapter
	*fileAdapter.FileAdapter
	*eventAdapter.EventBusAdapter
	*chatAdapter.ChatMgrAdapter
	*Dispatcher

	log    *wlog.Logger
	id     string
	config *bscfg.Config

	cluster *clusterPkg.Cluster
	Store   storage.Store

	srvs       bootstrapServers.Servers
	grpcServer *grpc.Server

	chatManager *grpc.ChatManager
	cases       *cases.Api

	cc domcc.CCManager

	stop    chan struct{}
	stopped chan struct{}

	eventQueue ports.EventBus

	callWatcher *callWatcherPkg.Worker
	listWatcher *listWatcher.Worker

	AiBots  *aibridge.Client
	meeting domainmeeting.Client

	cbr *callback.Resolver
}

func NewFlowManager(
	cfg *bscfg.Config,
	log *wlog.Logger,
	st storage.Store,
	checkpointRepo session.Repository,
	runtimeStateRepo persistence.Repository,
	cacheStores cache.CacheStores,
	storage domstorage.Client,
	casesClient *cases.Api,
	aiBots *aibridge.Client,
	meetingClient domainmeeting.Client,
	srvs bootstrapServers.Servers,
	chatMgr *grpc.ChatManager,
	ccMgr domcc.CCManager,
	eventQueue ports.EventBus,
	cb *callback.Resolver,
) (*FlowManager, error) {
	schemaCache := cache.NewLruWithParams(bscfg.SchemaCacheSize, "schema", bscfg.SchemaCacheExpire, "")

	stop := make(chan struct{})
	stopped := make(chan struct{})

	appID := fmt.Sprintf("%s-%s", bscfg.AppServiceName, cfg.Id)

	fm := &FlowManager{
		Adapter:         storeAdapter.New(st),
		CacheAdapter:    cacheAdapter.New(cacheStores, log),
		FMAdapter:       ccAdapter.NewFMAdapter(ccMgr, st),
		SchemaAdapter:   schemaAdapter.NewSchemaAdapter(st, schemaCache),
		FileAdapter:     fileAdapter.NewFileAdapter(storage),
		EventBusAdapter: eventAdapter.NewEventBusAdapter(eventQueue, st, cfg),
		ChatMgrAdapter:  chatAdapter.NewChatMgrAdapter(chatMgr, cfg.ChatTemplatesSettings.Path),
		Dispatcher: New(DispatcherConfig{
			Log:              log,
			ID:               appID,
			GrpcServer:       srvs.Grpc,
			EslServer:        srvs.Esl,
			MailServer:       srvs.Mail,
			ChannelServer:    srvs.Channel,
			ImServer:         srvs.Im,
			CheckpointRepo:   checkpointRepo,
			RuntimeStateRepo: runtimeStateRepo,
			Stop:             stop,
			Stopped:          stopped,
		}),
		log:         log,
		id:          appID,
		config:      cfg,
		Store:       st,
		cases:       casesClient,
		AiBots:      aiBots,
		meeting:     meetingClient,
		chatManager: chatMgr,
		cc:          ccMgr,
		eventQueue:  eventQueue,
		srvs:        srvs,
		grpcServer:  srvs.Grpc,
		stop:        stop,
		stopped:     stopped,
		cbr:         cb,
	}

	if cfg.ExternalSql {
		fm.Adapter.SetExternalStore(cache.NewExternalStoreManager())
	}

	wlog.Info(fmt.Sprintf("version: %s", bsversion.String()))
	wlog.Info("server is initializing...")

	fm.callWatcher = callWatcherPkg.New(st, fm, log)
	fm.listWatcher = listWatcher.New(st, log)

	return fm, nil
}

// Start runs all I/O-bound startup steps that must happen after the fx graph
// is fully wired. Called from RegisterStartupHooks via fx.Lifecycle.OnStart.
func (fm *FlowManager) Start() error {
	// Start servers first so that grpcServer.Host()/Port() reflect the bound address.
	if err := fm.srvs.Register(); err != nil {
		return err
	}

	// Build cluster after servers are up so we advertise the real gRPC address.
	fm.cluster = clusterPkg.New(
		fm.id,
		fm.config.DiscoverySettings.Url,
		fm.grpcServer.Host(),
		fm.grpcServer.Port(),
	)
	if err := fm.cluster.Start(); err != nil {
		return err
	}
	if err := fm.chatManager.Start(fm.cluster.Discovery); err != nil {
		return err
	}
	if err := fm.grpcServer.Cluster(fm.cluster.Discovery); err != nil {
		return err
	}
	if fm.config.PreSignedCertificateLocation != "" {
		cert, err := presign.NewPreSigned(fm.config.PreSignedCertificateLocation)
		if err != nil {
			return err
		}
		fm.SchemaAdapter.SetCert(cert)
	}
	if err := fm.InitCacheTimezones(); err != nil {
		return err
	}
	return nil
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
	f.srvs.Stop()
}

func (f *FlowManager) Log() *wlog.Logger             { return f.log }
func (f *FlowManager) AppID() string                 { return f.id }
func (f *FlowManager) Callback() *callback.Resolver  { return f.cbr }
func (f *FlowManager) GetStore() storage.Store         { return f.Store }
func (f *FlowManager) GetAiBots() *aibridge.Client   { return f.AiBots }
func (f *FlowManager) Meeting() domainmeeting.Client { return f.meeting }
func (f *FlowManager) Cases() domcases.Client        { return f.cases }

// ConsumeCallEvent satisfies call_watcher.CallEventDeps; delegates to eventQueue.
func (f *FlowManager) ConsumeCallEvent() <-chan call.CallActionData {
	return f.eventQueue.ConsumeCallEvent()
}

// Listen starts background watchers and then blocks in the transport dispatch
// loop until all server goroutines have finished.
//
// The transport-level loop is owned by the embedded Dispatcher; this wrapper
// starts the call and list watchers first so they are alive before any
// connections arrive.
func (f *FlowManager) Listen() {
	f.callWatcher.Start(f.stop)
	f.listWatcher.Start()
	f.Dispatcher.Listen()
}

func (f *FlowManager) Config() *bscfg.Config {
	return f.config
}
