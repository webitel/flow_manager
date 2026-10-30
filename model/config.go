package model

import (
	"errors"
	"fmt"
	"time"
)

const (
	DATABASE_DRIVER_POSTGRES = "postgres"
)

type Config struct {
	ConfigFile  *string `json:"-" flag:"config_file||JSON file configuration"`
	Id          string  `json:"id" flag:"id|1|Service id" env:"ID"`
	ExternalSql bool    `json:"external_sql" flag:"external_sql|false|Enable external sql query" env:"EXTERNAL_SQL"`
	AllowUseMQ  bool    `json:"allow_use_mq" flag:"allow_use_mq|false|Allow push message to MQ" env:"ALLOW_USE_MQ"`
	Record      struct {
		Sample int `json:"sample" flag:"record_sample|0|Set the sample rate of the recording" env:"RECORD_SAMPLE"`
	} `json:"record"`
	DebugImap                    bool              `json:"debug_imap" flag:"debug_imap|false|Debug IMAP protocol" env:"DEBUG_IMAP"`
	SqlSettings                  SqlSettings       `json:"sql_settings"`
	MQSettings                   MQSettings        `json:"mq_settings"`
	RedisSettings                RedisSettings     `json:"redis_settings"`
	DiscoverySettings            DiscoverySettings `json:"discovery_settings"`
	PreSignedCertificateLocation string            `json:"presigned_cert" flag:"presigned_cert|/opt/storage/key.pem|Location to pre signed certificate" env:"PRESIGNED_CERT"`
	Dev                          bool              `json:"dev" flag:"dev|false|Dev mode" env:"DEV"`
	Esl                          EslSettings       `json:"esl"`
	WebHook                      WebHookSettings   `json:"web_hook"`
	Grpc                         GrpcServeSettings `json:"grpc"`
	// EmailOAuth                   map[string]oauth2.Config `json:"email_oauth2,omitempty"`
	ChatTemplatesSettings ChatTemplatesSettings `json:"chat_templates_settings,omitempty"`
	Log                   LogSettings           `json:"log"`
	Tls                   TLSConfig             `json:"tls"`
	ProxyConfigFile       string                `json:"proxy_config_file" flag:"proxy_config_file||Proxy settings file (http_proxy/https_proxy/no_proxy) watched for changes" env:"PROXY_CONFIG_FILE"`
	EmailEvents           EmailEventsSettings   `json:"email_events"`
}

type LogSettings struct {
	Lvl     string `json:"lvl" flag:"log_lvl|debug|Log level" env:"LOG_LVL"`
	Json    bool   `json:"json" flag:"log_json|false|Log format JSON" env:"LOG_JSON"`
	Otel    bool   `json:"otel" flag:"log_otel|false|Log OTEL" env:"LOG_OTEL"`
	File    string `json:"file" flag:"log_file||Log file directory" env:"LOG_FILE"`
	Console bool   `json:"console" flag:"log_console|false|Log console" env:"LOG_CONSOLE"`
}

type TLSConfig struct {
	CAPath   string `json:"ca" flag:"service.conn.client.ca||Client CA certificate path" env:"SERVICE_CONN_CLIENT_CA"`
	KeyPath  string `json:"key" flag:"service.conn.client.key||Client certificate key path" env:"SERVICE_CONN_CLIENT_KEY"`
	CertPath string `json:"cert" flag:"service.conn.client.cert||Client certificate path" env:"SERVICE_CONN_CLIENT_CERT"`
}

type EslSettings struct {
	Host string `json:"host" flag:"esl_host|localhost|ESL server host" env:"ESL_HOST"`
	Port int    `json:"port" flag:"esl_port|10030|ESL server port" env:"ELSE_PORT"`
}

