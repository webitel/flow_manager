package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	pgsql "github.com/webitel/flow_manager/internal/infrastructure/sql/pgsql"
	"github.com/webitel/flow_manager/internal/storage"
)

type MemberRepository struct {
	db infraSql.Store
}

func NewMemberRepository(db infraSql.Store) storage.MemberStore {
	return &MemberRepository{db: db}
}

type callPositionRow struct {
	Pos int64 `db:"pos"`
}

const callPositionSQL = `select a.pos
from (
    select row_number()
        over (order by (extract(epoch from now() -  a.joined_at) + a.weight) desc) pos, a.member_call_id
    from call_center.cc_member_attempt a
    where a.queue_id = (
        select a2.queue_id
        from call_center.cc_member_attempt a2
        where a2.member_call_id = @CallId
        limit 1
    ) and a.bridged_at isnull and a.leaving_at isnull
    order by (extract(epoch from now() -  a.joined_at) + a.weight) desc
) a
where a.member_call_id = @CallId`

func (r *MemberRepository) CallPosition(callId string) (int64, error) {
	var row callPositionRow
	if err := r.db.Get(context.Background(), &row, callPositionSQL, pgx.NamedArgs{
		"CallId": callId,
	}); err != nil {
		return 0, err
	}
	return row.Pos, nil
}

type ewtRow struct {
	Awt float64 `db:"awt"`
}

const ewtPuzzleSQL = `with q as materialized (
     select a2.queue_id
     from call_center.cc_member_attempt a2
     where a2.member_call_id = @CallId
        and a2.leaving_at notnull
     limit 1
), att as (
    select case when exists(select * from q) then
        (select a.pos
    from (
             select row_number()
                    over (order by (extract(epoch from now() - a.joined_at) + a.weight) desc) pos,
                    a.member_call_id,
                    a.queue_id,
                    a.bucket_id
             from call_center.cc_member_attempt a
             where a.queue_id = (
                 select q.queue_id
                 from q
                 limit 1
             )
               and a.domain_id = @DomainId::int8
               and a.bridged_at isnull
               and a.leaving_at isnull
             order by (extract(epoch from now() - a.joined_at) + a.weight) desc
         ) a
    where a.member_call_id = @CallId
    limit 1) else (
             select count(*) + 1
             from call_center.cc_member_attempt a
             where a.queue_id = any(@QueueIds::int[])
                and case when @BucketIds::int[] notnull then a.bucket_id = any(@BucketIds::int[]) else a.bucket_id isnull end
               and a.domain_id = @DomainId::int8
               and a.bridged_at isnull
               and a.leaving_at isnull
        ) end as pos

)
select (coalesce(extract(epoch from avg(awt)), 0.0) * coalesce(max(att.pos), 0.0))::int8 as awt
from att
left join lateral (
      select bridged_at - joined_at awt
      from call_center.cc_member_attempt_history a
      where a.queue_id = any(@QueueIds::int[])
		and a.domain_id = @DomainId::int8
        and case when @BucketIds::int[] notnull then a.bucket_id = any(@BucketIds::int[]) else a.bucket_id isnull end
        and a.bridged_at notnull
        and a.agent_id notnull
        and a.leaving_at > now() - (@Min || ' min')::interval
      order by leaving_at desc
      limit 2
) s on true`

func (r *MemberRepository) EWTPuzzle(domainId int64, callId string, min int, queueIds, bucketIds []int) (float64, error) {
	var row ewtRow
	if err := r.db.Get(context.Background(), &row, ewtPuzzleSQL, pgx.NamedArgs{
		"CallId":    callId,
		"Min":       min,
		"DomainId":  domainId,
		"QueueIds":  queueIds,
		"BucketIds": bucketIds,
	}); err != nil {
		return 0, err
	}
	return row.Awt, nil
}

