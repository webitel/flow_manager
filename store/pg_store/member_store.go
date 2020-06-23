package sqlstore

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlMemberStore struct {
	SqlStore
}

func NewSqlMemberStore(sqlStore SqlStore) store.MemberStore {
	st := &SqlMemberStore{sqlStore}
	return st
}

func (s SqlMemberStore) CallPosition(callId string) (int64, *model.AppError) {
	pos, err := s.GetMaster().SelectInt(`select a.pos
from (
    select row_number()
        over (order by (extract(epoch from now() -  a.joined_at) + a.weight) desc) pos, a.member_call_id
    from cc_member_attempt a
    where a.queue_id = (
        select a2.queue_id
        from cc_member_attempt a2
        where a2.member_call_id = :CallId 
        limit 1
    ) and a.bridged_at isnull and a.leaving_at isnull
    order by (extract(epoch from now() -  a.joined_at) + a.weight) desc
) a
where a.member_call_id = :CallId`, map[string]interface{}{
		"CallId": callId,
	})

	if err != nil {
		return 0, model.NewAppError("SqlMemberStore.CallPosition", "store.sql_member.get_call_position.error", nil,
			fmt.Sprintf("callId=%v %v", callId, err.Error()), extractCodeFromErr(err))
	}

	return pos, nil
}