type ChatTemplatesSettings struct {
	Path string `json:"host" flag:"chat_templates_path|./message_templates|Path to the root folder of  the chat message templates. Templates used in the chat_history application" env:"CHAT_TEMPLATES_PATCH"`
}
type RedisSettings struct {
	Host     string `json:"host,omitempty" flag:"redis_host||Redis server host" env:"REDIS_HOST"`
	Port     int    `json:"port,omitempty" flag:"redis_port||Redis server port" env:"REDIS_PORT"`
	Password string `json:"password,omitempty" flag:"redis_password||Redis password" env:"REDIS_PASSWORD"`
	Database int    `json:"database,omitempty" flag:"redis_database|0|Redis database" env:"REDIS_DATABASE"`
}

func (r *RedisSettings) IsValid() bool {
	if r.Host == "" || r.Port == 0 {
		return false
	}
	return true
}

type GrpcServeSettings struct {
	Host string `json:"host" flag:"grpc_addr|localhost|GRPC server host" env:"GRPC_ADDR"`
	Port int    `json:"port" flag:"grpc_port|0|GRPC server port" env:"GRPC_PORT"`
}

type DiscoverySettings struct {
	Url string `json:"url" flag:"consul|consul:8500|Host to consul" env:"CONSUL"`
}

type SqlSettings struct {
	DriverName                  *string  `json:"driver_name" flag:"sql_driver_name|postgres|" env:"SQL_DRIVER_NAME"`
	DataSource                  *string  `json:"data_source" flag:"data_source|postgres://postgres:postgres@postgres:5432/webitel?fallback_application_name=engine&sslmode=disable&connect_timeout=10&search_path=call_center|Data source" env:"DATA_SOURCE"`
	DataSourceReplicas          []string `json:"data_source_replicas" flag:"sql_data_source_replicas" default:"" env:"SQL_DATA_SOURCE_REPLICAS"`
	MaxIdleConns                *int     `json:"max_idle_conns" flag:"sql_max_idle_conns|5|Maximum idle connections" env:"SQL_MAX_IDLE_CONNS"`
	MaxOpenConns                *int     `json:"max_open_conns" flag:"sql_max_open_conns|5|Maximum open connections" env:"SQL_MAX_OPEN_CONNS"`
	ConnMaxLifetimeMilliseconds *int     `json:"conn_max_lifetime_milliseconds" flag:"sql_conn_max_lifetime_milliseconds|300000|Connection maximum lifetime milliseconds" env:"SQL_CONN_MAX_LIFETIME_MILLISECONDS"`
	Trace                       bool     `json:"trace" flag:"sql_trace|false|Trace SQL" env:"SQL_TRACE"`
}

type WebHookSettings struct {
	Addr string `json:"addr" flag:"web_addr|localhost:5689|Web hook address" env:"WEB_ADDR"`
}

type MQSettings struct {
	Url   string `json:"url" flag:"amqp|amqp://admin:admin@rabbit:5672?heartbeat=10|AMQP connection" env:"AMQP"`
	UseIM bool   `json:"use_im" flag:"use_im|false|Use IM handlers" env:"USE_IM"`
}

