package sqlstore

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
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
    from call_center.cc_member_attempt a
    where a.queue_id = (
        select a2.queue_id
        from call_center.cc_member_attempt a2
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

func (s SqlMemberStore) EWTPuzzle(callId string, min int, queueIds []int, bucketIds []int) (float64, *model.AppError) {
	ewt, err := s.GetMaster().SelectFloat(`with att as (
    select a.pos, a.queue_id, a.bucket_id
    from (
             select row_number()
                    over (order by (extract(epoch from now() - a.joined_at) + a.weight) desc) pos,
                    a.member_call_id,
                    a.queue_id,
                    a.bucket_id
             from call_center.cc_member_attempt a
             where a.queue_id = (
                 select a2.queue_id
                 from call_center.cc_member_attempt a2
                 where a2.member_call_id = :CallId
                 limit 1
             )
               and a.bridged_at isnull
               and a.leaving_at isnull
             order by (extract(epoch from now() - a.joined_at) + a.weight) desc
         ) a
    where a.member_call_id = :CallId
    limit 1
)
select (coalesce(extract(epoch from avg(awt)), 0.0) * coalesce(max(att.pos), 0.0))::int8 as awt
from att
left join lateral (
      select bridged_at - joined_at awt
      from call_center.cc_member_attempt_history a
      where a.queue_id = any(:QueueIds::int[])
        and case when :BucketIds::int[] notnull then a.bucket_id = any(:BucketIds::int[]) else a.bucket_id isnull end
        and a.bridged_at notnull
        and a.agent_id notnull
        and a.leaving_at > now() - (:Min || ' min')::interval
      order by leaving_at desc
      limit 2
) s on true;`, map[string]interface{}{
		"CallId":    callId,
		"Min":       min,
		"QueueIds":  pq.Array(queueIds),
		"BucketIds": pq.Array(bucketIds),
	})

	if err != nil {
		return 0,
			model.NewAppError("SqlMemberStore.EWTPuzzle", "store.sql_member.ewt_puzzle.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return ewt, nil
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
		case "bucket_id":
			val = "bucket_id::varchar as " + pq.QuoteIdentifier(k)
		case "count_destinations":
			val = "array_length(m.sys_destinations, 1)::varchar as " + pq.QuoteIdentifier(k)
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
    from call_center.cc_member m
    where m.queue_id in (
        select id from call_center.cc_queue q where q.domain_id = :DomainId and q.id = any(:QueueIds::int[])
    )
    and (:Name::varchar isnull or m.name ilike :Name)
    and (:Today::bool isnull or (:Today and m.created_at >= now()::date))
    and (:Completed::bool isnull or ( case when :Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (:BucketId::int isnull or m.bucket_id = :BucketId)
	and (:Destination::varchar isnull or m.search_destinations && array[:Destination::varchar])
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

// todo variables
func (s SqlMemberStore) PatchMembers(domainId int64, req *model.SearchMember, patch *model.PatchMember) (int, *model.AppError) {
	i, err := s.GetMaster().SelectNullInt(`with m as (
    update call_center.cc_member mu
    set name = coalesce(:UName::varchar, mu.name),
        priority = coalesce(:UPriority::int, mu.priority),
        bucket_id = coalesce(:UBucketId::int, mu.bucket_id),
        ready_at = case when :UReadyAt::int8 notnull then to_timestamp(:UReadyAt::int8 /1000) else mu.ready_at end,
        stop_cause = case when :UStopCause::varchar notnull then :UStopCause::varchar else mu.stop_cause end,
        stop_at = case when :UStopCause::varchar notnull then now() else mu.stop_at end,
        variables = case when (:UVariables::jsonb) notnull then coalesce(mu.variables, '{}'::jsonb) || :UVariables::jsonb else mu.variables end,
        communications = case when :Communications::jsonb notnull and x.v notnull then x.v else mu.communications end
    from call_center.cc_member m
        left join lateral (
            select
                jsonb_agg(case when d notnull then x.comm || (d - '{"id"}') else x.comm end order by idx) v
            from jsonb_array_elements(m.communications) with ordinality x(comm, idx)
                left join jsonb_array_elements(:Communications::jsonb) d on (d->>'id')::int8 = idx - 1
        ) x on :Communications::jsonb notnull
    where m.id = mu.id
    and m.queue_id in (
        select id from call_center.cc_queue q where q.domain_id = :DomainId and q.id = any(:QueueIds::int[])
    )
    and (:Name::varchar isnull or m.name ilike :Name)
    and (:Id::int8 isnull or m.id = :Id::int8)
    and (:Today::bool isnull or (:Today and m.created_at >= now()::date))
    and (:Completed::bool isnull or ( case when :Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (:BucketId::int isnull or m.bucket_id = :BucketId)
    and (:Destination::varchar isnull or m.communications @>  any (array((select jsonb_build_array(jsonb_build_object('destination', :Destination::varchar))))))
    returning mu.id
)
select count(*)
from m`, map[string]interface{}{
		"DomainId":    domainId,
		"QueueIds":    pq.Array(req.GetQueueIds()),
		"Id":          req.Id,
		"Name":        req.Name,
		"Today":       req.Today,
		"Completed":   req.Completed,
		"BucketId":    req.Bucket.GetId(),
		"Destination": req.Destination,

		"UName":          patch.Name,
		"UPriority":      patch.Priority,
		"UBucketId":      patch.Bucket.GetId(),
		"UReadyAt":       patch.ReadyAt,
		"UStopCause":     patch.StopCause,
		"UVariables":     patch.Variables.ToString(),
		"Communications": patch.CommunicationsToJson(),
	})

	if err != nil {
		return 0, model.NewAppError("SqlMemberStore.PatchMembers", "store.sql_member.patch.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if i.Valid {
		return int(i.Int64), nil
	}

	return 0, nil
}

func (s SqlMemberStore) CreateMember(domainId int64, queueId int, holdSec int, member *model.CallbackMember) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into call_center.cc_member(queue_id, communications, name, variables, 
	ready_at, domain_id, timezone_id, priority, bucket_id, expire_at, agent_id)
select q.id queue_id, 
	   json_build_array(
              jsonb_build_object('destination', :Number::varchar)
              || jsonb_build_object('type', jsonb_build_object('id', :TypeId::int))
              || case when :Display::varchar notnull then jsonb_build_object('display', :Display::varchar) else '{}' end
			  || case when :CommunicationPriority::int notnull then jsonb_build_object('priority', :CommunicationPriority::int) else '{}' end
              || case when :ResourceId::int notnull then jsonb_build_object('resource', jsonb_build_object('id', :ResourceId::int)) else '{}'::jsonb end
       ),
       :Name::varchar,
	   case when :Variables::text notnull then :Variables::jsonb else '{}'::jsonb end as vars,
       case when not :HoldSec::int4 isnull then now() + (:HoldSec::int4 || ' sec')::interval else null end lh,
       q.domain_id,
	   :TimezoneId,
	   :Priority,
	   :BucketId,
	   case when :ExpireAt::int8 notnull and :ExpireAt::int8 > 0 then to_timestamp(:ExpireAt::int8/1000::double precision) at time zone tz.sys_name end,
	   :AgentId
from call_center.cc_queue q
	inner join flow.calendar c on c.id = q.calendar_id
    inner join flow.calendar_timezones tz on tz.id = c.timezone_id
where q.id = :QueueId::int4 and q.domain_id = :DomainId::int8`, map[string]interface{}{
		"DomainId":              domainId,
		"QueueId":               queueId,
		"Number":                member.Communication.Destination,
		"TypeId":                member.Communication.Type.GetId(),
		"Name":                  member.Name,
		"HoldSec":               holdSec,
		"Variables":             member.Variables.ToString(),
		"TimezoneId":            member.Timezone.Id,
		"Priority":              member.Priority,
		"BucketId":              member.Bucket.Id,
		"Display":               member.Communication.Display,
		"ResourceId":            member.Communication.ResourceId,
		"ExpireAt":              member.ExpireAt,
		"AgentId":               member.Agent.Id,
		"CommunicationPriority": member.Communication.Priority,
	})

	if err != nil {
		return model.NewAppError("SqlMemberStore.CreateMember", "store.sql_member.create.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return nil
}
