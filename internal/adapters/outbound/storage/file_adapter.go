package storage

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/webitel/flow_manager/internal/domain/files"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
)

// FileAdapter implements storage-backed Deps methods (file links, downloads,
// transcription). All methods delegate to the storage gRPC client only.
type FileAdapter struct {
	storage domstorage.Client
}

func NewFileAdapter(s domstorage.Client) *FileAdapter {
	return &FileAdapter{storage: s}
}

func (a *FileAdapter) GenerateTTSLink(ctx context.Context, text string, domainId int64, profileId int, textType, voice, language string) (string, error) {
	params := map[string]string{
		"text":       text,
		"profile_id": strconv.Itoa(profileId),
	}
	if textType != "" {
		params["text_type"] = textType
	}
	if voice != "" {
		params["voice"] = voice
	}
	if language != "" {
		params["language"] = language
	}
	resp, err := a.storage.GenerateFileLink(ctx, 0, domainId, "tts", "download", params)
	if err != nil {
		return "", fmt.Errorf("GenerateTTSLink: app.cert.generate_tts_link.get_link.error: %w", err)
	}
	return resp, nil
}

func (a *FileAdapter) generateResourceSignature(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, error) {
	resp, err := a.storage.GenerateFileLink(ctx, fileId, domainId, source, action, query)
	if err != nil {
		return "", fmt.Errorf("GeneratePreSignedLink: app.cert.generate_tts_link.get_link.error: %w", err)
	}
	return resp, nil
}

func (a *FileAdapter) GeneratePreSignedLink(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, error) {
	return a.generateResourceSignature(ctx, action, source, fileId, domainId, query)
}

func (a *FileAdapter) SetupPublicFileUrl(file *files.File, domainId int64, server, source string, expire int64) (*files.File, error) {
	if source == "" {
		source = "file"
	}
	link, err := a.generateResourceSignature(context.Background(), "download", source, int64(file.Id), domainId,
		map[string]string{"expires": strconv.FormatInt(expire*1000, 10)})
	if err != nil {
		return nil, err
	}
	file.Url = server + link
	return file, nil
}

func (a *FileAdapter) DownloadFile(domainId int64, id int64) (io.ReadCloser, error) {
	return a.storage.Download(context.TODO(), domainId, id)
}

func (a *FileAdapter) GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, error) {
	resp, err := a.storage.GetFileTranscription(ctx, fileId, domainId, profileId, language)
	if err != nil {
		return "", fmt.Errorf("GetFileTranscription: app.cert.generate_tts_link.get_link.error: %w", err)
	}
	return resp, nil
}

func (a *FileAdapter) BulkGenerateFileLink(ctx context.Context, domainId int64, fileLinks []files.FileLinkRequest) ([]string, error) {
	reqs := make([]domstorage.FileLinkRequest, len(fileLinks))
	for i, f := range fileLinks {
		reqs[i] = domstorage.FileLinkRequest{FileId: f.FileId, Action: f.Action, Source: f.Source}
	}
	resp, err := a.storage.BulkGenerateFileLink(ctx, domainId, reqs)
	if err != nil {
		return nil, fmt.Errorf("BulkGenerateFileLink: app.cert.generate_file_link.bulk_link.error: %w", err)
	}
	return resp, nil
}
