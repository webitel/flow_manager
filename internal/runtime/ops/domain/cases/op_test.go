package cases

import (
	"context"
	"fmt"
	"testing"

	cases2 "github.com/webitel/flow_manager/api/gen/cases"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── fakeCasesClient ───────────────────────────────────────────────────────────

type fakeCasesClient struct {
	searchResult   *cases2.CaseList
	searchErr      error
	locateResult   *cases2.Case
	locateErr      error
	createResult   *cases2.Case
	createErr      error
	updateResult   *cases2.UpdateCaseResponse
	updateErr      error
}

func (f *fakeCasesClient) SearchCases(_ context.Context, _ *cases2.SearchCasesRequest, _ string) (*cases2.CaseList, error) {
	return f.searchResult, f.searchErr
}
func (f *fakeCasesClient) LocateCase(_ context.Context, _ *cases2.LocateCaseRequest, _ string) (*cases2.Case, error) {
	return f.locateResult, f.locateErr
}
func (f *fakeCasesClient) CreateCase(_ context.Context, _ *cases2.CreateCaseRequest, _ string) (*cases2.Case, error) {
	return f.createResult, f.createErr
}
func (f *fakeCasesClient) UpdateCase(_ context.Context, _ *cases2.UpdateCaseRequest, _ string) (*cases2.UpdateCaseResponse, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeCasesClient) LinkCommunication(_ context.Context, _ *cases2.LinkCommunicationRequest, _ string) (*cases2.LinkCommunicationResponse, error) {
	return &cases2.LinkCommunicationResponse{}, nil
}
func (f *fakeCasesClient) GetServiceCatalogs(_ context.Context, _ *cases2.ListCatalogRequest, _ string) (*cases2.CatalogList, error) {
	return &cases2.CatalogList{}, nil
}
func (f *fakeCasesClient) PublishComment(_ context.Context, _ *cases2.PublishCommentRequest, _ string) (*cases2.CaseComment, error) {
	return &cases2.CaseComment{}, nil
}
func (f *fakeCasesClient) CreateLink(_ context.Context, _ *cases2.CreateLinkRequest, _ string) (*cases2.CaseLink, error) {
	return &cases2.CaseLink{}, nil
}
func (f *fakeCasesClient) DeleteLink(_ context.Context, _ *cases2.DeleteLinkRequest, _ string) (*cases2.CaseLink, error) {
	return &cases2.CaseLink{}, nil
}
func (f *fakeCasesClient) LocateService(_ context.Context, _ *cases2.LocateServiceRequest, _ string) (*cases2.LocateServiceResponse, error) {
	return &cases2.LocateServiceResponse{}, nil
}
func (f *fakeCasesClient) CreateRelatedCase(_ context.Context, _ *cases2.CreateRelatedCaseRequest, _ string) (*cases2.RelatedCase, error) {
	return &cases2.RelatedCase{}, nil
}
func (f *fakeCasesClient) ListCaseFiles(_ context.Context, _ *cases2.ListFilesRequest, _ string) (*cases2.CaseFileList, error) {
	return &cases2.CaseFileList{}, nil
}
func (f *fakeCasesClient) LocateCatalog(_ context.Context, _ *cases2.LocateCatalogRequest, _ string) (*cases2.LocateCatalogResponse, error) {
	return &cases2.LocateCatalogResponse{}, nil
}
func (f *fakeCasesClient) ListStatusConditions(_ context.Context, _ *cases2.ListStatusConditionRequest, _ string) (*cases2.StatusConditionList, error) {
	return &cases2.StatusConditionList{}, nil
}

var _ domcases.Client = (*fakeCasesClient)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func casesInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

func okCasesClient() *fakeCasesClient {
	return &fakeCasesClient{
		searchResult: &cases2.CaseList{},
		locateResult: &cases2.Case{},
		createResult: &cases2.Case{},
		updateResult: &cases2.UpdateCaseResponse{},
	}
}

// ── getCases ──────────────────────────────────────────────────────────────────

func TestGetCases_NoToken(t *testing.T) {
	op := &getCasesOp{client: okCasesClient()}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestGetCases_NoSetVar(t *testing.T) {
	op := &getCasesOp{client: okCasesClient()}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{"token": "tok"}))
	if err == nil {
		t.Fatal("expected error when setVar is missing")
	}
}

func TestGetCases_Success(t *testing.T) {
	op := &getCasesOp{client: okCasesClient()}
	out, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "cases",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["cases"]; !ok {
		t.Error("expected SetVars[cases] to be set")
	}
}

func TestGetCases_DepError(t *testing.T) {
	cli := &fakeCasesClient{searchErr: fmt.Errorf("unavailable")}
	op := &getCasesOp{client: cli}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "cases",
	}))
	if err == nil {
		t.Fatal("expected error when SearchCases fails")
	}
}

// ── locateCase ────────────────────────────────────────────────────────────────

func TestLocateCase_NoToken(t *testing.T) {
	op := &locateCaseOp{client: okCasesClient()}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestLocateCase_Success(t *testing.T) {
	op := &locateCaseOp{client: okCasesClient()}
	out, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "case_data",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["case_data"]; !ok {
		t.Error("expected SetVars[case_data] to be set")
	}
}

func TestLocateCase_DepError(t *testing.T) {
	cli := &fakeCasesClient{locateErr: fmt.Errorf("not found")}
	op := &locateCaseOp{client: cli}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "r",
	}))
	if err == nil {
		t.Fatal("expected error when LocateCase fails")
	}
}

// ── createCase ────────────────────────────────────────────────────────────────

func TestCreateCase_NoToken(t *testing.T) {
	op := &createCaseOp{client: okCasesClient()}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestCreateCase_Success(t *testing.T) {
	op := &createCaseOp{client: okCasesClient()}
	out, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token":  "tok",
		"setVar": "new_case",
		"input":  map[string]any{"subject": "Test", "source": map[string]any{"name": "phone"}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["new_case"]; !ok {
		t.Error("expected SetVars[new_case] to be set")
	}
}

func TestCreateCase_DepError(t *testing.T) {
	cli := &fakeCasesClient{createErr: fmt.Errorf("validation failed")}
	op := &createCaseOp{client: cli}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "r",
	}))
	if err == nil {
		t.Fatal("expected error when CreateCase fails")
	}
}

// ── updateCase ────────────────────────────────────────────────────────────────

func TestUpdateCase_NoToken(t *testing.T) {
	op := &updateCaseOp{client: okCasesClient()}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{"setVar": "r"}))
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestUpdateCase_Success(t *testing.T) {
	op := &updateCaseOp{client: okCasesClient()}
	out, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token":       "tok",
		"setVar":      "updated",
		"x_json_mask": []any{"status"},
		"input":       map[string]any{"etag": "case-1", "status": map[string]any{"name": "Resolved"}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := out.SetVars["updated"]; !ok {
		t.Error("expected SetVars[updated] to be set")
	}
}

func TestUpdateCase_DepError(t *testing.T) {
	cli := &fakeCasesClient{updateErr: fmt.Errorf("etag mismatch")}
	op := &updateCaseOp{client: cli}
	_, err := op.Execute(context.Background(), casesInput(map[string]any{
		"token": "tok", "setVar": "r",
	}))
	if err == nil {
		t.Fatal("expected error when UpdateCase fails")
	}
}
