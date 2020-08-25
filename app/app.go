package app

import (
	"fmt"
	"github.com/webitel/call_center/grpc_api/client"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/flow_manager/mq/rabbit"
	"github.com/webitel/flow_manager/providers/fs"
	"github.com/webitel/flow_manager/providers/grpc"
	"github.com/webitel/flow_manager/store"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"
	"github.com/webitel/wlog"
	"time"
)

type FlowManager struct {
	Log         *wlog.Logger
	id          string
	config      *model.Config
	cluster     *cluster
	Store       store.Store
	servers     []model.Server
	schemaCache utils.ObjectCache

	timezoneList map[int]*time.Location
	cc           client.CCManager
	stop         chan struct{}
	stopped      chan struct{}

	eventQueue mq.MQ

	CallRouter  model.Router
	GRPCRouter  model.Router
	EmailRouter model.Router
	ChatRouter  model.Router

	callWatcher *callWatcher
}

func NewFlowManager() (outApp *FlowManager, outErr error) {

	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	fm := &FlowManager{
		config:      config,
		id:          fmt.Sprintf("%s-%s", model.AppServiceName, config.Id),
		servers:     make([]model.Server, 0, 1),
		schemaCache: utils.NewLruWithParams(model.SchemaCacheSize, "schema", model.SchemaCacheExpire, ""),
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
	}

	fm.Log = wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
	})

	fm.callWatcher = NewCallWatcher(fm)

	wlog.RedirectStdLog(fm.Log)
	wlog.InitGlobalLogger(fm.Log)

	wlog.Info("server is initializing...")

	fm.Store = store.NewLayeredStore(sqlstore.NewSqlSupplier(fm.Config().SqlSettings))

	servers := []model.Server{
		grpc.NewServer(&grpc.Config{
			Host: fm.Config().Grpc.Host,
			Port: fm.Config().Grpc.Port,
		}),
		fs.NewServer(&fs.Config{
			Host: fm.Config().Esl.Host,
			Port: fm.Config().Esl.Port,
		}),
	}

	if err := fm.RegisterServers(servers...); err != nil {
		outErr = err
		return
	}

	fm.eventQueue = mq.NewMQ(rabbit.NewRabbitMQ(fm.Config().MQSettings, fm.id))

	fm.cluster = NewCluster(fm)
	if err = fm.cluster.Start(); err != nil {
		return nil, err
	}

	//todo fixme
	if err := servers[0].Cluster(fm.cluster.discovery); err != nil {
		outErr = err
		return
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

	if f.cc != nil {
		f.cc.Stop()
	}

	close(f.stop)
	<-f.stopped
	f.StopServers()
}
