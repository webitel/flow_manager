package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"net/http"
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
	case "avg", "max", "min":
	default:
		return 0, model.NewAppError("SqlQueueStore.HistoryStatistics", "store.sql_queue.stats.valid", nil,
			"bad metrics", http.StatusBadRequest)
	}

	switch search.Field {
	case "wait_time":
		agg = "extract(epoch from " + search.Metric + "(case when a.bridged_at isnull then (a.leaving_at - a.joined_at) else (a.bridged_at - a.joined_at) end))"
	case "talk_time":
		agg = "extract(epoch from " + search.Metric + "(a.hangup_at - a.bridged_at ))"
	default:
		return 0, model.NewAppError("SqlQueueStore.HistoryStatistics", "store.sql_queue.field.valid", nil,
			"bad field", http.StatusBadRequest)
	}

	res, err := s.GetReplica().SelectFloat(`select `+agg+`
	from cc_member_attempt_history a
	where queue_id = (
		select q.id
		from cc_queue q
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
