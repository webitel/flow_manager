package call

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) Request(ctx context.Context, scope *flow.Flow, req model.ApplicationRequest) <-chan model.Result {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(ctx, scope, h.ArgsParser(scope.Connection, req.Args()))
		}
		return h.Handler(ctx, scope, req.Args())
	}
	return flow.Do(func(result *model.Result) {
		result.Err = model.NewAppError("Call.Request", "call.request.not_found", nil,
			fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
	})
}