func (r *MemberRepository) GetProperties(domainId int64, req *queue.SearchMember, mapRes flow.Variables) (flow.Variables, error) {
	fields := make([]string, 0, len(mapRes))
	for k, v := range mapRes {
		var col string
		switch v {
		case "id":
			col = "id::varchar as " + pgsql.QuoteIdentifier(k)
		case "name":
			col = "name::varchar as " + pgsql.QuoteIdentifier(k)
		case "priority":
			col = "priority::varchar as " + pgsql.QuoteIdentifier(k)
		case "attempts":
			col = "attempts::varchar as " + pgsql.QuoteIdentifier(k)
		case "stop_cause":
			col = "stop_cause::varchar as " + pgsql.QuoteIdentifier(k)
		case "bucket_id":
			col = "bucket_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "count_destinations":
			col = "array_length(m.sys_destinations, 1)::varchar as " + pgsql.QuoteIdentifier(k)
		case "all_destinations":
			col = "array_length(m.search_destinations, 1)::varchar as " + pgsql.QuoteIdentifier(k)
		default:
			sv := fmt.Sprintf("%s", v)
			if !strings.HasPrefix(sv, "variables.") {
				continue
			}
			col = fmt.Sprintf("(m.variables->%s) as %s", pgsql.QuoteLiteral(sv[10:]), pgsql.QuoteIdentifier(k))
		}
		fields = append(fields, col)
	}

	q := `select row_to_json(t) variables
from (
    select
       ` + strings.Join(fields, ", ") + `
    from call_center.cc_member m
    where m.queue_id in (
        select id from call_center.cc_queue q where q.domain_id = @DomainId and q.id = any(@QueueIds::int[])
    )
    and (@Name::varchar isnull or m.name ilike @Name)
	and (@Id::int8 isnull or m.id = @Id::int8)
    and (@Today::bool isnull or (@Today and m.created_at >= now()::date))
    and (@Completed::bool isnull or ( case when @Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (@BucketId::int isnull or m.bucket_id = @BucketId)
	and (@Destination::varchar isnull or m.search_destinations && array[@Destination::varchar])
    limit 1
) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId":    domainId,
		"Id":          req.Id,
		"QueueIds":    req.GetQueueIds(),
		"Name":        req.Name,
		"Today":       req.Today,
		"Completed":   req.Completed,
		"BucketId":    req.BucketId,
		"Destination": req.Destination,
	}); err != nil {
		return nil, err
	}

	var vars flow.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

type patchMembersRow struct {
	Count int64 `db:"count"`
}

const patchMembersSQL = `with m as (
    update call_center.cc_member mu
    set name = coalesce(@UName::varchar, mu.name),
        priority = coalesce(@UPriority::int, mu.priority),
        bucket_id = coalesce(@UBucketId::int, mu.bucket_id),
        queue_id = coalesce(@UQueueId::int, mu.queue_id),
        ready_at = case when @UReadyAt::int8 notnull then to_timestamp(@UReadyAt::int8 /1000) else mu.ready_at end,
        stop_cause = case when @UStopCause::varchar notnull then @UStopCause::varchar else mu.stop_cause end,
        stop_at = case when @UStopCause::varchar notnull then now() else mu.stop_at end,
        variables = case when (@UVariables::jsonb) notnull then coalesce(mu.variables, '{}'::jsonb) || @UVariables::jsonb else mu.variables end,
        communications = case when @Communications::jsonb notnull and x.v notnull then x.v else mu.communications end
    from call_center.cc_member m
        left join lateral (
            select
                jsonb_agg(case when d notnull then x.comm || (d - '{"id"}') else x.comm end order by idx) v
            from jsonb_array_elements(m.communications) with ordinality x(comm, idx)
                left join jsonb_array_elements(@Communications::jsonb) d on (d->>'id')::int8 = idx - 1
        ) x on @Communications::jsonb notnull
    where m.id = mu.id
    and m.queue_id in (
        select id from call_center.cc_queue q where q.domain_id = @DomainId and q.id = any(@QueueIds::int[])
    )
    and (@Name::varchar isnull or m.name ilike @Name)
    and (@Id::int8 isnull or m.id = @Id::int8)
    and (@Today::bool isnull or not @Today::bool or (@Today and m.created_at >= now()::date))
    and (@Completed::bool isnull or ( case when @Completed then not m.stop_at isnull else m.stop_at isnull end ))
    and (@BucketId::int isnull or m.bucket_id = @BucketId)
    and (@Destination::varchar isnull or m.communications @>  any (array((select jsonb_build_array(jsonb_build_object('destination', @Destination::varchar))))))
    returning mu.id
)
select count(*)
from m`

func (r *MemberRepository) PatchMembers(domainId int64, req *queue.SearchMember, patch *queue.PatchMember) (int, error) {
	var row patchMembersRow
	if err := r.db.Get(context.Background(), &row, patchMembersSQL, pgx.NamedArgs{
		"DomainId":    domainId,
		"QueueIds":    req.GetQueueIds(),
		"Id":          req.Id,
		"Name":        req.GetName(),
		"Today":       req.Today,
		"Completed":   req.Completed,
		"BucketId":    req.Bucket.GetId(),
		"Destination": req.Destination,

		"UName":          patch.Name,
		"UPriority":      patch.Priority,
		"UBucketId":      patch.Bucket.GetId(),
		"UReadyAt":       patch.ReadyAt,
		"UStopCause":     patch.StopCause,
		"UVariables":     flow.VariablesToString(patch.Variables),
		"UQueueId":       patch.QueueId,
		"Communications": patch.CommunicationsToJson(),
	}); err != nil {
		return 0, err
	}
	return int(row.Count), nil
}

const createMemberSQL = `insert into call_center.cc_member(queue_id, communications, name, variables,
	ready_at, domain_id, timezone_id, priority, bucket_id, expire_at, agent_id, stop_cause, stop_at)
select q.id queue_id,
	   json_build_array(
              jsonb_build_object('destination', @Number::varchar)
              || jsonb_build_object('type', jsonb_build_object('id', @TypeId::int))
              || case when @Display::varchar notnull then jsonb_build_object('display', @Display::varchar) else '{}' end
			  || case when @CommunicationPriority::int notnull then jsonb_build_object('priority', @CommunicationPriority::int) else '{}' end
              || case when @ResourceId::int notnull then jsonb_build_object('resource', jsonb_build_object('id', @ResourceId::int)) else '{}'::jsonb end
       ),
       @Name::varchar,
	   case when @Variables::text notnull then @Variables::jsonb else '{}'::jsonb end as vars,
       case when not @HoldSec::int4 isnull then now() + (@HoldSec::int4 || ' sec')::interval else null end lh,
       q.domain_id,
	   @TimezoneId,
	   @Priority,
	   @BucketId,
	   case when @ExpireAt::int8 notnull and @ExpireAt::int8 > 0 then to_timestamp(@ExpireAt::int8/1000::double precision) at time zone tz.sys_name end,
	   @AgentId,
	   @StopCause,
	   case when @StopCause::varchar notnull then now() end
from call_center.cc_queue q
	inner join flow.calendar c on c.id = q.calendar_id
    inner join flow.calendar_timezones tz on tz.id = c.timezone_id
where q.id = @QueueId::int4 and q.domain_id = @DomainId::int8`

func (r *MemberRepository) CreateMember(domainId int64, queueId, holdSec int, member *queue.CallbackMember) error {
	return r.db.Exec(context.Background(), createMemberSQL, pgx.NamedArgs{
		"DomainId":              domainId,
		"QueueId":               queueId,
		"Number":                member.Communication.Destination,
		"TypeId":                member.Communication.Type.GetId(),
		"Name":                  member.Name,
		"HoldSec":               holdSec,
		"Variables":             flow.VariablesToString(member.Variables),
		"TimezoneId":            member.Timezone.Id,
		"Priority":              member.Priority,
		"BucketId":              member.Bucket.Id,
		"Display":               member.Communication.Display,
		"ResourceId":            member.Communication.ResourceId,
		"ExpireAt":              member.ExpireAt,
		"AgentId":               member.Agent.Id,
		"CommunicationPriority": member.Communication.Priority,
		"StopCause":             member.StopCause,
	})
}
