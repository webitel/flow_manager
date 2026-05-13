// Package processing provides native ops for the processing (form) channel.
package processing

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/model"
	procpkg "github.com/webitel/flow_manager/pkg/processing"
)

// ProcessingConn is the full interface all native processing ops need.
// *providers/grpc.processingConnection satisfies it.
type ProcessingConn interface {
	Id() string
	DomainId() int64
	Get(key string) (string, bool)
	Set(ctx context.Context, vars model.Variables) (model.Response, error)
	GetComponentByName(name string) any
	SetComponent(name string, component any)
	Export(ctx context.Context, vars []string)
	DumpExportVariables() map[string]string
	SendForm(ctx context.Context, form procpkg.FormElem) error
	OnFormAction(handler func(procpkg.FormAction)) (unregister func())
}

// Dispatcher routes a resume event to a suspended flow.
// *coordinator.Coordinator satisfies this interface.
type Dispatcher interface {
	Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error
}

// DispatchFunc is a function adapter for Dispatcher.
type DispatchFunc func(ctx context.Context, resumeKey string, payload map[string]string) error

func (f DispatchFunc) Dispatch(ctx context.Context, key string, payload map[string]string) error {
	return f(ctx, key, payload)
}

// Register adds the native generateForm op to reg.
// coord is late-bound: it may be nil when Register is called and must be set
// before the first form action arrives (Bootstrap late-binding pattern).
func Register(reg *ops.Registry, coord Dispatcher) {
	reg.Register("generateForm", &generateFormOp{coord: coord})
}

// ── context helper ────────────────────────────────────────────────────────────

type connKey struct{}

// WithConn stores a ProcessingConn in ctx for retrieval by the op.
func WithConn(ctx context.Context, conn ProcessingConn) context.Context {
	return context.WithValue(ctx, connKey{}, conn)
}

// connFromContext retrieves the ProcessingConn stored by WithConn.
func connFromContext(ctx context.Context) (ProcessingConn, bool) {
	c, ok := ctx.Value(connKey{}).(ProcessingConn)
	return c, ok
}

// ── generateForm ──────────────────────────────────────────────────────────────

type generateFormOp struct{ coord Dispatcher }

func (o *generateFormOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

type generateFormArgs struct {
	Id      string                    `json:"id"`
	Title   string                    `json:"title"`
	Actions []*procpkg.FormActionElem `json:"actions"`
	Body    []string                  `json:"body"`
}

func (o *generateFormOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	// ── Resume path ──────────────────────────────────────────────────────────
	// The payload contains form fields set by the agent plus the action name.
	if in.ResumePayload != nil {
		return ops.OpOutput{SetVars: in.ResumePayload}, nil
	}

	// ── Fresh path ────────────────────────────────────────────────────────────
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("generateForm: no processing connection in context")
	}

	var argv generateFormArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	// Build form body from named components.
	body := make([]any, 0, len(argv.Body))
	for _, name := range argv.Body {
		if c := conn.GetComponentByName(name); c != nil {
			body = append(body, c)
		}
	}

	form := procpkg.FormElem{
		Id:      argv.Id,
		Title:   argv.Title,
		Actions: argv.Actions,
		Body:    body,
	}

	if err := conn.SendForm(ctx, form); err != nil {
		return ops.OpOutput{}, fmt.Errorf("generateForm: send form: %v", err)
	}

	suspendKey := "form:" + in.ConnID

	// Register a one-shot handler: fires when agent submits the form, then
	// dispatches the action payload to the coordinator which resumes the flow.
	coord := o.coord
	formID := argv.Id
	var unregFn func()
	unregFn = conn.OnFormAction(func(action procpkg.FormAction) {
		if unregFn != nil {
			unregFn()
			unregFn = nil
		}
		payload := make(map[string]string, len(action.Fields)+1)
		// Map the button / action name to the form's id variable (legacy behaviour).
		if formID != "" {
			payload[formID] = action.Name
		}
		for k, v := range action.Fields {
			payload[k] = fmt.Sprintf("%v", v)
		}
		if coord != nil {
			_ = coord.Dispatch(ctx, suspendKey, payload)
		}
	})

	return ops.OpOutput{
		SuspendKey:      suspendKey,
		ReenterOnResume: true,
		Pending: &state.PendingIntent{
			OpName:    "generateForm",
			NodeID:    in.Node.ID,
			ResumeKey: suspendKey,
		},
	}, nil
}
