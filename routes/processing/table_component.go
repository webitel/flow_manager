package processing

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/pkg/processing"
	"net/http"
)

func (r *Router) formTable(ctx context.Context, scope *flow.Flow, conn Connection, args any) (model.Response, *model.AppError) {
	var argv processing.FormTable

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter("tableComponent", "name")
	}

	outputs, err := parseOutputs(args)
	if err != nil {
		return nil, err
	}

	argv.OutputsFn = make(map[string]processing.FormTableActionFn, len(outputs))
	for k, v := range outputs {
		// todo group context
		argv.OutputsFn[k] = func(_ context.Context, sync bool, vars map[string]any) error {
			_, err := conn.Set(ctx, vars)
			if err != nil {
				return err
			}

			if sync {
				flow.Route(ctx, scope.Fork("component-"+k, v), r)
			} else {
				go flow.Route(ctx, scope.Fork("component-"+k, v), r)
			}

			return nil
		}
	}

	conn.SetComponent(argv.Id, argv)

	return model.CallResponseOK, nil
}

func parseOutputs(in any) (map[string]model.Applications, *model.AppError) {
	var apps []any
	props, ok := in.(map[string]any)
	if !ok {
		return nil, model.NewAppError("Processing.Parse", "processing.valid.props", nil, fmt.Sprintf("bad arguments %v", in), http.StatusBadRequest)
	}

	outputs := make(map[string]model.Applications)

	for k, v := range props["outputs"].(map[string]any) {
		if apps, ok = v.([]any); ok {
			outputs[k] = flow.ArrInterfaceToArrayApplication(apps)
		}
	}

	return outputs, nil
}
