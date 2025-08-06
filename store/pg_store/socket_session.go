package sqlstore

import (
	"fmt"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SQLSocketSessionStore struct {
	SqlStore
}

func (S SQLSocketSessionStore) Get(userID int64, domainID int64, appName string) (*model.SocketSession, *model.AppError) {
	var session model.SocketSession
	_, err := S.GetReplica().Select(
		&session,
		`
		SELECT id, created_at, updated_at, user_agent, user_id, app_id, application_name
		FROM call_center.socket_session
		WHERE user_id = :UserId
		  AND domain_id = :DomainId
		  AND application_name NOT ILIKE :AppName
		ORDER BY updated_at DESC
		LIMIT 1
		`,
		map[string]interface{}{
			"UserId":   userID,
			"DomainId": domainID,
			"AppName":  appName,
		})
	if err != nil {
		return nil, model.NewAppError(
			"SQLSocketSessionStore.Get",
			"store.sql_socket_session.get.app_error",
			nil,
			fmt.Sprintf("domainID=%v %v", domainID, err.Error()),
			extractCodeFromErr(err),
		)
	}

	return &session, nil
}

func NewSQLSocketSessionStore(sqlStore SqlStore) store.SocketSessionStore {
	s := &SQLSocketSessionStore{sqlStore}
	return s
}
