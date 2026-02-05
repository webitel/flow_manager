package model

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
	//EmailOAuth                   map[string]oauth2.Config `json:"email_oauth2,omitempty"`
	ChatTemplatesSettings ChatTemplatesSettings `json:"chat_templates_settings,omitempty"`
	Log                   LogSettings           `json:"log"`
	Tls                   TLSConfig             `json:"tls"`
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
	Url string `json:"url" flag:"amqp|amqp://admin:admin@rabbit:5672?heartbeat=10|AMQP connection" env:"AMQP"`
}
