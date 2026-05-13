// Package cases provides native ops for the Webitel Cases service.
// Each op maps one-to-one to the legacy flow/cases.go handler: it decodes
// the same args schema, calls the Cases client, marshals the result to JSON
// and stores it in the named flow variable.
package cases

import (
	"context"
	"encoding/json"
	"fmt"

	cases2 "github.com/webitel/flow_manager/api/gen/cases"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// Register adds all cases ops to reg.
func Register(reg *ops.Registry, client domcases.Client) {
	reg.Register("getCases", &getCasesOp{client})
	reg.Register("locateCase", &locateCaseOp{client})
	reg.Register("createCase", &createCaseOp{client})
	reg.Register("updateCase", &updateCaseOp{client})
	reg.Register("linkCommunication", &linkCommunicationOp{client})
	reg.Register("getServiceCatalogs", &getServiceCatalogsOp{client})
	reg.Register("publishComment", &publishCommentOp{client})
	reg.Register("createLink", &createLinkOp{client})
	reg.Register("deleteLink", &deleteLinkOp{client})
	reg.Register("locateService", &locateServiceOp{client})
	reg.Register("createRelatedCase", &createRelatedCaseOp{client})
	reg.Register("listCaseFiles", &listCaseFilesOp{client})
}

// marshalToVar JSON-encodes res and returns SetVars{setVar: jsonStr}.
// An empty object/array/string is stored as an empty string, matching the
// legacy setResponse helper in flow/cases.go.
func marshalToVar(setVar string, res any) (ops.OpOutput, error) {
	b, err := json.Marshal(res)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("cases: marshal result: %w", err)
	}
	str := string(b)
	if str == "{}" || str == "[]" || str == `""` {
		str = ""
	}
	return ops.OpOutput{SetVars: map[string]string{setVar: str}}, nil
}

func requireTokenAndSetVar(opName, token, setVar string) error {
	if token == "" {
		return fmt.Errorf("%s: token is required", opName)
	}
	if setVar == "" {
		return fmt.Errorf("%s: setVar is required", opName)
	}
	return nil
}

// ── getCases ──────────────────────────────────────────────────────────────────

type getCasesOp struct{ client domcases.Client }

func (o *getCasesOp) Kind() ops.OpKind { return ops.OpKindSync }

type getCasesArgs struct {
	cases2.SearchCasesRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *getCasesOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getCasesArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("getCases", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.SearchCases(ctx, &argv.SearchCasesRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("getCases: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── locateCase ────────────────────────────────────────────────────────────────

type locateCaseOp struct{ client domcases.Client }

func (o *locateCaseOp) Kind() ops.OpKind { return ops.OpKindSync }

type locateCaseArgs struct {
	cases2.LocateCaseRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *locateCaseOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv locateCaseArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("locateCase", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.LocateCase(ctx, &argv.LocateCaseRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("locateCase: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── createCase ────────────────────────────────────────────────────────────────

type createCaseOp struct{ client domcases.Client }

func (o *createCaseOp) Kind() ops.OpKind { return ops.OpKindSync }

type createCaseArgs struct {
	cases2.CreateCaseRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *createCaseOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv createCaseArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("createCase", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.CreateCase(ctx, &argv.CreateCaseRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("createCase: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── updateCase ────────────────────────────────────────────────────────────────

type updateCaseOp struct{ client domcases.Client }

func (o *updateCaseOp) Kind() ops.OpKind { return ops.OpKindSync }

type updateCaseArgs struct {
	cases2.UpdateCaseRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *updateCaseOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv updateCaseArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("updateCase", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.UpdateCase(ctx, &argv.UpdateCaseRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("updateCase: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── linkCommunication ─────────────────────────────────────────────────────────

type linkCommunicationOp struct{ client domcases.Client }

func (o *linkCommunicationOp) Kind() ops.OpKind { return ops.OpKindSync }

type linkCommunicationArgs struct {
	cases2.LinkCommunicationRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *linkCommunicationOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv linkCommunicationArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("linkCommunication", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.LinkCommunication(ctx, &argv.LinkCommunicationRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("linkCommunication: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── getServiceCatalogs ────────────────────────────────────────────────────────

type getServiceCatalogsOp struct{ client domcases.Client }

func (o *getServiceCatalogsOp) Kind() ops.OpKind { return ops.OpKindSync }

type getServiceCatalogsArgs struct {
	cases2.ListCatalogRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *getServiceCatalogsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getServiceCatalogsArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("getServiceCatalogs", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.GetServiceCatalogs(ctx, &argv.ListCatalogRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("getServiceCatalogs: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── publishComment ────────────────────────────────────────────────────────────

type publishCommentOp struct{ client domcases.Client }

func (o *publishCommentOp) Kind() ops.OpKind { return ops.OpKindSync }

type publishCommentArgs struct {
	cases2.PublishCommentRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *publishCommentOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv publishCommentArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("publishComment", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.PublishComment(ctx, &argv.PublishCommentRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("publishComment: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── createLink ────────────────────────────────────────────────────────────────

type createLinkOp struct{ client domcases.Client }

func (o *createLinkOp) Kind() ops.OpKind { return ops.OpKindSync }

type createLinkArgs struct {
	cases2.CreateLinkRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *createLinkOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv createLinkArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("createLink", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.CreateLink(ctx, &argv.CreateLinkRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("createLink: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── deleteLink ────────────────────────────────────────────────────────────────

type deleteLinkOp struct{ client domcases.Client }

func (o *deleteLinkOp) Kind() ops.OpKind { return ops.OpKindSync }

type deleteLinkArgs struct {
	cases2.DeleteLinkRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *deleteLinkOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv deleteLinkArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("deleteLink", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.DeleteLink(ctx, &argv.DeleteLinkRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("deleteLink: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── locateService ─────────────────────────────────────────────────────────────

type locateServiceOp struct{ client domcases.Client }

func (o *locateServiceOp) Kind() ops.OpKind { return ops.OpKindSync }

type locateServiceArgs struct {
	cases2.LocateServiceRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *locateServiceOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv locateServiceArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("locateService", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.LocateService(ctx, &argv.LocateServiceRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("locateService: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── createRelatedCase ─────────────────────────────────────────────────────────

type createRelatedCaseOp struct{ client domcases.Client }

func (o *createRelatedCaseOp) Kind() ops.OpKind { return ops.OpKindSync }

type createRelatedCaseArgs struct {
	cases2.CreateRelatedCaseRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *createRelatedCaseOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv createRelatedCaseArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("createRelatedCase", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.CreateRelatedCase(ctx, &argv.CreateRelatedCaseRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("createRelatedCase: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}

// ── listCaseFiles ─────────────────────────────────────────────────────────────

type listCaseFilesOp struct{ client domcases.Client }

func (o *listCaseFilesOp) Kind() ops.OpKind { return ops.OpKindSync }

type listCaseFilesArgs struct {
	cases2.ListFilesRequest
	Token  string `json:"token"`
	SetVar string `json:"setVar"`
}

func (o *listCaseFilesOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv listCaseFilesArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := requireTokenAndSetVar("listCaseFiles", argv.Token, argv.SetVar); err != nil {
		return ops.OpOutput{}, err
	}
	res, err := o.client.ListCaseFiles(ctx, &argv.ListFilesRequest, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("listCaseFiles: %w", err)
	}
	return marshalToVar(argv.SetVar, res)
}
