package app

import (
	"strconv"

	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError) {
	link, err := fm.GeneratePreSignetResourceSignature("/any/file", "download", strconv.Itoa(file.Id), domainId, expire*1000)
	if err != nil {
		return nil, err
	}

	if source != "" {
		link += "&source=" + source
	}

	file.Url = server + link
	return file, nil
}
