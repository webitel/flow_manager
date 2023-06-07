package app

import (
	"fmt"
	"time"

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
)

type FlowManager struct {
	Log           *wlog.Logger
	id            string
	config        *model.Config
	cluster       *cluster
	Store         store.Store
	ExternalStore *cachelayer.ExternalStoreManager

	grpcServer model.Server
	mailServer model.Server
	eslServer  model.Server

	schemaCache utils.ObjectCache
	chatManager *grpc.ChatManager
	storage     *storage.Api

	timezoneList map[int]*time.Location
	cc           client.CCManager
	stop         chan struct{}
	stopped      chan struct{}

	eventQueue mq.MQ

	CallRouter  model.Router
	GRPCRouter  model.Router
	EmailRouter model.Router
	ChatRouter  model.Router
	FormRouter  model.Router

	callWatcher *callWatcher
	cert        presign.PreSign
	listWatcher *listWatcher

	cacheStore map[string]cachelayer.CacheStore
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
	}

	if config.ExternalSql {
		fm.ExternalStore = cachelayer.NewExternalStoreManager()
	}

	fm.Log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	fm.callWatcher = NewCallWatcher(fm)
	fm.listWatcher = NewListWatcher(fm)

	wlog.RedirectStdLog(fm.Log)
	wlog.InitGlobalLogger(fm.Log)

	wlog.Info("server is initializing...")

	fm.Store = store.NewLayeredStore(sqlstore.NewSqlSupplier(fm.Config().SqlSettings))

	fm.cluster = NewCluster(fm)

	fm.cacheStore = map[string]cachelayer.CacheStore{}
	fm.cacheStore["memory"] = cachelayer.NewMemoryCache(&cachelayer.MemoryCacheConfig{Size: 1000, DefaultExpiry: 10000})
	if config.RedisSettings.IsValid() {
		fm.cacheStore["redis"] = cachelayer.NewRedisCache(config.RedisSettings.Host, config.RedisSettings.Port, config.RedisSettings.Password, config.RedisSettings.Database)
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
	fm.mailServer = email.New(fm.storage, fm.Store.Email(), fm.Config().EmailOAuth)

	if err := fm.RegisterServers(); err != nil {
		outErr = err
		return
	}

	fm.eventQueue = mq.NewMQ(rabbit.NewRabbitMQ(fm.Config().MQSettings, fm.id))

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
}
