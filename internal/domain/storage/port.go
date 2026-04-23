package storage

import (
	"context"
	"io"
)

type Client interface {
	Upload(ctx context.Context, domainId int64, uuid string, r io.Reader, f File) (File, error)
	Download(ctx context.Context, domainId, id int64) (io.ReadCloser, error)
	GenerateFileLink(ctx context.Context, fileId, domainId int64, source, action string, query map[string]string) (string, error)
	BulkGenerateFileLink(ctx context.Context, domainId int64, files []FileLinkRequest) ([]string, error)
	GetFileTranscription(ctx context.Context, fileId, domainId, profileId int64, language string) (string, error)
}
