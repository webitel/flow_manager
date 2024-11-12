package app

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/storage"
	"net/http"
	"strconv"
)

//todo export from storage

func (fm *FlowManager) GenerateSignature(msg []byte) (string, *model.AppError) {
	signature, err := fm.cert.Generate(msg)
	if err != nil {
		return "", model.NewAppError("GenerateSignature", "app.signature.generate.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return signature, nil
}

func (fm *FlowManager) GenerateTTSLink(ctx context.Context, text string, domainId int64, profileId int, textType string, voice string, language string) (string, *model.AppError) {
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
	resp, err := fm.storage.GenerateFileLink(ctx, 0, domainId, "tts", "download", params)

	if err != nil {
		return "", model.NewAppError("GenerateTTSLink", "app.cert.generate_tts_link.get_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil

}

// For possible values for source parameter look to the storage.service -> any_files.go
func (fm *FlowManager) GeneratePreSignedResourceSignature(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, *model.AppError) {

	resp, err := fm.storage.GenerateFileLink(ctx, fileId, domainId, source, action, query)

	if err != nil {
		return "", model.NewAppError("GenerateTTSLink", "app.cert.generate_tts_link.get_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil

}

func (fm *FlowManager) BulkGenerateFileLink(ctx context.Context, domainId int64, files []storage.FileLinkRequest) ([]string, *model.AppError) {

	resp, err := fm.storage.BulkGenerateFileLink(ctx, domainId, files)

	if err != nil {
		return nil, model.NewAppError("BulkGenerateFileLink", "app.cert.generate_file_link.bulk_link.error", nil, err.Error(), http.StatusInternalServerError)
	}
	return resp, nil

}

// Previous version
//func (fm *FlowManager) GeneratePreSignetResourceSignature(resource, action string, uuid string, domainId int64, timeout int64) (string, *model.AppError) {
//	key := fmt.Sprintf("%s/%s?domain_id=%d&uuid=%s&expires=%d", resource, action, domainId, uuid,
//		(model.GetMillis() + timeout))
//
//	signature, err := fm.GenerateSignature([]byte(key))
//	if err != nil {
//		return "", err
//	}
//
//	return key + "&signature=" + signature, nil
//
//}
