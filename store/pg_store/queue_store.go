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

func (s SqlQueueStore) Statistics(domainId int64, search *model.SearchQueue, metric string) (float64, *model.AppError) {
	var f = ""

	switch metric {
	case "avg":
		f = "extract( epoch  from avg(bridged_at - joined_at))"
	case "max":
		f = "extract( epoch  from max(bridged_at - joined_at))"
	case "min":
		f = "extract( epoch  from min(bridged_at - joined_at))"
	case "count":
		f = "count(*)"
	default:
		return 0, model.NewAppError("SqlQueueStore.Statistics", "store.sql_queue.stats.valid", nil,
			"bad metrics", http.StatusBadRequest)
	}

	res, err := s.GetReplica().SelectFloat(`select `+f+`
	from cc_member_attempt_history h
	where queue_id = (
		select q.id
		from cc_queue q
		where q.domain_id = :DomainId::int8 and (q.id = :QueueId::int or q.name = :QueueName::varchar)
		limit 1
	) and leaving_at between now() - (:Min::int || ' min')::interval and now()
	  and (:BucketId::int isnull or h.bucket_id = :BucketId)`, map[string]interface{}{
		"DomainId":  domainId,
		"QueueId":   search.Id,
		"QueueName": search.Name,
		"BucketId":  search.BucketId,
		"Min":       search.LastMinutes,
	})

	if err != nil {
		return 0, model.NewAppError("SqlQueueStore.Statistics", "store.sql_queue.stats.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}
