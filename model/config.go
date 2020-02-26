package model

const (
	DATABASE_DRIVER_POSTGRES = "postgres"
)

type Config struct {
	NodeName    string      `json:"node_name"`
	SqlSettings SqlSettings `json:"sql_settings"`
	Dev         bool        `json:"dev"`
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
