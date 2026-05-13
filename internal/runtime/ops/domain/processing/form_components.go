package processing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/webitel/flow_manager/api/gen/cases"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	procpkg "github.com/webitel/flow_manager/pkg/processing"
)

// ComponentDeps is the cases client interface that component ops need.
type ComponentDeps = domcases.Client

// RegisterComponents adds export, formFile, formComponent, formSelectCaseStatus to reg.
func RegisterComponents(reg *ops.Registry, deps ComponentDeps) {
	reg.Register("export", &exportOp{})
	reg.Register("formFile", &formFileOp{})
	reg.Register("formComponent", &formComponentOp{deps: deps})
	reg.Register("formSelectCaseStatus", &formSelectCaseStatusOp{deps: deps})
}

// ── export ────────────────────────────────────────────────────────────────────

type exportOp struct{}

func (exportOp) Kind() ops.OpKind { return ops.OpKindSync }

func (exportOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("export: no processing connection in context")
	}
	conn.Export(ctx, rawStringSlice(in))
	return ops.OpOutput{}, nil
}

// rawStringSlice extracts []string from RawArgs (JSON array or single string).
func rawStringSlice(in ops.OpInput) []string {
	switch v := in.Node.RawArgs.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, ops.ExpandStr(s, in.Variables, in.GlobalVar))
			}
		}
		return out
	case string:
		if s := ops.ExpandStr(v, in.Variables, in.GlobalVar); s != "" {
			return []string{s}
		}
	}
	return nil
}

// ── formFile ──────────────────────────────────────────────────────────────────

type formFileOp struct{}

func (formFileOp) Kind() ops.OpKind { return ops.OpKindSync }

