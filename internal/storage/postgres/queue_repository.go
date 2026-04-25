package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	pgsql "github.com/webitel/flow_manager/infra/sql/pgsql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type QueueRepository struct {
	db infraSql.Store
}

func NewQueueRepository(db infraSql.Store) store.QueueStore {
	return &QueueRepository{db: db}
}

type historyStatRow struct {
	Value *float64 `db:"value"`
}

type findQueueRow struct {
	Id int32 `db:"id"`
}

func (r *QueueRepository) HistoryStatistics(domainId int64, search *model.SearchQueueCompleteStatistics) (float64, error) {
	var agg string

	switch search.Metric {
	case "avg", "max", "min", "sl":
	default:
		return 0, errors.New("bad metrics")
	}

	switch search.Field {
	case "sl":
		agg = fmt.Sprintf(`((count(*) filter ( where bridged_at notnull and bridged_at - joined_at < interval '%d sec'))::decimal / count(*)::decimal) * 100`,
			search.SlSec)
	case "wait_time":
		agg = "extract(epoch from " + search.Metric + "(case when a.bridged_at isnull then (a.leaving_at - a.joined_at) else (a.bridged_at - a.joined_at) end))"
	case "talk_time":
		agg = "extract(epoch from " + search.Metric + "(a.hangup_at - a.bridged_at ))"
	default:
		return 0, errors.New("bad field")
	}

	q := `select ` + agg + ` as value
	from call_center.cc_member_attempt_history a
	where queue_id = (
		select q.id
		from call_center.cc_queue q
		where q.domain_id = @DomainId::int8 and (q.id = @QueueId::int or q.name = @QueueName::varchar)
		limit 1
	) and joined_at between now() - (@Min::int || ' min')::interval and now()
	  and ((@BucketId::int isnull and @BucketName::varchar isnull) or a.bucket_id = (
	      select b.id
	      from call_center.cc_bucket b
	      where b.domain_id = @DomainId::int8 and (b.id = @BucketId::int or b.name = @BucketName::varchar)
	    ))`

	var row historyStatRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId":   domainId,
		"QueueId":    search.QueueId,
		"QueueName":  search.QueueName,
		"BucketId":   search.BucketId,
		"BucketName": search.BucketName,
		"Min":        search.LastMinutes,
	}); err != nil {
		return 0, err
	}

	if row.Value == nil {
		return 0, nil
	}
	return *row.Value, nil
}

func (r *QueueRepository) GetQueueData(domainId int64, search *model.SearchEntity, mapRes model.Variables) (model.Variables, error) {
	f := make([]string, 0, len(mapRes))
	for k, v := range mapRes {
		var val string
		switch v {
		case "type":
			val = "type::varchar as " + pgsql.QuoteIdentifier(k)
		case "name":
			val = "name::varchar as " + pgsql.QuoteIdentifier(k)
		case "enabled":
			val = "enabled::varchar as " + pgsql.QuoteIdentifier(k)
		case "priority":
			val = "priority::varchar as " + pgsql.QuoteIdentifier(k)
		case "waiting":
			val = "waiting::varchar as " + pgsql.QuoteIdentifier(k)
		case "size":
			val = "size::varchar as " + pgsql.QuoteIdentifier(k)
		default:
			continue
		}
		f = append(f, val)
	}

	q := `select row_to_json(t) as variables
from (
    select ` + strings.Join(f, ", ") + `
    from (
        select q.type, q.enabled, q.priority, q.name,
               coalesce(case when q.type = any(array[1,6]) then (select count(*) from call_center.cc_member_attempt a1 where a1.queue_id = q.id and a1.bridged_at isnull)
                       else (select sum(s.member_waiting) from call_center.cc_queue_statistics s where s.queue_id = q.id) end, 0) waiting,
               coalesce(case when q.type = any(array[1,6]) then (select count(*) from call_center.cc_member_attempt a1 where a1.queue_id = q.id and a1.state != 'leaving')
                       else (select sum(s.member_waiting) from call_center.cc_queue_statistics s where s.queue_id = q.id) end, 0) size
        from call_center.cc_queue q
        where q.domain_id = @DomainId and (q.id = @Id or q.name = @Name)
        limit 1
    ) t
) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId": domainId,
		"Id":       search.Id,
		"Name":     search.Name,
	}); err != nil {
		return nil, err
	}

	var vars model.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

func (r *QueueRepository) GetQueueAgents(domainId int64, queueId int, channel string, mapRes model.Variables) (model.Variables, error) {
	f := make([]string, 0, len(mapRes))
	for k, v := range mapRes {
		switch v {
		case "count", "online", "offline", "pause", "waiting":
			f = append(f, fmt.Sprintf("%s as %s", v, pgsql.QuoteIdentifier(k)))
		}
	}

	if len(f) == 0 {
		return nil, errors.New("bad request")
	}

	q := `select row_to_json(t.*) as variables
from (
    select ` + strings.Join(f, ", ") + `
    from (
        SELECT count(distinct a.id)::varchar as count,
               (count(distinct a.id) filter ( where a.status = 'offline' ))::varchar as offline,
               (count(distinct a.id) filter ( where a.status = 'online' ))::varchar as online,
               (count(distinct a.id) filter ( where a.status = 'pause' ))::varchar as pause,
               (count(distinct a.id) filter ( where a.status = 'online' and ac.state = 'waiting' ))::varchar as waiting
        from call_center.cc_queue q
                 inner join call_center.cc_queue_skill qs on qs.queue_id = q.id
                 inner join call_center.cc_skill_in_agent sa
                            on sa.skill_id = qs.skill_id and sa.capacity between qs.min_capacity and qs.max_capacity
                 inner join call_center.cc_agent a on a.id = sa.agent_id and (q.team_id isnull or q.team_id = a.team_id)
                 left join call_center.cc_agent_channel ac on ac.agent_id = a.id and ac.channel = @Channel::text
        where q.domain_id = @DomainId
          and q.id = @Id
          and qs.enabled
          and sa.enabled
    ) t
) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"Id":       queueId,
		"DomainId": domainId,
		"Channel":  channel,
	}); err != nil {
		return nil, err
	}

	var vars model.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

func (r *QueueRepository) FindQueueByName(domainId int64, name string) (int32, error) {
	var row findQueueRow
	if err := r.db.Get(context.Background(), &row, `select q.id
from call_center.cc_queue q
where q.domain_id = @DomainId::int8 and q.name = @Name::text
limit 1`, pgx.NamedArgs{
		"DomainId": domainId,
		"Name":     name,
	}); err != nil {
		return 0, err
	}
	return row.Id, nil
}
