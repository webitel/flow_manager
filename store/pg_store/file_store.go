package sqlstore

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlFileStore struct {
	SqlStore
}

func NewSqlFileStore(sqlStore SqlStore) store.FileStore {
	st := &SqlFileStore{sqlStore}
	return st
}

func (s SqlFileStore) GetMetadata(domainId int64, ids []int64) ([]model.File, *model.AppError) {
	var files []model.File
	_, err := s.GetReplica().Select(&files, `
		select f.id, f.name, f.size, f.mime_type, f.view_name
		from storage.files f
		where f.domain_id = :DomainId and f.id = any(:Ids::int8[])
	`, map[string]interface{}{
		"DomainId": domainId,
		"Ids":      pq.Array(ids),
	})

	if err != nil {
		return nil, model.NewAppError("SqlFileStore.GetMetadata", "store.sql_file.metadata.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), extractCodeFromErr(err))
	}

	return files, nil
}
