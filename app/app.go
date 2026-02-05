package app

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/webitel/engine/pkg/presign"
	"github.com/webitel/engine/pkg/wbt"
	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/app/bots_client"
	"github.com/webitel/flow_manager/app/cc"
	"github.com/webitel/flow_manager/app/meeting"
	"github.com/webitel/flow_manager/cases"
	"github.com/webitel/flow_manager/gen/contacts"
	"github.com/webitel/flow_manager/gen/engine"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/flow_manager/mq/rabbit"
	"github.com/webitel/flow_manager/providers/channel"
	"github.com/webitel/flow_manager/providers/email"
	"github.com/webitel/flow_manager/providers/fs"
	"github.com/webitel/flow_manager/providers/grpc"
	"github.com/webitel/flow_manager/providers/im"
	"github.com/webitel/flow_manager/providers/web_hook"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/flow_manager/store/cachelayer"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"

	_ "github.com/mbobakov/grpc-consul-resolver"
	// -------------------- plugin(s) -------------------- //
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"
)

type FlowManager struct {
	log           *wlog.Logger
	id            string
	config        *model.Config
	cluster       *cluster
	Store         store.Store
	ExternalStore *cachelayer.ExternalStoreManager

	grpcServer    model.Server
	mailServer    model.Server
	eslServer     model.Server
	channelServer model.Server
	httpServer    model.Server
	imServer      model.Server

	schemaCache model.ObjectCache
	chatManager *grpc.ChatManager
	storage     *storageClient
	cases       *cases.Api
	meeting     *meeting.Client

	timezoneList map[int]*time.Location
	cc           cc.CCManager

	stop    chan struct{}
	stopped chan struct{}

	eventQueue mq.MQ

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

	cacheStore map[CacheType]cachelayer.CacheStore

	//------------- Contacts GRPC Client -------------//
	contacts            *wbt.Client[contacts.ContactsClient]
	contactPhoneNumbers *wbt.Client[contacts.PhonesClient]
	contactVariables    *wbt.Client[contacts.VariablesClient]

	engineCallCli     *wbt.Client[engine.CallServiceClient]
	engineFeedbackCli *wbt.Client[engine.FeedbackServiceClient]
	AiBots            *bots_client.Client

	ctx              context.Context
	otelShutdownFunc otelsdk.ShutdownFunc
	cbr              *CallbackResolver
}

