package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError) {
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

func (f *FlowManager) SearchOutboundToDestinationRouting(domainId int64, dest string) (*model.Routing, *model.AppError) {
	routing, err := f.Store.CallRouting().SearchToDestination(domainId, dest)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = f.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (f *FlowManager) SearchOutboundFromQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError) {
	routing, err := f.Store.CallRouting().FromQueue(domainId, queueId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = f.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (f *FlowManager) TransferQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError) {
	return &model.Routing{
		SourceId:        0,
		SourceName:      "transfer",
		SourceData:      "transfer",
		DomainId:        domainId,
		SchemaId:        0,
		SchemaName:      "transfer queue",
		SchemaUpdatedAt: 0,
		Schema: &model.Schema{
			Id:        0,
			DomainId:  domainId,
			UpdatedAt: 0,
			Type:      "call",
			Name:      "transfer queue",
			Schema: model.Applications{
				{
					"sleep": 500,
				},
				{
					"unSet": []any{"wbt_bt_queue_id", "wbt_bt_queue"},
				},
				{
					"joinQueue": map[string]any{
						"queue": map[string]any{
							"id": queueId,
						},
					},
				},
				{
					"hangup": nil,
				},
			},
			Debug: false,
		},
		Variables: nil,
		Debug:     false,
	}, nil
}

func (f *FlowManager) TransferAgentRouting(domainId int64, agentId int) (*model.Routing, *model.AppError) {
	return &model.Routing{
		SourceId:        0,
		SourceName:      "transfer",
		SourceData:      "transfer",
		DomainId:        domainId,
		SchemaId:        0,
		SchemaName:      "transfer agent",
		SchemaUpdatedAt: 0,
		Schema: &model.Schema{
			Id:        0,
			DomainId:  domainId,
			UpdatedAt: 0,
			Type:      "call",
			Name:      "transfer agent",
			Schema: model.Applications{
				{
					"sleep": 500,
				},
				{
					"unSet": []any{"wbt_bt_agent_id"},
				},
				{
					"joinAgent": map[string]any{
						"agent": map[string]any{
							"id": agentId,
						},
						"queue_name": "transfer",
					},
				},
				{
					"hangup": nil,
				},
			},
			Debug: false,
		},
		Variables: nil,
		Debug:     false,
	}, nil
}
