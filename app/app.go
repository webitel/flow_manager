package app

import (
	"context"
	"fmt"
	otelsdk "github.com/webitel/webitel-go-kit/otel/sdk"
	"go.opentelemetry.io/otel/sdk/resource"
	"time"

	"github.com/webitel/flow_manager/providers/web_hook"

	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/webitel/flow_manager/providers/channel"

	"github.com/webitel/engine/pkg/webitel_client"

	"github.com/webitel/flow_manager/storage"

	"github.com/webitel/flow_manager/providers/email"

	"github.com/webitel/call_center/grpc_api/client"
	presign "github.com/webitel/engine/presign"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/flow_manager/mq/rabbit"
	"github.com/webitel/flow_manager/providers/fs"
	"github.com/webitel/flow_manager/providers/grpc"
	"github.com/webitel/flow_manager/store"
	"github.com/webitel/flow_manager/store/cachelayer"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"
	"github.com/webitel/wlog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

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

	schemaCache utils.ObjectCache
	chatManager *grpc.ChatManager
	storage     *storage.Api

	timezoneList map[int]*time.Location
	cc           client.CCManager
	stop         chan struct{}
	stopped      chan struct{}

	eventQueue mq.MQ

	CallRouter    model.Router
	GRPCRouter    model.Router
	EmailRouter   model.Router
	ChatRouter    model.Router
	FormRouter    model.Router
	ChannelRouter model.Router
	WebHookRouter model.Router

	callWatcher *callWatcher
	cert        presign.PreSign
	listWatcher *listWatcher

	cacheStore       map[CacheType]cachelayer.CacheStore
	wbtCli           *webitel_client.Client
	ctx              context.Context
	otelShutdownFunc otelsdk.ShutdownFunc
}

func NewFlowManager() (outApp *FlowManager, outErr error) {

	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	fm := &FlowManager{
		config:      config,
		id:          fmt.Sprintf("%s-%s", model.AppServiceName, config.Id),
		schemaCache: utils.NewLruWithParams(model.SchemaCacheSize, "schema", model.SchemaCacheExpire, ""),
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		ctx:         context.Background(),
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

	wlog.Info("server is initializing...")

	fm.Store = store.NewLayeredStore(sqlstore.NewSqlSupplier(fm.Config().SqlSettings))

	fm.cluster = NewCluster(fm)

	fm.cacheStore = map[CacheType]cachelayer.CacheStore{}
	fm.cacheStore[Memory] = cachelayer.NewMemoryCache(&cachelayer.MemoryCacheConfig{Size: 10000, DefaultExpiry: 10000})
	if config.RedisSettings.IsValid() {
		storage, err := cachelayer.NewRedisCache(config.RedisSettings.Host, config.RedisSettings.Port, config.RedisSettings.Password, config.RedisSettings.Database)
		if err != nil {
			outErr = err
			return
		}
		fm.cacheStore[Redis] = storage
	}
	fm.chatManager = grpc.NewChatManager()

	grpcSrv := grpc.NewServer(&grpc.Config{
		Host:     fm.Config().Grpc.Host,
		Port:     fm.Config().Grpc.Port,
		NodeName: fm.id,
	}, fm.chatManager)

	fm.storage, outErr = storage.NewClient(fm.Config().DiscoverySettings.Url)
	if outErr != nil {
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

	if len(fm.Config().WebHook.Addr) > 1 {
		fm.httpServer = web_hook.NewServer(fm, fm.Config().WebHook.Addr)
	}

	if err := fm.RegisterServers(); err != nil {
		outErr = err
		return
	}

	if err = fm.cluster.Start(); err != nil {
		return nil, err
	}

	if err := fm.chatManager.Start(fm.cluster.discovery); err != nil {
		outErr = err
		return
	}

	//todo fixme
	if err := grpcSrv.Cluster(fm.cluster.discovery); err != nil {
		outErr = err
		return
	}

	if config.PreSignedCertificateLocation != "" {
		if fm.cert, err = presign.NewPreSigned(config.PreSignedCertificateLocation); err != nil {
			return nil, err
		}
	}

	fm.cc = client.NewCCManager(fm.cluster.discovery)
	if err = fm.cc.Start(); err != nil {
		return nil, err
	}
	if err := fm.InitCacheTimezones(); err != nil {
		return nil, err
	}

	if fm.wbtCli, err = webitel_client.New(0, 0, config.DiscoverySettings.Url); err != nil {
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
