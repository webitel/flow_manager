package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError) {
	res, err := f.Store.Media().GetFiles(domainId, req)
	return res, toAppError("App.GetMediaFiles", err)
}

func (f *FlowManager) GetMediaFile(domainId int64, id int) (*model.File, *model.AppError) {
	res, err := f.Store.Media().Get(domainId, id)
	return res, toAppError("App.GetMediaFile", err)
}

func (f *FlowManager) SearchMediaFile(domainId int64, search *model.SearchFile) (*model.File, *model.AppError) {
	res, err := f.Store.Media().SearchOne(domainId, search)
	return res, toAppError("App.SearchMediaFile", err)
}

func (f *FlowManager) GetPlaybackFile(domainId int64, search *model.PlaybackFile) (*model.PlaybackFile, *model.AppError) {
	res, err := f.Store.Media().GetPlaybackFile(domainId, search)
	return res, toAppError("App.GetPlaybackFile", err)
}
