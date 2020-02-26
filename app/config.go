package app

import (
	"flag"
	"github.com/webitel/flow_manager/model"
)

var (
	dataSource = flag.String("data_source", "postgres://opensips:webitel@10.9.8.111:5432/webitel?fallback_application_name=flow_manager&sslmode=disable&connect_timeout=10&search_path=call_center", "Data source")
)

func (f *FlowManager) Config() *model.Config {
	return f.config
}

func loadConfig() (*model.Config, error) {
	flag.Parse()
	config := &model.Config{
		SqlSettings: model.SqlSettings{
			DriverName:                  model.NewString("postgres"),
			DataSource:                  dataSource,
			MaxIdleConns:                model.NewInt(5),
			MaxOpenConns:                model.NewInt(5),
			ConnMaxLifetimeMilliseconds: model.NewInt(300000),
			Trace:                       true,
		},
	}

	return config, nil
}
