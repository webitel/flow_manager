package flow

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/model"
)

func (r *router) dumpVarsHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {

	for k, v := range conn.Variables() {
		fmt.Printf("%s = %s\n", k, v)
	}

	return model.CallResponseOK, nil
}
