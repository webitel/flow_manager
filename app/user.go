package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) GetUserProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError) {
	res, err := f.Store.User().GetProperties(domainId, search, mapRes)
	if err != nil {
		return nil, model.NewAppError("GetUserProperties", "store.user.get_properties", nil, err.Error(), http.StatusInternalServerError)
	}
	return res, nil
}

func (f *FlowManager) GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError) {
	res, err := f.Store.User().GetAgentIdByExtension(domainId, extension)
	if err != nil {
		return nil, model.NewAppError("GetAgentIdByExtension", "store.user.get_agent_id_by_extension", nil, err.Error(), http.StatusInternalServerError)
	}
	return res, nil
}
