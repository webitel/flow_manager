package app

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError) {
	if source == "" {
		source = "file"
	}

	link, err := fm.GeneratePreSignedResourceSignature(context.Background(), "download", source, int64(file.Id), domainId, map[string]string{"expires": strconv.FormatInt(expire*1000, 10)})
	if err != nil {
		return nil, err
	}

	file.Url = server + link
	return file, nil
}

func (fm *FlowManager) DownloadFile(domainId int64, id int64) (io.ReadCloser, *model.AppError) {
	reader, err := fm.storage.Download(context.TODO(), domainId, id)
	if err != nil {
		return nil, model.NewAppError("DownloadFile", "app.storage.download.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return reader, nil
}

func (fm *FlowManager) GetFileMetadata(domainId int64, ids []int64) ([]model.File, *model.AppError) {
	return fm.Store.File().GetMetadata(domainId, ids)
}
