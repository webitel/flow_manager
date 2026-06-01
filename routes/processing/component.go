package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	casespb "github.com/webitel/flow_manager/gen/cases"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/pkg/processing"
)

const appFormComponent = "formComponent"

func (r *Router) formComponent(ctx context.Context, scope *flow.Flow, conn Connection, args any) (model.Response, *model.AppError) {
	var argv processing.FormComponent
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter(appFormComponent, "name")
	}

	val, _ := conn.Get(argv.Id)

	switch argv.View.Component {
	case "wt-input": // TODO DEV-5230
		argv.Value = val
	case "form-select-case-status": // TODO WTEL-9157
		argv.View.Options, err = r.casesStatusOptions(ctx, argv.View.Token, argv.View.ServiceId)
		if err != nil {
			return nil, err
		}
		fallthrough

	default:
		argv.Value = setToJson(val)
	}

	conn.SetComponent(argv.Id, argv)

	return model.CallResponseOK, nil
}

func (r *Router) casesStatusOptions(ctx context.Context, token string, serviceId int64) ([]processing.SelectOption, *model.AppError) {
	if serviceId == 0 {
		return nil, model.ErrorRequiredParameter(appFormComponent, "serviceId")
	}

	if token == "" {
		return nil, model.ErrorRequiredParameter(appFormComponent, "token")
	}

	// 1. Locate the service to get its catalog ID
	svcRes, err := r.fm.LocateService(ctx, &casespb.LocateServiceRequest{
		Id:     serviceId,
		Fields: []string{"id", "catalog_id"},
	}, token)
	if err != nil {
		return nil, model.NewAppError(appFormComponent, "locate_service_failed", nil, err.Error(), 500)
	}

	svc := svcRes.GetService()
	if svc == nil {
		return nil, model.NewAppError(appFormComponent, "service_not_found", nil, "service not found", 404)
	}

	catalogId := svc.GetCatalogId()
	if catalogId == 0 {
		return nil, model.NewAppError(appFormComponent, "catalog_not_found", nil, "service has no catalog", 404)
	}

	// 2. Locate the catalog to get its linked status dictionary
	catRes, err := r.fm.LocateCatalog(ctx, &casespb.LocateCatalogRequest{
		Id: catalogId,
	}, token)
	if err != nil {
		return nil, model.NewAppError(appFormComponent, "locate_catalog_failed", nil, err.Error(), 500)
	}

	cat := catRes.GetCatalog()
	if cat == nil || cat.GetStatus() == nil || cat.GetStatus().GetId() == 0 {
		return nil, model.NewAppError(appFormComponent, "status_not_linked", nil, "catalog has no linked status dictionary", 404)
	}

	statusId := cat.GetStatus().GetId()

	// 3. List status conditions from the linked status dictionary
	condRes, err := r.fm.ListStatusConditions(ctx, &casespb.ListStatusConditionRequest{
		StatusId: statusId,
		Size:     100,
	}, token)
	if err != nil {
		return nil, model.NewAppError(appFormComponent, "list_status_conditions_failed", nil, err.Error(), 500)
	}

	// 4. Build the options from status conditions
	options := make([]processing.SelectOption, 0, len(condRes.GetItems()))
	for _, item := range condRes.GetItems() {
		options = append(options, processing.SelectOption{
			Id:      int(item.GetId()),
			Name:    item.GetName(),
			Initial: item.GetInitial(),
			Final:   item.GetFinal(),
		})
	}

	return options, nil
}
