package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"net/http"
)

type SqlMediaStore struct {
	SqlStore
}

func NewSqlMediaStore(sqlStore SqlStore) store.MediaStore {
	st := &SqlMediaStore{sqlStore}
	return st
}

type playbackResponse struct {
	Idx  int `json:"idx"`
	Id   int
	Type string
}

func (s SqlMediaStore) GetFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError) {
	ids := make([]*int, 0)
	names := make([]*string, 0)

	for _, v := range *req {
		if v == nil {
			continue
		}
		if v.Type == nil && (v.Id != nil || v.Name != nil) {
			ids = append(ids, v.Id)
			names = append(names, v.Name)
		}
	}

	var out []*playbackResponse

	_, err := s.GetReplica().Select(&out, `select rec.id - 1 as idx, m.id, m.mime_type as type
from (
    select id, x, (:Names::varchar[])[id] as name
    from unnest(:Ids::int[])  with ordinality ids (x, id)
) rec,
  lateral (
            select m.id, m.mime_type
            from storage.media_files m
            where m.domain_id = :DomainId
                and (rec.x::int8 isnull or m.id = rec.x)
                and (rec.name::varchar isnull or m.name = rec.name)
            limit 1
         ) m`, map[string]interface{}{
		"Ids":      pq.Array(ids),
		"Names":    pq.Array(names),
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlMediaStore.GetFiles", "store.sql_media.get_files.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), http.StatusBadRequest)
	}

	for _, v := range out {
		(*req)[v.Idx].Id = &v.Id
		(*req)[v.Idx].Type = &v.Type
	}

	return *req, nil
}
