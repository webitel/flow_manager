package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SocketSessionRepository struct {
	db infraSql.Store
}

func NewSocketSessionRepository(db infraSql.Store) store.SocketSessionStore {
	return &SocketSessionRepository{db: db}
}

const getSocketSessionSQL = `
SELECT id, created_at, updated_at, user_agent, user_id, app_id, application_name
FROM call_center.socket_session
WHERE user_id = @UserId
  AND domain_id = @DomainId
  AND application_name NOT ILIKE @AppName
ORDER BY updated_at DESC
LIMIT 1
`

func (r *SocketSessionRepository) Get(userID, domainID int64, appName string) (*model.SocketSession, error) {
	var s model.SocketSession
	if err := r.db.Get(context.Background(), &s, getSocketSessionSQL, pgx.NamedArgs{
		"UserId":   userID,
		"DomainId": domainID,
		"AppName":  appName,
	}); err != nil {
		return nil, fmt.Errorf("domainID=%v userID=%v: %w", domainID, userID, err)
	}
	return &s, nil
}
