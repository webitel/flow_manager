package sqlstore

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlUserNotificationStore struct {
	SqlStore
}

func NewSqlUserNotificationStore(sqlStore SqlStore) store.UserNotificationStore {
	st := &SqlUserNotificationStore{sqlStore}

	return st
}

func (s SqlUserNotificationStore) Create(ctx context.Context, n *model.UserNotification) *model.AppError {
	err := s.GetMaster().WithContext(ctx).SelectOne(n, `with n as (
    select nextval('call_center.cc_user_notification_id_seq') id, now() created_at
),
ins as (
    insert into call_center.cc_user_notification (id, domain_id, user_id, created_at, type, message)
    select n.id, :DomainId, u.id, n.created_at, :Type, :Message
    from n, directory.wbt_user u
    where u.dc = :DomainId and u.id = any(:UserIds::int8[])
)
select n.id, call_center.cc_view_timestamp(n.created_at) created_at
from n`, map[string]any{
		"DomainId": n.DomainId,
		"UserIds":  pq.Array(n.ForUsers),
		"Type":     n.Type,
		"Message":  n.Message,
	})
	if err != nil {
		return model.NewAppError("SqlUserNotificationStore.Create", "store.sql_user_notification.create.error", nil,
			fmt.Sprintf("domainId=%v %v", n.DomainId, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlUserNotificationStore) CleanExpired() (int64, *model.AppError) {
	res, err := s.GetMaster().Exec(`delete
from call_center.cc_user_notification t
where t.created_at < now() - make_interval(days => coalesce((select ss.value::int
    from call_center.system_settings ss
    where ss.domain_id = t.domain_id and ss.name = 'message_ttl'), 30))`)
	if err != nil {
		return 0, model.NewAppError("SqlUserNotificationStore.CleanExpired", "store.sql_user_notification.clean_expired.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	count, _ := res.RowsAffected()

	return count, nil
}
