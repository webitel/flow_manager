package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError) {
	return f.Store.Media().GetFiles(domainId, req)
}

func (f *FlowManager) GetMediaFile(domainId int64, id int) (*model.File, *model.AppError) {
	return f.Store.Media().Get(domainId, id)
}
