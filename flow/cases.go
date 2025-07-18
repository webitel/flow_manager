package flow

import (
	"context"
	"encoding/json"

	pb "github.com/webitel/flow_manager/gen/cases"
	"github.com/webitel/flow_manager/model"
)

// ---------------------//
// ** protobuf types ** //
// ---------------------//
type (
	SearchCasesRequest       = pb.SearchCasesRequest
	LocateCaseRequest        = pb.LocateCaseRequest
	CreateCaseRequest        = pb.CreateCaseRequest
	UpdateCaseRequest        = pb.UpdateCaseRequest
	LinkCommunicationRequest = pb.LinkCommunicationRequest
	GetServiceCatalogRequest = pb.ListCatalogRequest
	PublishCommentRequest    = pb.PublishCommentRequest
	CreateLinkRequest        = pb.CreateLinkRequest
	DeleteLinkRequest        = pb.DeleteLinkRequest
	LocateServiceRequest     = pb.LocateServiceRequest
	CreateRelatedCaseRequest = pb.CreateRelatedCaseRequest
)

// ----------------//
// ** Arguments ** //
// ----------------//

type GetCasesArgs struct {
	SearchCasesRequest
	Token  string
	SetVar string
}

type LocateCaseArgs struct {
	LocateCaseRequest
	Token  string
	SetVar string
}

type CreateCaseArgs struct {
	CreateCaseRequest
	Token  string
	SetVar string
}

type UpdateCaseArgs struct {
	UpdateCaseRequest
	Token  string
	SetVar string
}

type LinkCommunicationArgs struct {
	LinkCommunicationRequest
	Token  string
	SetVar string
}

type GetServiceCatalogsArgs struct {
	GetServiceCatalogRequest
	Token  string
	SetVar string
}

type PublishCommentArgs struct {
	PublishCommentRequest
	Token  string
	SetVar string
}

type CreateLinkArgs struct {
	CreateLinkRequest
	Token  string
	SetVar string
}

type DeleteLinkArgs struct {
	DeleteLinkRequest
	Token  string
	SetVar string
}

type LocateServiceArgs struct {
	LocateServiceRequest
	Token  string
	SetVar string
}

type CreateRelatedCaseArgs struct {
	CreateRelatedCaseRequest
	Token  string
	SetVar string
}

// --------------------//
// ** Function names **//
// --------------------//
const (
	funcGetCases           = "getCases"
	funcLocateCase         = "locateCase"
	funcCreateCase         = "createCase"
	funcUpdateCase         = "updateCase"
	funcLinkCommunication  = "linkCommunication"
	funcGetServiceCatalogs = "getServiceCatalogs"
	funcPublishComment     = "publishComment"
	funcCreateLink         = "createLink"
	funcDeleteLink         = "deleteLink"
	funcLocateService      = "locateService"
	funcCreateRelatedCase  = "createRelatedCase"
)

// ** Get Cases **
func (r *router) getCases(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv GetCasesArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if err := checkRequiredFields(argv.Token, argv.SetVar, funcGetCases); err != nil {
		return nil, err
	}

	res, err := r.fm.SearchCases(ctx, &argv.SearchCasesRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcGetCases, "get_cases_failed", nil, err.Error(), 500)
	}

	return setResponse(ctx, conn, argv.SetVar, res)
}

// ** Locate Case **
func (r *router) locateCase(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv LocateCaseArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if err := checkRequiredFields(argv.Token, argv.SetVar, funcLocateCase); err != nil {
		return nil, err
	}

	res, err := r.fm.LocateCase(ctx, &argv.LocateCaseRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcLocateCase, "locate_case_failed", nil, err.Error(), 500)
	}

	return setResponse(ctx, conn, argv.SetVar, res)
}

