package processing

import (
	"context"

	casespb "github.com/webitel/flow_manager/gen/cases"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type formSelectCaseStatusArgs struct {
	Id        string `json:"id"`
	ServiceId int64  `json:"serviceId"`
	Token     string `json:"token"`
	View      struct {
		Label        string `json:"label,omitempty"`
		Hint         string `json:"hint,omitempty"`
		InitialValue string `json:"initialValue,omitempty"`
	} `json:"view"`
}

type formSelectCaseStatusComponent struct {
	Id    string                     `json:"id"`
	View  *formSelectCaseStatusView  `json:"view"`
	Value interface{}                `json:"value"`
}

type formSelectCaseStatusView struct {
	Component    string                  `json:"component"`
	Label        string                  `json:"label,omitempty"`
	Hint         string                  `json:"hint,omitempty"`
	InitialValue string                  `json:"initialValue,omitempty"`
	Options      []statusConditionOption `json:"options"`
}

type statusConditionOption struct {
	Id      int64  `json:"id"`
	Name    string `json:"name"`
	Initial bool   `json:"initial"`
	Final   bool   `json:"final"`
}

const appFormSelectCaseStatus = "formSelectCaseStatus"

func (r *Router) formSelectCaseStatus(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv formSelectCaseStatusArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter(appFormSelectCaseStatus, "id")
	}

	if argv.ServiceId == 0 {
		return nil, model.ErrorRequiredParameter(appFormSelectCaseStatus, "serviceId")
	}

	if argv.Token == "" {
		return nil, model.ErrorRequiredParameter(appFormSelectCaseStatus, "token")
	}

	// 1. Locate the service to get its catalog ID
	svcRes, err := r.fm.LocateService(ctx, &casespb.LocateServiceRequest{
		Id:     argv.ServiceId,
		Fields: []string{"id", "catalog_id"},
	}, argv.Token)
	if err != nil {
		return nil, model.NewAppError(appFormSelectCaseStatus, "locate_service_failed", nil, err.Error(), 500)
	}

	svc := svcRes.GetService()
	if svc == nil {
		return nil, model.NewAppError(appFormSelectCaseStatus, "service_not_found", nil, "service not found", 404)
	}

	catalogId := svc.GetCatalogId()
	if catalogId == 0 {
		return nil, model.NewAppError(appFormSelectCaseStatus, "catalog_not_found", nil, "service has no catalog", 404)
	}

	// 2. Locate the catalog to get its linked status dictionary
	catRes, err := r.fm.LocateCatalog(ctx, &casespb.LocateCatalogRequest{
		Id: catalogId,
	}, argv.Token)
	if err != nil {
		return nil, model.NewAppError(appFormSelectCaseStatus, "locate_catalog_failed", nil, err.Error(), 500)
	}

	cat := catRes.GetCatalog()
	if cat == nil || cat.GetStatus() == nil || cat.GetStatus().GetId() == 0 {
		return nil, model.NewAppError(appFormSelectCaseStatus, "status_not_linked", nil, "catalog has no linked status dictionary", 404)
	}

	statusId := cat.GetStatus().GetId()

	// 3. List status conditions from the linked status dictionary
	condRes, err := r.fm.ListStatusConditions(ctx, &casespb.ListStatusConditionRequest{
		StatusId: statusId,
		Size:     100,
	}, argv.Token)
	if err != nil {
		return nil, model.NewAppError(appFormSelectCaseStatus, "list_status_conditions_failed", nil, err.Error(), 500)
	}

	// 4. Build the options from status conditions
	options := make([]statusConditionOption, 0, len(condRes.GetItems()))
	for _, item := range condRes.GetItems() {
		options = append(options, statusConditionOption{
			Id:      item.GetId(),
			Name:    item.GetName(),
			Initial: item.GetInitial(),
			Final:   item.GetFinal(),
		})
	}

	// 5. Build the component
	component := formSelectCaseStatusComponent{
		Id: argv.Id,
		View: &formSelectCaseStatusView{
			Component:    "form-select-case-status",
			Label:        argv.View.Label,
			Hint:         argv.View.Hint,
			InitialValue: argv.View.InitialValue,
			Options:      options,
		},
	}

	// 6. Set existing value if any
	val, _ := conn.Get(argv.Id)
	component.Value = setToJson(val)

	conn.SetComponent(argv.Id, component)

	return model.CallResponseOK, nil
}
