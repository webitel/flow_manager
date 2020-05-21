package model

const (
	DATABASE_DRIVER_POSTGRES = "postgres"
)

type Config struct {
	Id                string            `json:"id"`
	SqlSettings       SqlSettings       `json:"sql_settings"`
	MQSettings        MQSettings        `json:"mq_settings"`
	DiscoverySettings DiscoverySettings `json:"discovery_settings"`
	Dev               bool              `json:"dev"`
	Esl               ServeSettings     `json:"esl"`
	Grpc              ServeSettings     `json:"grpc"`
	WebChat           WebChatSettings   `json:"web_chat"`
}

type ServeSettings struct {
	Host string
	Port int
}

type DiscoverySettings struct {
	Url string
}

type ServiceSettings struct {
	NodeId                *string
	ListenAddress         *string
	ListenInternalAddress *string
	SessionCacheInMinutes *int
}

type WebChatSettings struct {
	Host string
	Port int
}

type SqlSettings struct {
	DriverName                  *string
	DataSource                  *string
	DataSourceReplicas          []string
	DataSourceSearchReplicas    []string
	MaxIdleConns                *int
	ConnMaxLifetimeMilliseconds *int
	MaxOpenConns                *int
	Trace                       bool
	AtRestEncryptKey            string
	QueryTimeout                *int
}

type MQSettings struct {
	Url string
}
