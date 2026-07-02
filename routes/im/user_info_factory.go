package im

import (
	"context"
	"errors"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type IMUserInfoRequestType string

const (
	IMUserInfoRequestTypeDevice IMUserInfoRequestType = "device"
	IMUserInfoRequestTypeGate   IMUserInfoRequestType = "gate"
)

type IMUserInfoHandler interface {
	Type() IMUserInfoRequestType
	Handle(ctx context.Context, dialog Dialog) (string, *model.AppError)
}

type IMUserInfoHandlerFactory struct {
	handlers map[IMUserInfoRequestType]IMUserInfoHandler
}

func NewIMUserInfoHandlerFactory(handlers ...IMUserInfoHandler) *IMUserInfoHandlerFactory {
	factory := &IMUserInfoHandlerFactory{handlers: make(map[IMUserInfoRequestType]IMUserInfoHandler)}

	for _, h := range handlers {
		factory.Register(h)
	}

	return factory
}

func (f *IMUserInfoHandlerFactory) Register(handler IMUserInfoHandler) {
	f.handlers[handler.Type()] = handler
}

func (f *IMUserInfoHandlerFactory) Handle(ctx context.Context, requestType IMUserInfoRequestType, dialog Dialog) (string, *model.AppError) {
	handler, exists := f.handlers[requestType]
	if !exists {
		return "", model.NewAppError(
			"IMUserInfoHandlerFactory.Handle",
			"im.user_info.handler_factory.handle",
			map[string]any{"type": string(requestType)},
			"received unregistered im user info request type",
			404,
		)
	}

	return handler.Handle(ctx, dialog)
}

type IMUserInfoDeviceHandler struct{}

func (h *IMUserInfoDeviceHandler) Type() IMUserInfoRequestType { return IMUserInfoRequestTypeDevice }

func (h *IMUserInfoDeviceHandler) Handle(ctx context.Context, dialog Dialog) (string, *model.AppError) {
	deviceID := dialog.DeviceID()
	if deviceID == "" {
		return "", model.NewRequestError("IMUserInfoDeviceHandler.Handle", "received empty device id call")
	}

	session, err := dialog.GetAuthSession(ctx, deviceID)
	if err != nil {
		if errors.Is(err, model.ErrAuthSesionNotFound) {
			wlog.Warn("zero devices found for requested device id", wlog.String("device_id", deviceID), wlog.Err(err))
			return "{}", nil
		}

		return "", err
	}

	serialized, err := session.Serialize()
	if err != nil {
		return "", err
	}

	return serialized, nil
}

type IMUserInfoGateHandler struct{}

func (h *IMUserInfoGateHandler) Type() IMUserInfoRequestType { return IMUserInfoRequestTypeGate }

func (h *IMUserInfoGateHandler) Handle(ctx context.Context, dialog Dialog) (string, *model.AppError) {
	via := dialog.Via()
	if via == "" {
		return "", model.NewRequestError("IMUserInfoGateHandler.Handle", "received call with empty via info")
	}

	gateType := model.IMGateTypeFromString(dialog.From().Issuer)
	gate, err := dialog.HandleGateInfo(ctx, gateType, via)
	if err != nil {
		return "", err
	}

	serialized, werr := gate.MarshalJSON()
	if werr != nil {
		return "", model.NewAppError("IMUserInfoGateHandler.Handle", "im.user_info_factory.handle.marshaling_gate", nil, werr.Error(), 500)
	}

	return string(serialized), nil
}
