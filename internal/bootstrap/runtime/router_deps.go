package runtime

import (
	"fmt"

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
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	bsversion "github.com/webitel/flow_manager/internal/bootstrap/version"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	_ "github.com/webitel/flow_manager/internal/infrastructure/resolver"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/internal/usecase/callback"
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

// RouterDeps satisfies all channel router Deps interfaces (call, chat, im, email,
// processing, channel, grpc) through adapter embedding + forwarding methods.
// It has no lifecycle concerns — Start/Shutdown belong to RegisterInfraHooks and
// Dispatcher respectively.
type RouterDeps struct {
	*storeAdapter.Adapter
	*cacheAdapter.CacheAdapter
	*ccAdapter.FMAdapter
	*schemaAdapter.SchemaAdapter
	*fileAdapter.FileAdapter
	*eventAdapter.EventBusAdapter
	*chatAdapter.ChatMgrAdapter

	log    *wlog.Logger
	appID  string
	config *bscfg.Config

	Store storage.Store

	cases   *cases.Api
	AiBots  *aibridge.Client
	meeting domainmeeting.Client
	cbr     *callback.Resolver

	checkpointRepo   session.Repository
	runtimeStateRepo persistence.Repository
}

func NewRouterDeps(
	cfg *bscfg.Config,
	log *wlog.Logger,
	st storage.Store,
	checkpointRepo session.Repository,
	runtimeStateRepo persistence.Repository,
	cacheStores cache.CacheStores,
	storageClient domstorage.Client,
	casesClient *cases.Api,
	aiBots *aibridge.Client,
	meetingClient domainmeeting.Client,
	chatMgr *grpc.ChatManager,
	ccMgr domcc.CCManager,
	eventBus *eventAdapter.EventBusAdapter,
	cb *callback.Resolver,
) *RouterDeps {
	schemaCache := cache.NewLruWithParams(bscfg.SchemaCacheSize, "schema", bscfg.SchemaCacheExpire, "")

	appID := fmt.Sprintf("%s-%s", bscfg.AppServiceName, cfg.Id)

	d := &RouterDeps{
		Adapter:         storeAdapter.New(st),
		CacheAdapter:    cacheAdapter.New(cacheStores, log),
		FMAdapter:       ccAdapter.NewFMAdapter(ccMgr, st),
		SchemaAdapter:   schemaAdapter.NewSchemaAdapter(st, schemaCache),
		FileAdapter:     fileAdapter.NewFileAdapter(storageClient),
		EventBusAdapter: eventBus,
		ChatMgrAdapter:  chatAdapter.NewChatMgrAdapter(chatMgr, cfg.ChatTemplatesSettings.Path),
		log:             log,
		appID:           appID,
		config:          cfg,
		Store:           st,
		cases:           casesClient,
		AiBots:          aiBots,
		meeting:         meetingClient,
		cbr:             cb,
		checkpointRepo:  checkpointRepo,
		runtimeStateRepo: runtimeStateRepo,
	}

	if cfg.ExternalSql {
		d.Adapter.SetExternalStore(cache.NewExternalStoreManager())
	}

	wlog.Info(fmt.Sprintf("version: %s", bsversion.String()))
	wlog.Info("server is initializing...")

	return d
}

func (d *RouterDeps) Log() *wlog.Logger             { return d.log }
func (d *RouterDeps) AppID() string                 { return d.appID }
func (d *RouterDeps) Config() *bscfg.Config         { return d.config }
func (d *RouterDeps) Callback() *callback.Resolver  { return d.cbr }
func (d *RouterDeps) GetStore() storage.Store       { return d.Store }
func (d *RouterDeps) GetAiBots() *aibridge.Client   { return d.AiBots }
func (d *RouterDeps) Meeting() domainmeeting.Client { return d.meeting }
func (d *RouterDeps) Cases() domcases.Client        { return d.cases }

func (d *RouterDeps) CheckpointRepo() session.Repository       { return d.checkpointRepo }
func (d *RouterDeps) RuntimeStateRepo() persistence.Repository { return d.runtimeStateRepo }
