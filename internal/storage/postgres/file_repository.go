package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/files"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/storage"
)

type FileRepository struct {
	db infraSql.Store
}

func NewFileRepository(db infraSql.Store) storage.FileStore {
	return &FileRepository{db: db}
}

const getFileMetadataSQL = `select f.id, f.name, f.size, f.mime_type, f.view_name
from storage.files f
where f.domain_id = @DomainId and f.id = any(@Ids::int8[])`

func (r *FileRepository) GetMetadata(domainId int64, ids []int64) ([]files.File, error) {
	var result []files.File
	if err := r.db.Select(context.Background(), &result, getFileMetadataSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Ids":      ids,
	}); err != nil {
		return nil, fmt.Errorf("domainId=%v: %w", domainId, err)
	}
	return result, nil
}
