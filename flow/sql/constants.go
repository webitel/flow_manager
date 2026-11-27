package sql

const (
	Postgres string = "postgres"
	MySQL    string = "mysql"
	SQLite   string = "sqlite"
	SQLite3  string = "sqlite3"
)

var supportedDrivers = map[string]struct{}{
	Postgres: {},
	MySQL:    {},
	SQLite3:  {},
	SQLite:   {},
}
