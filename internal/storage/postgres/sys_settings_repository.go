package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/storage"
)

type SysSettingsRepository struct {
	db infraSql.Store
}

func NewSysSettingsRepository(db infraSql.Store) storage.SystemcSettings {
	return &SysSettingsRepository{db: db}
}

const getSysSettingSQL = `
select value
from call_center.system_settings
where domain_id = @DomainId and name = @Name
`

func (r *SysSettingsRepository) Get(ctx context.Context, domainId int64, name string) (json.RawMessage, error) {
	type row struct {
		Value json.RawMessage `db:"value"`
	}
	var res row
	if err := r.db.Get(ctx, &res, getSysSettingSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Name":     name,
	}); err != nil {
		return nil, fmt.Errorf("domainId=%v name=%q: %w", domainId, name, err)
	}
	return res.Value, nil
}