// EmailEventsSettings configures consuming email-service events into the Inbox.
type EmailEventsSettings struct {
	Enabled         bool          `json:"enabled" flag:"email_events|false|Consume email-service events and start Flows from them" env:"EMAIL_EVENTS"`
	Prefetch        int           `json:"prefetch" flag:"email_events_prefetch|20|Unacknowledged email events per consumer" env:"EMAIL_EVENTS_PREFETCH"`
	DeliveryLimit   int           `json:"delivery_limit" flag:"email_events_delivery_limit|20|Deliveries of an email event before it goes to the DLQ" env:"EMAIL_EVENTS_DELIVERY_LIMIT"`
	Workers         int           `json:"workers" flag:"email_events_workers|4|Inbox workers starting Flows from email events" env:"EMAIL_EVENTS_WORKERS"`
	Retention       time.Duration `json:"retention" flag:"email_events_retention|720h|How long accepted email events are kept for deduplication" env:"EMAIL_EVENTS_RETENTION"`
	CleanupInterval time.Duration `json:"cleanup_interval" flag:"email_events_cleanup_interval|1h|Interval between Inbox cleanups" env:"EMAIL_EVENTS_CLEANUP_INTERVAL"`
	CleanupBatch    int           `json:"cleanup_batch" flag:"email_events_cleanup_batch|1000|Accepted email events deleted per statement" env:"EMAIL_EVENTS_CLEANUP_BATCH"`

	// RouteTimeout is how long the Flow of an email to a Thread without an attempt may choose a queue.
	RouteTimeout time.Duration `json:"route_timeout" flag:"email_route_timeout|60s|How long the Flow may choose a queue before the fallback queue is used" env:"EMAIL_ROUTE_TIMEOUT"`
	// The route watcher finishes routes on every instance: fallback joins and joins of the Flow left unfinished.
	RouteWatcherInterval  time.Duration `json:"route_watcher_interval" flag:"email_route_watcher_interval|1s|Interval between route watcher cycles" env:"EMAIL_ROUTE_WATCHER_INTERVAL"`
	RouteWatcherBatch     int           `json:"route_watcher_batch" flag:"email_route_watcher_batch|100|Routes claimed by one route watcher cycle" env:"EMAIL_ROUTE_WATCHER_BATCH"`
	RouteLease            time.Duration `json:"route_lease" flag:"email_route_lease|1m|How long the route watcher holds the routes it claimed" env:"EMAIL_ROUTE_LEASE"`
	RouteRetryMaxInterval time.Duration `json:"route_retry_max_interval" flag:"email_route_retry_max_interval|1m|Longest pause between joins of one route" env:"EMAIL_ROUTE_RETRY_MAX_INTERVAL"`
	RouteMaxAttempts      int           `json:"route_max_attempts" flag:"email_route_max_attempts|10|Joins of one route after unexpected errors before it fails" env:"EMAIL_ROUTE_MAX_ATTEMPTS"`
}

const (
	// maxEmailEventsPrefetch is the limit of the AMQP prefetch count, a uint16.
	maxEmailEventsPrefetch     = 65535
	maxEmailEventsCleanupBatch = 10000
	// One join (5s) and its write (5s) must fit in the lease with room to spare.
	minEmailRouteLease = 15 * time.Second
)

// Validate rejects zero values too: an unparsable duration is set to zero.
func (s EmailEventsSettings) Validate() error {
	var errs []error
	positive := func(name string, value int64) {
		if value <= 0 {
			errs = append(errs, fmt.Errorf("%s must be positive", name))
		}
	}

	positive("email_events_prefetch", int64(s.Prefetch))
	positive("email_events_delivery_limit", int64(s.DeliveryLimit))
	positive("email_events_workers", int64(s.Workers))
	positive("email_events_retention", int64(s.Retention))
	positive("email_events_cleanup_interval", int64(s.CleanupInterval))
	positive("email_events_cleanup_batch", int64(s.CleanupBatch))
	positive("email_route_timeout", int64(s.RouteTimeout))
	positive("email_route_watcher_interval", int64(s.RouteWatcherInterval))
	positive("email_route_watcher_batch", int64(s.RouteWatcherBatch))
	positive("email_route_retry_max_interval", int64(s.RouteRetryMaxInterval))
	positive("email_route_max_attempts", int64(s.RouteMaxAttempts))

	if s.Prefetch > maxEmailEventsPrefetch {
		errs = append(errs, fmt.Errorf("email_events_prefetch must not exceed %d", maxEmailEventsPrefetch))
	}
	// One join of the watcher has to fit in its lease.
	if s.RouteLease < minEmailRouteLease {
		errs = append(errs, fmt.Errorf("email_route_lease must be at least %s", minEmailRouteLease))
	}
	if s.CleanupBatch > maxEmailEventsCleanupBatch {
		errs = append(errs, fmt.Errorf("email_events_cleanup_batch must not exceed %d", maxEmailEventsCleanupBatch))
	}

	return errors.Join(errs...)
}
