package storage

import (
	"context"
	"io"
	"net/http"
	"strconv"

	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/model"
)

// FileAdapter implements storage-backed Deps methods (file links, downloads,
// transcription). All methods delegate to the storage gRPC client only.
type FileAdapter struct {
	storage domstorage.Client
}

func NewFileAdapter(s domstorage.Client) *FileAdapter {
	return &FileAdapter{storage: s}
}

func (a *FileAdapter) GenerateTTSLink(ctx context.Context, text string, domainId int64, profileId int, textType, voice, language string) (string, *model.AppError) {
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
		return "", model.NewAppError("GenerateTTSLink", "app.cert.generate_tts_link.get_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil
}

func (a *FileAdapter) generateResourceSignature(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, *model.AppError) {
	resp, err := a.storage.GenerateFileLink(ctx, fileId, domainId, source, action, query)
	if err != nil {
		return "", model.NewAppError("GeneratePreSignedLink", "app.cert.generate_tts_link.get_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil
}

func (a *FileAdapter) GeneratePreSignedLink(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, error) {
	link, appErr := a.generateResourceSignature(ctx, action, source, fileId, domainId, query)
	if appErr != nil {
		return "", appErr
	}
	return link, nil
}

func (a *FileAdapter) SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError) {
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

func (a *FileAdapter) GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, *model.AppError) {
	resp, err := a.storage.GetFileTranscription(ctx, fileId, domainId, profileId, language)
	if err != nil {
		return "", model.NewAppError("GetFileTranscription", "app.cert.generate_tts_link.get_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil
}

func (a *FileAdapter) BulkGenerateFileLink(ctx context.Context, domainId int64, files []model.FileLinkRequest) ([]string, *model.AppError) {
	reqs := make([]domstorage.FileLinkRequest, len(files))
	for i, f := range files {
		reqs[i] = domstorage.FileLinkRequest{FileId: f.FileId, Action: f.Action, Source: f.Source}
	}
	resp, err := a.storage.BulkGenerateFileLink(ctx, domainId, reqs)
	if err != nil {
		return nil, model.NewAppError("BulkGenerateFileLink", "app.cert.generate_file_link.bulk_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil
}
