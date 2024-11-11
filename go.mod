module github.com/webitel/flow_manager

go 1.22.5

toolchain go1.22.6

replace github.com/emersion/go-imap v1.2.1 => github.com/navrotskyj/go-imap v1.2.2-0.20240927130548-8f6fa2edadb3

require github.com/emersion/go-imap v1.2.1

require (
	buf.build/gen/go/webitel/cc/grpc/go v1.4.0-20240527133229-8d9250d78122.2
	buf.build/gen/go/webitel/cc/protocolbuffers/go v1.34.2-20240527133229-8d9250d78122.2
	buf.build/gen/go/webitel/chat/grpc/go v1.4.0-20240703121916-db3266bbfcd3.2
	buf.build/gen/go/webitel/chat/protocolbuffers/go v1.34.2-20240703121916-db3266bbfcd3.2
	buf.build/gen/go/webitel/engine/protocolbuffers/go v1.35.1-20240402125447-cb375844242f.1
	buf.build/gen/go/webitel/storage/grpc/go v1.3.0-20240402131908-59ee244702ce.2
	buf.build/gen/go/webitel/storage/protocolbuffers/go v1.35.1-20241001075334-99db0a72402e.1
	buf.build/gen/go/webitel/workflow/grpc/go v1.3.0-20240411120545-24ef43af6db3.2
	buf.build/gen/go/webitel/workflow/protocolbuffers/go v1.33.0-20240411132047-cd3c8f61d791.1
	github.com/BoRuDar/configuration/v4 v4.5.0
	github.com/emersion/go-message v0.18.1
	github.com/emersion/go-sasl v0.0.0-20231106173351-e73c9f7bad43
	github.com/euskadi31/go-tokenizer v1.0.0
	github.com/go-gorp/gorp v2.2.0+incompatible
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/h2non/filetype v1.1.3
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/k3a/html2text v1.2.1
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/mbobakov/grpc-consul-resolver v1.5.3
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pborman/uuid v1.2.1
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/robertkrimen/otto v0.3.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/tidwall/gjson v1.17.1
	github.com/webitel/call_center v0.0.0-20240724125933-0749f15dcada
	github.com/webitel/engine v0.0.0-20240725113537-577386ebe3e6
	github.com/webitel/webitel-go-kit v0.0.13-0.20240908192731-3abe573c0e41
	github.com/webitel/wlog v0.0.0-20240815102009-b8f74bd674e2
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/sdk v1.29.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sync v0.8.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.35.1
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/xmlpath.v2 v2.0.0-20150820204837-860cbeca3ebc
)

require (
	buf.build/gen/go/grpc-ecosystem/grpc-gateway/protocolbuffers/go v1.35.1-20231027202514-3f42134f4c56.1 // indirect
	buf.build/gen/go/webitel/webitel-go/grpc/go v1.4.0-20240725111140-8ade8003e454.2 // indirect
	buf.build/gen/go/webitel/webitel-go/protocolbuffers/go v1.34.2-20240725111140-8ade8003e454.2 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dchest/cssmin v0.0.0-20151210170030-fb8d9b44afdc // indirect
	github.com/dchest/htmlmin v1.2.0 // indirect
	github.com/dchest/jsmin v0.0.0-20220218165748-59f39799265f // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form v3.1.4+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.21.0 // indirect
	github.com/hashicorp/consul/api v1.28.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.0.0-20240812153829-bb9ac54eca05 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.0.0-20240805233418-127d068751eb // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.4.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.28.0 // indirect
	go.opentelemetry.io/otel/log v0.5.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.5.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20240404231335-c0f41cb1a7a0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
)
