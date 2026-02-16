package call

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"strings"
)

type MarkIVRArgs struct {
	Name  string
	Value string
}

func (r *Router) MarkIVR(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv MarkIVRArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Name == "" || argv.Value == "" {
		return nil, ErrorRequiredParameter("MarkIVR", "name or value")
	}

	return call.Push(ctx, fmt.Sprintf("usr_%s", strings.Replace(argv.Name, "'", "", -1)), argv.Value)
}
