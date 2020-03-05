package model

const (
	DATABASE_DRIVER_POSTGRES = "postgres"
)

type Config struct {
	Id                string            `json:"id"`
	SqlSettings       SqlSettings       `json:"sql_settings"`
	DiscoverySettings DiscoverySettings `json:"discovery_settings"`
	Dev               bool              `json:"dev"`
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