func (formFileOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("formFile: no processing connection in context")
	}
	var argv struct {
		Id    string `json:"id"`
		Value any    `json:"value,omitempty"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id == "" {
		return ops.OpOutput{}, fmt.Errorf("formFile: id is required")
	}
	val, _ := conn.Get(argv.Id)
	if val == "" {
		argv.Value = make([]any, 0)
	} else {
		argv.Value = setToJSON(val)
	}
	conn.SetComponent(argv.Id, argv)
	return ops.OpOutput{}, nil
}

// ── formComponent ─────────────────────────────────────────────────────────────

type formComponentOp struct{ deps ComponentDeps }

func (o *formComponentOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *formComponentOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("formComponent: no processing connection in context")
	}
	var argv procpkg.FormComponent
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id == "" {
		return ops.OpOutput{}, fmt.Errorf("formComponent: id is required")
	}

	val, _ := conn.Get(argv.Id)

	switch argv.View.Component {
	case "wt-input":
		argv.Value = val
	case "form-select-case-status":
		opts, err := casesStatusOptions(ctx, o.deps, argv.View.Token, argv.View.ServiceId)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("formComponent: %w", err)
		}
		argv.View.Options = opts
		argv.Value = setToJSON(val)
	default:
		argv.Value = setToJSON(val)
	}

	conn.SetComponent(argv.Id, argv)
	return ops.OpOutput{}, nil
}

// ── formSelectCaseStatus (deprecated) ────────────────────────────────────────

type formSelectCaseStatusOp struct{ deps ComponentDeps }

func (o *formSelectCaseStatusOp) Kind() ops.OpKind { return ops.OpKindSync }

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

type fscStatusComponent struct {
	Id    string         `json:"id"`
	View  *fscStatusView `json:"view"`
	Value any            `json:"value"`
}

type fscStatusView struct {
	Component    string            `json:"component"`
	Label        string            `json:"label,omitempty"`
	Hint         string            `json:"hint,omitempty"`
	InitialValue string            `json:"initialValue,omitempty"`
	Options      []fscStatusOption `json:"options"`
}

type fscStatusOption struct {
	Id      int64  `json:"id"`
	Name    string `json:"name"`
	Initial bool   `json:"initial"`
	Final   bool   `json:"final"`
}

func (o *formSelectCaseStatusOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: no processing connection in context")
	}
	var argv formSelectCaseStatusArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id == "" {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: id is required")
	}
	if argv.ServiceId == 0 {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: serviceId is required")
	}
	if argv.Token == "" {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: token is required")
	}

	svcRes, err := o.deps.LocateService(ctx, &cases.LocateServiceRequest{
		Id:     argv.ServiceId,
		Fields: []string{"id", "catalog_id"},
	}, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: locate service: %w", err)
	}
	svc := svcRes.GetService()
	if svc == nil {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: service not found")
	}
	catalogId := svc.GetCatalogId()
	if catalogId == 0 {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: service has no catalog")
	}

	catRes, err := o.deps.LocateCatalog(ctx, &cases.LocateCatalogRequest{Id: catalogId}, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: locate catalog: %w", err)
	}
	cat := catRes.GetCatalog()
	if cat == nil || cat.GetStatus() == nil || cat.GetStatus().GetId() == 0 {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: catalog has no linked status dictionary")
	}

	condRes, err := o.deps.ListStatusConditions(ctx, &cases.ListStatusConditionRequest{
		StatusId: cat.GetStatus().GetId(),
		Size:     100,
	}, argv.Token)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("formSelectCaseStatus: list conditions: %w", err)
	}

	options := make([]fscStatusOption, 0, len(condRes.GetItems()))
	for _, item := range condRes.GetItems() {
		options = append(options, fscStatusOption{
			Id:      item.GetId(),
			Name:    item.GetName(),
			Initial: item.GetInitial(),
			Final:   item.GetFinal(),
		})
	}

	val, _ := conn.Get(argv.Id)
	conn.SetComponent(argv.Id, fscStatusComponent{
		Id: argv.Id,
		View: &fscStatusView{
			Component:    "form-select-case-status",
			Label:        argv.View.Label,
			Hint:         argv.View.Hint,
			InitialValue: argv.View.InitialValue,
			Options:      options,
		},
		Value: setToJSON(val),
	})
	return ops.OpOutput{}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// casesStatusOptions fetches select options from the cases service.
func casesStatusOptions(ctx context.Context, deps ComponentDeps, token string, serviceId int64) ([]procpkg.SelectOption, error) {
	svcRes, err := deps.LocateService(ctx, &cases.LocateServiceRequest{
		Id:     serviceId,
		Fields: []string{"id", "catalog_id"},
	}, token)
	if err != nil {
		return nil, fmt.Errorf("locate service: %w", err)
	}
	svc := svcRes.GetService()
	if svc == nil {
		return nil, fmt.Errorf("service not found")
	}
	catalogId := svc.GetCatalogId()
	if catalogId == 0 {
		return nil, fmt.Errorf("service has no catalog")
	}

	catRes, err := deps.LocateCatalog(ctx, &cases.LocateCatalogRequest{Id: catalogId}, token)
	if err != nil {
		return nil, fmt.Errorf("locate catalog: %w", err)
	}
	cat := catRes.GetCatalog()
	if cat == nil || cat.GetStatus() == nil || cat.GetStatus().GetId() == 0 {
		return nil, fmt.Errorf("catalog has no linked status dictionary")
	}

	condRes, err := deps.ListStatusConditions(ctx, &cases.ListStatusConditionRequest{
		StatusId: cat.GetStatus().GetId(),
		Size:     100,
	}, token)
	if err != nil {
		return nil, fmt.Errorf("list conditions: %w", err)
	}

	opts := make([]procpkg.SelectOption, 0, len(condRes.GetItems()))
	for _, item := range condRes.GetItems() {
		opts = append(opts, procpkg.SelectOption{
			Id:      int(item.GetId()),
			Name:    item.GetName(),
			Initial: item.GetInitial(),
			Final:   item.GetFinal(),
		})
	}
	return opts, nil
}

// setToJSON coerces a raw string into a JSON-parsed value when possible.
func setToJSON(src string) any {
	l := len(src)
	if l < 2 {
		return src
	}
	switch {
	case src[0] == '{' && src[l-1] == '}':
		var res map[string]any
		if json.Unmarshal([]byte(src), &res) == nil {
			return res
		}
	case src[0] == '[' && src[l-1] == ']':
		var res []any
		if json.Unmarshal([]byte(src), &res) == nil {
			return res
		}
	}
	return src
}
