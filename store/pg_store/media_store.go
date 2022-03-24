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
	Idx  *int `json:"idx"`
	Id   *int
	Type *string
}

func (s SqlMediaStore) Get(domainId int64, id int) (*model.File, *model.AppError) {
	var file *model.File
	err := s.GetReplica().SelectOne(&file, `select f.id, f.name, f.size, f.mime_type
from storage.media_files f
where f.domain_id = :DomainId and f.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	})

	if err != nil {
		return nil, model.NewAppError("SqlMediaStore.Get", "store.sql_media.get.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), http.StatusBadRequest)
	}

	return file, nil
}

func (s SqlMediaStore) SearchOne(domainId int64, search *model.SearchFile) (*model.File, *model.AppError) {
	var file *model.File
	err := s.GetReplica().SelectOne(&file, `select f.id, f.name, f.size, f.mime_type
from storage.media_files f
where f.domain_id = :DomainId and (f.id = :Id or f.name = :Name) limit 1`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       search.Id,
		"Name":     search.Name,
	})

	if err != nil {
		return nil, model.NewAppError("SqlMediaStore.SearchOne", "store.sql_media.search.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), http.StatusBadRequest)
	}

	return file, nil
}

func (s SqlMediaStore) GetFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError) {
	ids := make([]*int, 0)
	names := make([]*string, 0)

	for _, v := range *req {
		if v != nil && v.Type == nil && (v.Id != nil || v.Name != nil) {
			ids = append(ids, v.Id)
			names = append(names, v.Name)
		} else {
			ids = append(ids, nil)
			names = append(names, nil)
		}
	}

	var out []*playbackResponse

	_, err := s.GetReplica().Select(&out, `select rec.id - 1 as idx, m.id, m.mime_type as type
from (
    select id, x, (:Names::varchar[])[id] as name
    from unnest(:Ids::int[])  with ordinality ids (x, id)
) rec
  left join lateral (
            select m.id, m.mime_type
            from storage.media_files m
            where m.domain_id = :DomainId and not (rec.x::int8 isnull and rec.name::varchar isnull )
                and ( (rec.x::int8 notnull and m.id = rec.x) or
                      (rec.name::varchar notnull and m.name = rec.name) )
            limit 1
         ) m on true`, map[string]interface{}{
		"Ids":      pq.Array(ids),
		"Names":    pq.Array(names),
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlMediaStore.GetFiles", "store.sql_media.get_files.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), http.StatusBadRequest)
	}

	for _, v := range out {
		if v.Type == nil || v.Idx == nil || (*req)[*v.Idx] == nil {
			continue
		}
		(*req)[*v.Idx].Id = v.Id
		(*req)[*v.Idx].Type = v.Type
	}

	return *req, nil
}
