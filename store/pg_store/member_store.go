package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"strings"
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

func (s SqlMemberStore) GetProperties(domainId int64, req *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError) {
	f := make([]string, 0)

	for k, v := range mapRes {
		var val = ""
		switch v {
		case "id":
			val = "id::varchar as " + pq.QuoteIdentifier(k)
		case "name":
			val = "name::varchar as " + pq.QuoteIdentifier(k)
		case "priority":
			val = "priority::varchar as " + pq.QuoteIdentifier(k)
		case "attempts":
			val = "attempts::varchar as " + pq.QuoteIdentifier(k)
		case "stop_cause":
			val = "stop_cause::varchar as " + pq.QuoteIdentifier(k)
		default:

			if !strings.HasPrefix(fmt.Sprintf("%s", v), "variables.") {
				continue
			}

			val = fmt.Sprintf("(m.variables->%s) as %s", pq.QuoteLiteral(fmt.Sprintf("%s", v)[10:]), pq.QuoteIdentifier(k))
		}

		f = append(f, val)
	}

	var t *properties

	err := s.GetReplica().SelectOne(&t, `select row_to_json(t) variables
from (
    select
       `+strings.Join(f, ", ")+`
    from cc_member m
    where m.queue_id in (
        select id from cc_queue q where q.domain_id = :DomainId and q.id = any(:QueueIds::int[])
    )
    and (:Name::varchar isnull or m.name ilike :Name)
    and (:Today::bool isnull or (:Today and m.created_at >= ((date_part('epoch'::text, now()::date) * (1000)::double precision))::bigint))
    and (:Completed::bool isnull or ( case when :Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (:BucketId::int isnull or m.bucket_id = :BucketId)
    and (:Destination::varchar isnull or m.communications @>  any (array((select jsonb_build_array(jsonb_build_object('destination', :Destination::varchar))))))
    limit 1
) t`, map[string]interface{}{
		"DomainId":    domainId,
		"QueueIds":    pq.Array(req.QueueIds),
		"Name":        req.Name,
		"Today":       req.Today,
		"Completed":   req.Completed,
		"BucketId":    req.BucketId,
		"Destination": req.Destination,
	})

	if err != nil {
		return nil, model.NewAppError("SqlMemberStore.Get", "store.sql_member.search.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return t.Variables, nil
}

func (s SqlMemberStore) PatchMembers(domainId int64, req *model.SearchMember, patch *model.PatchMember) (int, *model.AppError) {
	i, err := s.GetMaster().SelectNullInt(`with m as (
    update cc_member m
    set name = coalesce(:UName::varchar, name),
        priority = coalesce(:UPriority::int, priority),
        bucket_id = coalesce(:UBucketId::int, bucket_id),
        ready_at = case when :UReadyAt::int8 notnull then to_timestamp(:UReadyAt::int8 /1000) else ready_at end,
        stop_cause = case when :UStopCause::varchar notnull then :UStopCause::varchar else stop_cause end,
        stop_at = case when :UStopCause::varchar notnull then now() else stop_at end,
        variables = case when :UVariables::jsonb notnull then variables || :UVariables::jsonb else variables end
    where m.queue_id in (
        select id from cc_queue q where q.domain_id = :DomainId and q.id = any(:QueueIds::int[])
    )
    and (:Name::varchar isnull or m.name ilike :Name)
    and (:Today::bool isnull or (:Today and m.created_at >= ((date_part('epoch'::text, now()::date) * (1000)::double precision))::bigint))
    and (:Completed::bool isnull or ( case when :Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (:BucketId::int isnull or m.bucket_id = :BucketId)
    and (:Destination::varchar isnull or m.communications @>  any (array((select jsonb_build_array(jsonb_build_object('destination', :Destination::varchar))))))
    returning id
)
select count(*)
from m`, map[string]interface{}{
		"DomainId":    domainId,
		"QueueIds":    pq.Array(req.QueueIds),
		"Name":        req.Name,
		"Today":       req.Today,
		"Completed":   req.Completed,
		"BucketId":    req.BucketId,
		"Destination": req.Destination,

		"UName":      patch.Name,
		"UPriority":  patch.Priority,
		"UBucketId":  patch.BucketId,
		"UReadyAt":   patch.ReadyAt,
		"UStopCause": patch.StopCause,
		"UVariables": patch.Variables.ToJson(),
	})

	if err != nil {
		return 0, model.NewAppError("SqlMemberStore.PatchMembers", "store.sql_member.patch.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if i.Valid {
		return int(i.Int64), nil
	}

	return 0, nil
}
