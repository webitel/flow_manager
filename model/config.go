package model

const (
	DATABASE_DRIVER_POSTGRES = "postgres"
)

type Config struct {
	ConfigFile  *string `json:"-" flag:"config_file||JSON file configuration"`
	Id          string  `json:"id" flag:"id|1|Service id"`
	ExternalSql bool    `json:"external_sql" flag:"external_sql|false|Enable external sql query"`
	AllowUseMQ  bool    `json:"allow_use_mq" flag:"allow_use_mq|false|Allow push message to MQ"`
	Record      struct {
		Sample int `json:"sample" flag:"record_sample|0|Set the sample rate of the recording"`
	} `json:"record"`
	DebugImap                    bool              `json:"debug_imap" flag:"debug_imap|false|Debug IMAP protocol"`
	SqlSettings                  SqlSettings       `json:"sql_settings"`
	MQSettings                   MQSettings        `json:"mq_settings"`
	RedisSettings                RedisSettings     `json:"redis_settings"`
	DiscoverySettings            DiscoverySettings `json:"discovery_settings"`
	PreSignedCertificateLocation string            `json:"presigned_cert" flag:"presigned_cert|/opt/storage/key.pem|Location to pre signed certificate"`
	Dev                          bool              `json:"dev" flag:"dev|false|Dev mode"`
	Esl                          EslSettings       `json:"esl"`
	WebHook                      WebHookSettings   `json:"web_hook"`
	Grpc                         GrpcServeSettings `json:"grpc"`
	//EmailOAuth                   map[string]oauth2.Config `json:"email_oauth2,omitempty"`
	ChatTemplatesSettings ChatTemplatesSettings `json:"chat_templates_settings,omitempty"`
	Log                   LogSettings           `json:"log"`
}

type LogSettings struct {
	Lvl     string `json:"lvl" flag:"log_lvl|debug|Log level"`
	Json    bool   `json:"json" flag:"log_json|false|Log format JSON"`
	Otel    bool   `json:"otel" flag:"log_otel|false|Log OTEL"`
	File    string `json:"file" flag:"log_file||Log file directory"`
	Console bool   `json:"console" flag:"log_console|true|Log console" env:"LOG_CONSOLE"`
}

type EslSettings struct {
	Host string `json:"host" flag:"esl_host|localhost|ESL server host"`
	Port int    `json:"port" flag:"esl_port|10030|ESL server port"`
}

type ChatTemplatesSettings struct {
	Path string `json:"host" flag:"chat_templates_path|./message_templates|Path to the root folder of  the chat message templates. Templates used in the chat_history application"`
}
type RedisSettings struct {
	Host     string `json:"host,omitempty" flag:"redis_host||Redis server host"`
	Port     int    `json:"port,omitempty" flag:"redis_port||Redis server port"`
	Password string `json:"password,omitempty" flag:"redis_password||Redis password"`
	Database int    `json:"database,omitempty" flag:"redis_database|0|Redis database"`
}

func (r *RedisSettings) IsValid() bool {
	if r.Host == "" || r.Port == 0 {
		return false
	}
	return true
}

type GrpcServeSettings struct {
	Host string `json:"host" flag:"grpc_addr|localhost|GRPC server host"`
	Port int    `json:"port" flag:"grpc_port|0|GRPC server port"`
}

type DiscoverySettings struct {
	Url string `json:"url" flag:"consul|consul:8500|Host to consul"`
}

type SqlSettings struct {
	DriverName                  *string  `json:"driver_name" flag:"sql_driver_name|postgres|"`
	DataSource                  *string  `json:"data_source" flag:"data_source|postgres://opensips:webitel@postgres:5432/webitel?fallback_application_name=engine&sslmode=disable&connect_timeout=10&search_path=call_center|Data source"`
	DataSourceReplicas          []string `json:"data_source_replicas" flag:"sql_data_source_replicas" default:""`
	MaxIdleConns                *int     `json:"max_idle_conns" flag:"sql_max_idle_conns|5|Maximum idle connections"`
	MaxOpenConns                *int     `json:"max_open_conns" flag:"sql_max_open_conns|5|Maximum open connections"`
	ConnMaxLifetimeMilliseconds *int     `json:"conn_max_lifetime_milliseconds" flag:"sql_conn_max_lifetime_milliseconds|300000|Connection maximum lifetime milliseconds"`
	Trace                       bool     `json:"trace" flag:"sql_trace|false|Trace SQL"`
}

type WebHookSettings struct {
	Addr string `json:"addr" flag:"web_addr|localhost:5689|Web hook address"`
}

type MQSettings struct {
	Url string `json:"url" flag:"amqp|amqp://webitel:webitel@rabbit:5672?heartbeat=10|AMQP connection"`
}
