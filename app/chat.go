package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) GetChatRouteFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError) {
	routing, err := fm.Store.Chat().RoutingFromProfile(domainId, profileId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = fm.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}
