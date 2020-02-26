package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetRoutingFromGateway(domainId, gatewayId int) (*model.Routing, *model.AppError) {
	routing, err := f.Store.CallRouting().FromGateway(domainId, gatewayId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = f.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}
