package app

import (
	"fmt"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/fs"
	"github.com/webitel/flow_manager/providers/grpc"
	"github.com/webitel/flow_manager/store"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"
	"github.com/webitel/wlog"
)

type FlowManager struct {
	Log         *wlog.Logger
	id          string
	config      *model.Config
	cluster     *cluster
	Store       store.Store
	servers     []model.Server
	schemaCache utils.ObjectCache
	stop        chan struct{}
	stopped     chan struct{}

	FlowRouter model.Router
	CallRouter model.CallRouter
	GRPCRouter model.GRPCRouter
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

	wlog.RedirectStdLog(fm.Log)
	wlog.InitGlobalLogger(fm.Log)

	wlog.Info("server is initializing...")

	servers := []model.Server{
		grpc.NewServer(&grpc.Config{
			Host: "",
			Port: 8043,
		}),
		fs.NewServer(&fs.Config{
			Host: "",
			Port: 10030,
		}),
	}

	fm.Store = store.NewLayeredStore(sqlstore.NewSqlSupplier(fm.Config().SqlSettings))

	if err := fm.RegisterServers(servers...); err != nil {
		outErr = err
		return
	}

	fm.cluster = NewCluster(fm)
	if err = fm.cluster.Start(); err != nil {
		return nil, err
	}

	return fm, outErr
}

func (f *FlowManager) Shutdown() {
	wlog.Info("stopping Server...")
	if f.cluster != nil {
		f.cluster.Stop()
	}
	close(f.stop)
	<-f.stopped
	f.StopServers()
}
