package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

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

func (fm *FlowManager) GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError) {
	routing, err := fm.Store.Chat().RoutingFromSchemaId(domainId, schemaId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = fm.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (fm *FlowManager) GetChatRouteFromUserId(domainId int64, userId int64) (*model.Routing, *model.AppError) {
	routing := &model.Routing{
		SourceId:   0,
		SourceName: "Blind transfer to user",
		SourceData: "Blind transfer to user",
		DomainId:   domainId,
		Schema: &model.Schema{
			DomainId: domainId,
			Name:     "transfer to user",
			Schema: model.Applications{
				{
					"bridge": map[string]interface{}{
						"userId": userId,
					},
				},
			},
		},
	}

	return routing, nil
}

func (fm *FlowManager) BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat) *model.AppError {
	err := fm.chatManager.BroadcastMessage(ctx, domainId, req)
	if err != nil {
		return model.NewAppError("Chat", "chat.broadcast.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