// ** Create Case **
func (r *router) createCase(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv CreateCaseArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if err := checkRequiredFields(argv.Token, argv.SetVar, funcCreateCase); err != nil {
		return nil, err
	}

	res, err := r.fm.CreateCase(ctx, &argv.CreateCaseRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcCreateCase, "create_case_failed", nil, err.Error(), 500)
	}

	return setResponse(ctx, conn, argv.SetVar, res)
}

// ** Update Case **
func (r *router) updateCase(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv UpdateCaseArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if err := checkRequiredFields(argv.Token, argv.SetVar, funcUpdateCase); err != nil {
		return nil, err
	}

	res, err := r.fm.UpdateCase(ctx, &argv.UpdateCaseRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcUpdateCase, "update_case_failed", nil, err.Error(), 500)
	}

	return setResponse(ctx, conn, argv.SetVar, res)
}

// ** Link Communication **
func (r *router) linkCommunication(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv LinkCommunicationArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcLinkCommunication); err != nil {
		return nil, err
	}

	res, err := r.fm.LinkCommunication(ctx, &argv.LinkCommunicationRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcLinkCommunication, "link_communication_failed", nil, err.Error(), 500)
	}

	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) getServiceCatalogs(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv GetServiceCatalogsArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if err := checkRequiredFields(argv.Token, argv.SetVar, funcGetServiceCatalogs); err != nil {
		return nil, err
	}

	res, err := r.fm.GetServiceCatalogs(ctx, &argv.GetServiceCatalogRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcGetServiceCatalogs, "get_service_catalogs_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) publishComment(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv PublishCommentArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcPublishComment); err != nil {
		return nil, err
	}
	res, err := r.fm.PublishComment(ctx, &argv.PublishCommentRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcPublishComment, "publish_comment_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) createLink(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv CreateLinkArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcCreateLink); err != nil {
		return nil, err
	}
	res, err := r.fm.CreateLink(ctx, &argv.CreateLinkRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcCreateLink, "create_link_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) deleteLink(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv DeleteLinkArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcDeleteLink); err != nil {
		return nil, err
	}
	res, err := r.fm.DeleteLink(ctx, &argv.DeleteLinkRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcDeleteLink, "delete_link_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) locateService(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv LocateServiceArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcLocateService); err != nil {
		return nil, err
	}
	res, err := r.fm.LocateService(ctx, &argv.LocateServiceRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcLocateService, "locate_service_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

func (r *router) createRelatedCase(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	var argv CreateRelatedCaseArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}
	if err := checkRequiredFields(argv.Token, argv.SetVar, funcCreateRelatedCase); err != nil {
		return nil, err
	}
	res, err := r.fm.CreateRelatedCase(ctx, &argv.CreateRelatedCaseRequest, argv.Token)
	if err != nil {
		return nil, model.NewAppError(funcCreateRelatedCase, "create_related_case_failed", nil, err.Error(), 500)
	}
	return setResponse(ctx, conn, argv.SetVar, res)
}

// -------------------//
// ** Helper Methods **//
// -------------------//

// ** Helper function to check required fields and log errors **
func checkRequiredFields(token, setVar, funcName string) *model.AppError {
	if token == "" {
		return model.ErrorRequiredParameter(funcName, "token")
	}

	if setVar == "" {
		return model.ErrorRequiredParameter(funcName, "setVar")
	}
	return nil
}

// ** Function to marshal response and set the variable in connection **
func setResponse(ctx context.Context, conn model.Connection, setVar string, res any) (model.Response, *model.AppError) {
	jsonData, err := json.Marshal(res)
	if err != nil {
		conn.Log().Error(err.Error())
		return nil, model.NewAppError("json_encode_failed", "json_encode_failed", nil, err.Error(), 500)
	}

	// If a result is empty JSON object or array, set variable to nil
	str := string(jsonData)
	if str == "{}" || str == "[]" || str == `""` {
		return conn.Set(ctx, model.Variables{
			setVar: "",
		})
	}

	return conn.Set(ctx, model.Variables{
		setVar: str,
	})
}
