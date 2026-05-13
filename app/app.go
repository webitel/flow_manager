package app

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/engine/pkg/presign"
	"github.com/webitel/wlog"

	_ "github.com/webitel/flow_manager/infra/resolver"
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
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/internal/infrastructure/cache"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"

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

	log              *wlog.Logger
	id               string
	config           *model.Config
	cluster          *cluster
	Store            store.Store
	ExternalStore    *cache.ExternalStoreManager
	checkpointRepo   session.Repository
	runtimeStateRepo persistence.Repository

	grpcServer    *grpc.Server
	mailServer    model.Server
	eslServer     model.Server
	channelServer model.Server
	imServer      model.Server

	chatManager *grpc.ChatManager
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
	IMRouter      model.Router

	callWatcher *callWatcher
	listWatcher *listWatcher

	cacheStore cache.CacheStores

	AiBots  *aibridge.Client
	meeting domainmeeting.Client

	ctx context.Context
	cbr *CallbackResolver
}

func NewFlowManager(
	cfg *model.Config,
	log *wlog.Logger,
	st store.Store,
	checkpointRepo session.Repository,
	runtimeStateRepo persistence.Repository,
	cacheStores cache.CacheStores,
	storage domstorage.Client,
	casesClient *cases.Api,
	aiBots *aibridge.Client,
	meetingClient domainmeeting.Client,
	srvs Servers,
	chatMgr *grpc.ChatManager,
	ccMgr domcc.CCManager,
	eventQueue ports.EventBus,
	cb *CallbackResolver,
) (*FlowManager, error) {
	schemaCache := model.NewLruWithParams(model.SchemaCacheSize, "schema", model.SchemaCacheExpire, "")
	fm := &FlowManager{
		Adapter:         storeAdapter.New(st),
		CacheAdapter:    cacheAdapter.New(cacheStores, log),
		FMAdapter:       ccAdapter.NewFMAdapter(ccMgr, st),
		SchemaAdapter:   schemaAdapter.NewSchemaAdapter(st, schemaCache),
		FileAdapter:     fileAdapter.NewFileAdapter(storage),
		EventBusAdapter: eventAdapter.NewEventBusAdapter(eventQueue, st, cfg),
		ChatMgrAdapter:  chatAdapter.NewChatMgrAdapter(chatMgr, cfg.ChatTemplatesSettings.Path),
		log:             log,
		id:              fmt.Sprintf("%s-%s", model.AppServiceName, cfg.Id),
		config:          cfg,
		Store:           st,
		checkpointRepo:  checkpointRepo,
		runtimeStateRepo: runtimeStateRepo,
		cacheStore:      cacheStores,
		storage:         storage,
		cases:           casesClient,
		AiBots:          aiBots,
		meeting:         meetingClient,
		chatManager:     chatMgr,
		cc:              ccMgr,
		eventQueue:      eventQueue,
		grpcServer:      srvs.Grpc,
		eslServer:       srvs.Esl,
		mailServer:      srvs.Mail,
		channelServer:   srvs.Channel,
		imServer:        srvs.Im,
		stop:            make(chan struct{}),
		stopped:         make(chan struct{}),
		ctx:             context.Background(),
		cbr:             cb,
	}

	if cfg.ExternalSql {
		fm.ExternalStore = cache.NewExternalStoreManager()
	}

	wlog.Info(fmt.Sprintf("version: %s", Version()))
	wlog.Info("server is initializing...")

	fm.callWatcher = NewCallWatcher(fm)
	fm.listWatcher = NewListWatcher(fm)
	fm.cluster = NewCluster(fm)

	return fm, nil
}

// Start runs all I/O-bound startup steps that must happen after the fx graph
// is fully wired. Called from RegisterStartupHooks via fx.Lifecycle.OnStart.
func (fm *FlowManager) Start() error {
	if err := fm.cluster.Start(); err != nil {
		return err
	}
	if err := fm.chatManager.Start(fm.cluster.discovery); err != nil {
		return err
	}
	if err := fm.grpcServer.Cluster(fm.cluster.discovery); err != nil {
		return err
	}
	if err := fm.RegisterServers(); err != nil {
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
	f.StopServers()
}

func (f *FlowManager) Log() *wlog.Logger         { return f.log }
func (f *FlowManager) AppID() string              { return f.id }
func (f *FlowManager) CheckpointRepo() session.Repository   { return f.checkpointRepo }
func (f *FlowManager) RuntimeStateRepo() persistence.Repository { return f.runtimeStateRepo }
func (f *FlowManager) Callback() *CallbackResolver { return f.cbr }
func (f *FlowManager) GetStore() store.Store       { return f.Store }
func (f *FlowManager) GetAiBots() *aibridge.Client { return f.AiBots }
func (f *FlowManager) Meeting() domainmeeting.Client { return f.meeting }
func (f *FlowManager) Cases() domcases.Client      { return f.cases }
