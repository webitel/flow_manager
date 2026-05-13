package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type MediaRepository struct {
	db infraSql.Store
}

func NewMediaRepository(db infraSql.Store) store.MediaStore {
	return &MediaRepository{db: db}
}

type fileRow struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	Size     int64  `db:"size"`
	MimeType string `db:"mime_type"`
}

func toFile(r fileRow) *model.File {
	return &model.File{Id: r.Id, Name: r.Name, Size: r.Size, MimeType: r.MimeType}
}

const getFileSQL = `
SELECT f.id, f.name, f.size, f.mime_type
  FROM storage.media_files f
 WHERE f.domain_id = @domain_id AND f.id = @id`

func (r *MediaRepository) Get(domainId int64, id int) (*model.File, error) {
	var row fileRow
	if err := r.db.Get(context.Background(), &row, getFileSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"id":        id,
	}); err != nil {
		return nil, err
	}
	return toFile(row), nil
}

const searchOneSQL = `
SELECT f.id, f.name, f.size, f.mime_type
  FROM storage.media_files f
 WHERE f.domain_id = @domain_id AND (f.id = @id OR f.name = @name)
 LIMIT 1`

func (r *MediaRepository) SearchOne(domainId int64, search *model.SearchFile) (*model.File, error) {
	var row fileRow
	if err := r.db.Get(context.Background(), &row, searchOneSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"id":        search.Id,
		"name":      search.Name,
	}); err != nil {
		return nil, err
	}
	return toFile(row), nil
}

type playbackRow struct {
	Idx  *int32  `db:"idx"`
	Id   *int32  `db:"id"`
	Type *string `db:"type"`
}

// getFilesSQL uses positional params: $1=names(varchar[]), $2=ids(int4[]), $3=domain_id
const getFilesSQL = `
SELECT rec.id - 1 AS idx, m.id, m.mime_type AS type
FROM (
    SELECT id, x, ($1::varchar[])[id] AS name
    FROM unnest($2::int4[]) WITH ORDINALITY ids (x, id)
) rec
LEFT JOIN LATERAL (
    SELECT m.id, m.mime_type
    FROM storage.media_files m
    WHERE m.domain_id = $3
      AND NOT (rec.x::int8 IS NULL AND rec.name::varchar IS NULL)
      AND (
          (rec.x::int8 IS NOT NULL AND m.id = rec.x)
          OR (rec.name::varchar IS NOT NULL AND m.name = rec.name)
      )
    LIMIT 1
) m ON TRUE`

func (r *MediaRepository) GetFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, error) {
	pgIds := make([]pgtype.Int4, len(*req))
	pgNames := make([]pgtype.Text, len(*req))

	for i, v := range *req {
		if v != nil && v.Type == nil && (v.Id != nil || v.Name != nil) {
			if v.Id != nil {
				pgIds[i] = pgtype.Int4{Int32: int32(*v.Id), Valid: true}
			}
			if v.Name != nil {
				pgNames[i] = pgtype.Text{String: *v.Name, Valid: true}
			}
		}
	}

	var out []playbackRow
	if err := r.db.SelectArgs(context.Background(), &out, getFilesSQL, pgNames, pgIds, domainId); err != nil {
		return nil, err
	}

	for _, row := range out {
		if row.Type == nil || row.Idx == nil || (*req)[*row.Idx] == nil {
			continue
		}
		id := int(*row.Id)
		(*req)[*row.Idx].Id = &id
		(*req)[*row.Idx].Type = row.Type
	}

	return *req, nil
}

const getPlaybackFileSQL = `
SELECT 0 AS idx, m.id, m.mime_type AS type
  FROM storage.media_files m
 WHERE m.domain_id = @domain_id
   AND (m.id = @id OR m.name = @name)
 LIMIT 1`

func (r *MediaRepository) GetPlaybackFile(domainId int64, req *model.PlaybackFile) (*model.PlaybackFile, error) {
	var row playbackRow
	if err := r.db.Get(context.Background(), &row, getPlaybackFileSQL, pgx.NamedArgs{
		"domain_id": domainId,
		"id":        req.Id,
		"name":      req.Name,
	}); err != nil {
		return nil, err
	}
	if row.Id != nil {
		id := int(*row.Id)
		req.Id = &id
		req.Type = row.Type
	}
	return req, nil
}
