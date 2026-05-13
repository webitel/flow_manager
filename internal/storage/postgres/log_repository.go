package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
	"github.com/webitel/flow_manager/store"
)

type LogRepository struct {
	db infraSql.Store
}

func NewLogRepository(db infraSql.Store) store.LogStore {
	return &LogRepository{db: db}
}

const saveLogSQL = `insert into flow.scheme_log (schema_id, log, conn_id) values (@SchemaId, @Log, @ConnId)`

func (r *LogRepository) Save(schemaId int, connId string, log any) error {
	return r.db.Exec(context.Background(), saveLogSQL, pgx.NamedArgs{
		"SchemaId": schemaId,
		"Log":      utils.InterfaceToJson(log),
		"ConnId":   connId,
	})
}
