package app

import (
	"flag"
	"github.com/webitel/flow_manager/model"
)

var (
	nodeName      = flag.String("id", "1", "Node ID")
	dataSource    = flag.String("data_source", "postgres://opensips:webitel@postgres:5432/webitel?fallback_application_name=engine&sslmode=disable&connect_timeout=10&search_path=call_center", "Data source")
	consulHost    = flag.String("consul", "consul:8500", "Host to consul")
	amqp          = flag.String("amqp", "amqp://webitel:webitel@rabbit:5672?heartbeat=10", "AMQP connection")
	eslServerHost = flag.String("esl_host", "", "ESL server host")
	eslServerPort = flag.Int("esl_port", 10030, "ESL server port")

	grpcServerHost = flag.String("grpc_addr", "", "GRPC server host")
	grpcServerPort = flag.Int("grpc_port", 0, "GRPC server port")

	webChatServerHost = flag.String("web_addr", "", "WebChat server host")
	webChatServerPort = flag.Int("web_port", 7777, "WebChat server port")
)

func (f *FlowManager) Config() *model.Config {
	return f.config
}

func loadConfig() (*model.Config, error) {
	flag.Parse()
	config := &model.Config{
		Id: *nodeName,
		SqlSettings: model.SqlSettings{
			DriverName:                  model.NewString("postgres"),
			DataSource:                  dataSource,
			MaxIdleConns:                model.NewInt(5),
			MaxOpenConns:                model.NewInt(5),
			ConnMaxLifetimeMilliseconds: model.NewInt(30000),
			Trace:                       false,
		},
		DiscoverySettings: model.DiscoverySettings{
			Url: *consulHost,
		},
		MQSettings: model.MQSettings{
			Url: *amqp,
		},
		Esl: model.ServeSettings{
			Host: *eslServerHost,
			Port: *eslServerPort,
		},
		Grpc: model.ServeSettings{
			Host: *grpcServerHost,
			Port: *grpcServerPort,
		},
		WebChat: model.WebChatSettings{
			Host: *webChatServerHost,
			Port: *webChatServerPort,
		},
	}

	return config, nil
}
