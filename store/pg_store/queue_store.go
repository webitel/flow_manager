package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"net/http"
	"strings"
)

type SqlQueueStore struct {
	SqlStore
}

func NewSqlQueueStore(sqlStore SqlStore) store.QueueStore {
	st := &SqlQueueStore{sqlStore}
	return st
}

func (s SqlQueueStore) HistoryCallHoldStatistics(domainId int64, search *model.SearchQueueCompleteStatistics, metric string) (float64, *model.AppError) {
	return 0, nil
}

func (s SqlQueueStore) HistoryStatistics(domainId int64, search *model.SearchQueueCompleteStatistics) (float64, *model.AppError) {
	var agg = ""

	switch search.Metric {
	case "avg", "max", "min", "sl":
	default:
		return 0, model.NewAppError("SqlQueueStore.HistoryStatistics", "store.sql_queue.stats.valid", nil,
			"bad metrics", http.StatusBadRequest)
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
		return 0, model.NewAppError("SqlQueueStore.HistoryStatistics", "store.sql_queue.field.valid", nil,
			"bad field", http.StatusBadRequest)
	}

	fmt.Println(`select ` + agg + `
	from call_center.cc_member_attempt_history a
	where queue_id = (
		select q.id
		from call_center.cc_queue q
		where q.domain_id = :DomainId::int8 and (q.id = :QueueId::int or q.name = :QueueName::varchar)
		limit 1
	) and joined_at between now() - (:Min::int || ' min')::interval and now()
	  and ((:BucketId::int isnull and :BucketName::varchar isnull) or a.bucket_id = (
	      select b.id
	      from call_center.cc_bucket b
	      where b.domain_id = :DomainId::int8 and (b.id = :BucketId::int or b.name = :BucketName::varchar)
        ))`)

	res, err := s.GetReplica().SelectFloat(`select `+agg+`
	from call_center.cc_member_attempt_history a
	where queue_id = (
		select q.id
		from call_center.cc_queue q
		where q.domain_id = :DomainId::int8 and (q.id = :QueueId::int or q.name = :QueueName::varchar)
		limit 1
	) and joined_at between now() - (:Min::int || ' min')::interval and now()
	  and ((:BucketId::int isnull and :BucketName::varchar isnull) or a.bucket_id = (
	      select b.id
	      from call_center.cc_bucket b
	      where b.domain_id = :DomainId::int8 and (b.id = :BucketId::int or b.name = :BucketName::varchar)
        ))`, map[string]interface{}{
		"DomainId":   domainId,
		"QueueId":    search.QueueId,
		"QueueName":  search.QueueName,
		"BucketId":   search.BucketId,
		"BucketName": search.BucketName,
		"Min":        search.LastMinutes,
	})

	if err != nil {
		return 0, model.NewAppError("SqlQueueStore.Statistics", "store.sql_queue.stats.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}

func (s SqlQueueStore) GetQueueData(domainId int64, search *model.SearchEntity) (*model.QueueData, *model.AppError) {
	var res *model.QueueData
	err := s.GetReplica().SelectOne(&res, `select q.type, q.enabled, q.priority
from call_center.cc_queue q
where q.domain_id = :DomainId and (q.id = :Id or q.name = :Name) 
limit 1`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       search.Id,
		"Name":     search.Name,
	})

	if err != nil {
		return nil, model.NewAppError("SqlQueueStore.GetQueueData", "store.sql_queue.data.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}

func (s SqlQueueStore) GetQueueAgents(domainId int64, queueId int, mapRes model.Variables) (model.Variables, *model.AppError) {
	var t *properties
	f := make([]string, 0)

	for k, v := range mapRes {
		switch v {
		case "count", "online", "offline", "pause", "waiting":
			f = append(f, fmt.Sprintf("%s as %s", v, pq.QuoteIdentifier(k)))
		}
	}

	if len(f) == 0 {
		return nil, model.NewAppError("SqlQueueStore.GetQueueAgents", "store.sql_queue.agents.app_error", nil, "bad request", http.StatusBadRequest)
	}

	err := s.GetReplica().SelectOne(&t, `select row_to_json(t.*) as variables
from (
    select
        `+strings.Join(f, ", ")+`
    from (
        SELECT count( distinct a.id)::varchar                                                              as count,
			   (count(distinct a.id) filter ( where a.status = 'offline' ))::varchar                      as offline,
			   (count(distinct a.id) filter ( where a.status = 'online' ))::varchar                       as online,
			   (count(distinct a.id) filter ( where a.status = 'pause' ))::varchar                        as pause,
			   (count(distinct a.id) filter ( where a.status = 'online' and ac.channel isnull ))::varchar as waiting
		from call_center.cc_queue q
				 inner join call_center.cc_queue_skill qs on qs.queue_id = q.id
				 inner join call_center.cc_skill_in_agent sa
							on sa.skill_id = qs.skill_id and sa.capacity between qs.min_capacity and qs.max_capacity
				 inner join call_center.cc_agent a on a.id = sa.agent_id and (q.team_id isnull or q.team_id = a.team_id)
				 left join call_center.cc_agent_channel ac on ac.agent_id = a.id
		where q.domain_id = :DomainId
		  and q.id = :Id
		  and qs.enabled
		  and sa.enabled
             ) t
) t`, map[string]interface{}{
		"Id":       queueId,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlQueueStore.GetQueueAgents", "store.sql_queue.agents.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return t.Variables, nil
}
