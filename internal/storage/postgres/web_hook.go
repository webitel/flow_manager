package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type WebHookRepository struct {
	db infraSql.Store
}

func NewWebHookRepository(db infraSql.Store) store.WebHookStore {
	return &WebHookRepository{db: db}
}

type webHookRow struct {
	Key           string   `json:"key" db:"key"`
	Name          string   `json:"name" db:"name"`
	Enabled       bool     `json:"enabled" db:"enabled"`
	SchemaId      int      `json:"schema_id" db:"schema_id"`
	AllowOrigins  []string `json:"origin" db:"origin"`
	DomainId      int64    `json:"domain_id" db:"domain_id"`
	Authorization *string  `json:"authorization" db:"authorization"`
}

const webHookSQL = `select key, name, schema_id, origin, domain_id, "authorization"
from flow.web_hook
where key = @id`

func (r *WebHookRepository) Get(id string) (model.WebHook, error) {
	var row webHookRow
	if err := r.db.Get(context.Background(), &row, webHookSQL, pgx.NamedArgs{
		"id": id,
	}); err != nil {
		return model.WebHook{}, err
	}
	return toWebHook(row), nil
}

func toWebHook(row webHookRow) model.WebHook {
	return model.WebHook{
		Key:           row.Key,
		Name:          row.Name,
		Enabled:       row.Enabled,
		SchemaId:      row.SchemaId,
		AllowOrigins:  row.AllowOrigins,
		DomainId:      row.DomainId,
		Authorization: row.Authorization,
	}
}