func NewFlowManager() (outApp *FlowManager, outErr error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	fm := &FlowManager{
		config:      config,
		id:          fmt.Sprintf("%s-%s", model.AppServiceName, config.Id),
		schemaCache: model.NewLruWithParams(model.SchemaCacheSize, "schema", model.SchemaCacheExpire, ""),
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		ctx:         context.Background(),
		cbr:         NewCallbackResolver(),
	}

	if config.ExternalSql {
		fm.ExternalStore = cachelayer.NewExternalStoreManager()
	}

	logConfig := &wlog.LoggerConfiguration{
		EnableConsole: config.Log.Console,
		ConsoleJson:   false,
		ConsoleLevel:  config.Log.Lvl,
	}

	if config.Log.File != "" {
		logConfig.FileLocation = config.Log.File
		logConfig.EnableFile = true
		logConfig.FileJson = true
		logConfig.FileLevel = config.Log.Lvl
	}

	if config.Log.Otel {
		// TODO
		logConfig.EnableExport = true
		fm.otelShutdownFunc, err = otelsdk.Configure(
			fm.ctx,
			otelsdk.WithResource(resource.NewSchemaless(
				semconv.ServiceName(model.AppServiceName),
				semconv.ServiceVersion(model.CurrentVersion),
				semconv.ServiceInstanceID(fm.id),
				semconv.ServiceNamespace("webitel"),
			)),
		)
		if err != nil {
			return nil, err
		}
	}

	fm.log = wlog.NewLogger(logConfig)

	fm.callWatcher = NewCallWatcher(fm)
	fm.listWatcher = NewListWatcher(fm)

	wlog.RedirectStdLog(fm.log)
	wlog.InitGlobalLogger(fm.log)

	wlog.Info(fmt.Sprintf("version: %s", Version()))
	wlog.Info("server is initializing...")

	fm.Store = store.NewLayeredStore(sqlstore.NewSqlSupplier(fm.Config().SqlSettings))

	fm.cluster = NewCluster(fm)

	fm.cacheStore = map[CacheType]cachelayer.CacheStore{}
	fm.cacheStore[Memory] = cachelayer.NewMemoryCache(&cachelayer.MemoryCacheConfig{Size: 10000, DefaultExpiry: 10000})
	if config.RedisSettings.IsValid() {
		storage, err := cachelayer.NewRedisCache(config.RedisSettings.Host, config.RedisSettings.Port, config.RedisSettings.Password, config.RedisSettings.Database)
		if err != nil {
			outErr = err
			return outApp, outErr
		}
		fm.cacheStore[Redis] = storage
	}
	fm.chatManager = grpc.NewChatManager()

	grpcSrv := grpc.NewServer(&grpc.Config{
		Host:     fm.Config().Grpc.Host,
		Port:     fm.Config().Grpc.Port,
		NodeName: fm.id,
	}, fm.chatManager, fm.Callback())

	fm.storage, outErr = NewStorageClient(fm.Config().DiscoverySettings.Url)
	if outErr != nil {
		return nil, outErr
	}

	fm.cases, outErr = cases.NewClient(fm.Config().DiscoverySettings.Url)
	if outErr != nil {
		return nil, outErr
	}

	fm.AiBots = bots_client.New(fm.Config().DiscoverySettings.Url)
	if outErr = fm.AiBots.Start(); outErr != nil {
		return nil, outErr
	}

	fm.meeting = meeting.New(fm.Config().DiscoverySettings.Url)
	if outErr = fm.meeting.Start(); outErr != nil {
		return nil, outErr
	}

	fm.grpcServer = grpcSrv
	fm.eslServer = fs.NewServer(&fs.Config{
		Host:           fm.Config().Esl.Host,
		Port:           fm.Config().Esl.Port,
		RecordResample: fm.Config().Record.Sample,
	})
	fm.mailServer = email.New(fm.storage, fm.Store.Email(), fm.Config().DebugImap)
	fm.eventQueue = mq.NewMQ(rabbit.NewRabbitMQ(fm.Config().MQSettings, fm.id))
	fm.channelServer = channel.New(fm.eventQueue.ConsumeExec())

	t, err := LoadTlsCreds(config.Tls)
	if err != nil {
		return nil, err
	}
	fm.imServer = im.NewServer(fm.id, fm.Config().DiscoverySettings.Url, fm.eventQueue.ConsumeIM(),
		fm.log, t, fm.Store.Session())

	if len(fm.Config().WebHook.Addr) > 1 {
		fm.httpServer = web_hook.NewServer(fm, fm.Config().WebHook.Addr)
	}

	if err := fm.RegisterServers(); err != nil {
		outErr = err
		return outApp, outErr
	}

	if err = fm.cluster.Start(); err != nil {
		return nil, err
	}

	if err := fm.chatManager.Start(fm.cluster.discovery); err != nil {
		outErr = err
		return outApp, outErr
	}

	// todo fixme
	if err := grpcSrv.Cluster(fm.cluster.discovery); err != nil {
		outErr = err
		return outApp, outErr
	}

	if config.PreSignedCertificateLocation != "" {
		if fm.cert, err = presign.NewPreSigned(config.PreSignedCertificateLocation); err != nil {
			return nil, err
		}
	}

	fm.cc = cc.NewCCManager(config.DiscoverySettings.Url)

	if err = fm.cc.Start(); err != nil {
		return nil, err
	}
	if err := fm.InitCacheTimezones(); err != nil {
		return nil, err
	}

	if fm.contacts, err = wbt.NewClient(config.DiscoverySettings.Url, wbt.WebitelServiceName, contacts.NewContactsClient); err != nil {
		return nil, err
	}
	if fm.contactPhoneNumbers, err = wbt.NewClient(config.DiscoverySettings.Url, wbt.WebitelServiceName, contacts.NewPhonesClient); err != nil {
		return nil, err
	}
	if fm.contactVariables, err = wbt.NewClient(config.DiscoverySettings.Url, wbt.WebitelServiceName, contacts.NewVariablesClient); err != nil {
		return nil, err
	}

	if fm.engineCallCli, err = wbt.NewClient(config.DiscoverySettings.Url, wbt.EngineServiceName, engine.NewCallServiceClient); err != nil {
		return nil, err
	}

	if fm.engineFeedbackCli, err = wbt.NewClient(config.DiscoverySettings.Url, wbt.EngineServiceName, engine.NewFeedbackServiceClient); err != nil {
		return nil, err
	}

	return fm, outErr
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

	if f.meeting != nil {
		f.meeting.Stop()
	}

	close(f.stop)
	<-f.stopped
	f.StopServers()

	if f.otelShutdownFunc != nil {
		f.otelShutdownFunc(f.ctx)
	}
}

func (f *FlowManager) Log() *wlog.Logger {
	return f.log
}

func (f *FlowManager) Callback() *CallbackResolver {
	return f.cbr
}

func (f *FlowManager) Meeting() *meeting.Client {
	return f.meeting
}
