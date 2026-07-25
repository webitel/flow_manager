package im

import (
	"cmp"
	"context"

	providers "github.com/webitel/flow_manager/gen/im/service/provider/v1"
	"github.com/webitel/flow_manager/model"
)

type GateHandler interface {
	Type() model.IMGateType
	Handle(ctx context.Context, id string) (*model.IMGate, *model.AppError)
}

type GateHandlerFactory struct {
	handlers map[model.IMGateType]GateHandler
}

func NewGateHandlerFactory(handlers ...GateHandler) *GateHandlerFactory {
	factory := &GateHandlerFactory{handlers: make(map[model.IMGateType]GateHandler)}

	for _, h := range handlers {
		factory.Register(h)
	}

	return factory
}

func (f *GateHandlerFactory) Register(handler GateHandler) {
	f.handlers[handler.Type()] = handler
}

func (f *GateHandlerFactory) Handle(ctx context.Context, gateType model.IMGateType, id string) (*model.IMGate, *model.AppError) {
	if handler, exists := f.handlers[gateType]; exists {
		return handler.Handle(ctx, id)
	}

	return nil, model.NewAppError(
		"GateHandlerFactory.Handle",
		"im.providers_factory.zero_handlers",
		map[string]any{"type": string(gateType)},
		"resolved zero handlers for provided type",
		404,
	)
}

type FacebookGateHandler struct {
	client *Client
}

func NewFacebookGateHandler(client *Client) *FacebookGateHandler {
	return &FacebookGateHandler{client: client}
}

func (f *FacebookGateHandler) Type() model.IMGateType { return model.IMGateTypeFacebook }
func (f *FacebookGateHandler) Handle(ctx context.Context, id string) (*model.IMGate, *model.AppError) {
	response, err := f.client.facebookService.Api.GetFacebookGate(ctx, &providers.ProviderGetFacebookGateRequest{Id: id})
	if err != nil {
		return nil, model.NewAppError(
			"FacebookGateHandler.Handle",
			"providers.im.providers_factory.request",
			nil,
			err.Error(),
			model.ExtractHTPPStatusCodeFromGRPC(err),
		)
	}

	facebookInfo := &model.GateFacebook{
		ID:        response.Item.GetId(),
		Name:      response.GetItem().GetName(),
		MetaAppID: response.GetItem().GetMetaAppId(),
		PageID:    response.GetItem().GetPageId(),
		PageName:  cmp.Or(response.GetItem().GetPageName(), response.GetItem().GetName()),
	}

	return &model.IMGate{
		Type:    model.IMGateTypeFacebook,
		Payload: facebookInfo,
	}, nil
}

type ViberGateHandler struct {
	client *Client
}

func NewViberGateHandler(client *Client) *ViberGateHandler {
	return &ViberGateHandler{client: client}
}

func (v *ViberGateHandler) Type() model.IMGateType { return model.IMGateTypeViber }
func (v *ViberGateHandler) Handle(ctx context.Context, id string) (*model.IMGate, *model.AppError) {
	response, err := v.client.viberService.Api.GetViberGate(ctx, &providers.ProviderGetViberGateRequest{Id: id})
	if err != nil {
		return nil, model.NewAppError(
			"ViberGateHandler.Handle",
			"providers.im.providers_factory.request",
			nil,
			err.Error(),
			model.ExtractHTPPStatusCodeFromGRPC(err),
		)
	}

	viberInfo := &model.GateViber{
		ID:         response.GetItem().GetId(),
		Name:       response.GetItem().GetName(),
		BotID:      response.GetItem().GetBotId(),
		BotURI:     response.GetItem().GetBotUri(),
		SenderName: response.GetItem().GetSenderName(),
	}

	return &model.IMGate{
		Type:    model.IMGateTypeViber,
		Payload: viberInfo,
	}, nil
}
