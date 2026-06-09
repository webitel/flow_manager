package im

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (r *Router) Menu(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv model.ChatMenuArgs
	argv.Type = "buttons"

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	switch argv.Type {
	case "buttons", "inline":
	default:
		return model.CallResponseError, nil
	}

	return conv.SendMenu(ctx, &argv)
}

type MenuButton struct {
	model.KeyboardButton `json:",inline"`

	NextFlow []any `json:"nextFlow,omitempty"`
}

type MenuArgs struct {
	model.SendInteractiveRequestGeneric[MenuButton] `json:",inline"`

	Timeout        int    `json:"timeout"`
	MessageTimeout int    `json:"messageTimeout"`
	Set            string `json:"set"`
	Default        []any  `json:"default"`
}

func ConvertMenuRequestToStandard(menuReq model.SendInteractiveRequestGeneric[MenuButton]) model.SendInteractiveRequest {
	var standardMarkup *model.KeyboardMarkup
	var standardListReply *model.KeyboardListReply

	if menuReq.Interactive.Markup != nil {
		standardRows := make([]model.KeyboardRowGeneric[model.KeyboardButton], len(menuReq.Interactive.Markup.Rows))
		for i, menuRow := range menuReq.Interactive.Markup.Rows {
			standardButtons := make([]model.KeyboardButton, len(menuRow.Buttons))
			for j, menuBtn := range menuRow.Buttons {
				standardButtons[j] = menuBtn.KeyboardButton
			}
			standardRows[i] = model.KeyboardRowGeneric[model.KeyboardButton]{
				Buttons: standardButtons,
			}
		}
		standardMarkup = &model.KeyboardMarkup{Rows: standardRows}
	}

	if menuReq.Interactive.ListReply != nil {
		standardSections := make([]model.KeyboardRowWithSectionGeneric[model.KeyboardButton], len(menuReq.Interactive.ListReply.Sections))
		for i, menuSection := range menuReq.Interactive.ListReply.Sections {
			standardButtons := make([]model.KeyboardButton, len(menuSection.Buttons))
			for j, menuBtn := range menuSection.Buttons {
				standardButtons[j] = menuBtn.KeyboardButton
			}
			standardSections[i] = model.KeyboardRowWithSectionGeneric[model.KeyboardButton]{
				Section: menuSection.Section,
				Buttons: standardButtons,
			}
		}
		standardListReply = &model.KeyboardListReply{
			MainButtonTitle: menuReq.Interactive.ListReply.MainButtonTitle,
			Sections:        standardSections,
		}
	}

	return model.SendInteractiveRequest{
		Body:     menuReq.Body,
		Metadata: menuReq.Metadata,
		Interactive: model.InteractiveGeneric[model.KeyboardButton]{
			Documents: menuReq.Interactive.Documents,
			Images:    menuReq.Interactive.Images,
			SingleUse: menuReq.Interactive.SingleUse,
			Markup:    standardMarkup,
			ListReply: standardListReply,
		},
	}
}

func NewReceiveMenuArgs(properties any) (*MenuArgs, *model.AppError) {
	bytes, err := json.Marshal(properties)
	if err != nil {
		return nil, model.NewAppError("IM.NewReceiveMenuArgs", "flow.menu.marshal_err", nil, err.Error(), http.StatusInternalServerError)
	}

	var args MenuArgs
	if err := json.Unmarshal(bytes, &args); err != nil {
		return nil, model.NewAppError("IM.NewReceiveMenuArgs", "flow.menu.unmarshal_err", nil, err.Error(), http.StatusBadRequest)
	}

	return &args, nil
}

func (r *Router) MenuWithReceive(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	req, err := NewReceiveMenuArgs(args)
	if err != nil {
		return nil, err
	}

	if _, err := conv.SendInteractive(ctx, ConvertMenuRequestToStandard(req.SendInteractiveRequestGeneric)); err != nil {
		return nil, err
	}

	events, appErr := conv.ReceiveMessage(ctx, req.Set, req.Timeout, req.MessageTimeout)
	if appErr != nil {
		return nil, appErr
	}

	if len(events) == 0 {
		return nil, model.NewAppError("Flow.MenuWithReceive", "flow.menu.empty_event", nil, "no event received", http.StatusInternalServerError)
	}

	if _, err := conv.Set(ctx, model.Variables{req.Set: strings.Join(events, " ")}); err != nil {
		wlog.Error("setting received message as variable", wlog.String("conversation_id", conv.Id()), wlog.Err(err))
	}

	clickedButtonID := events[0]
	var clickedButton *MenuButton

	if req.Interactive.Markup != nil {
		for i := range req.Interactive.Markup.Rows {
			for j := range req.Interactive.Markup.Rows[i].Buttons {
				if req.Interactive.Markup.Rows[i].Buttons[j].ID == clickedButtonID {
					clickedButton = &req.Interactive.Markup.Rows[i].Buttons[j]
					break
				}
			}
			if clickedButton != nil {
				break
			}
		}
	}

	if clickedButton == nil && req.Interactive.ListReply != nil {
		for i := range req.Interactive.ListReply.Sections {
			for j := range req.Interactive.ListReply.Sections[i].Buttons {
				if req.Interactive.ListReply.Sections[i].Buttons[j].ID == clickedButtonID {
					clickedButton = &req.Interactive.ListReply.Sections[i].Buttons[j]
					break
				}
			}
			if clickedButton != nil {
				break
			}
		}
	}

	if clickedButton != nil && len(clickedButton.NextFlow) > 0 {
		forkedBranch := scope.Fork(clickedButtonID, flow.ArrInterfaceToArrayApplication(clickedButton.NextFlow))
		flow.Route(ctx, forkedBranch, r)

		wlog.Debug("menu routing set to button node", wlog.String("conversation_id", conv.Id()), wlog.String("clicked_button_id", clickedButtonID))
		scope.SetCancel()
		return model.CallResponseOK, nil
	}

	if req.Default != nil {
		forkedBranch := scope.Fork(clickedButtonID, flow.ArrInterfaceToArrayApplication(req.Default))
		flow.Route(ctx, forkedBranch, r)

		wlog.Debug("menu routing set to default node", wlog.String("conversation_id", conv.Id()))
		scope.SetCancel()
		return model.CallResponseOK, nil
	}

	wlog.Debug("menu button has no explicit next node, continuing standard flow", wlog.String("conversation_id", conv.Id()), wlog.String("clicked_button_id", clickedButtonID))
	return model.CallResponseOK, nil
}
