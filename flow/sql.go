package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type SqlArgs struct {
	Driver string        `json:"driver"`
	Dns    string        `json:"dns"`
	Query  string        `json:"query"`
	Params []interface{} `json:"params"`
}

func (r *router) SqlHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *SqlArgs

	if err := scope.Decode(args, &req); err != nil {
		return nil, err
	}

	if req.Query == "" {
		return nil, ErrorRequiredParameter("sql", "query")
	}
	if req.Driver == "" {
		return nil, ErrorRequiredParameter("sql", "driver")
	}
	if req.Dns == "" {
		return nil, ErrorRequiredParameter("sql", "dns")
	}

	db, err := r.fm.ExternalStore.Connect(req.Driver, req.Dns)
	if err != nil {
		return model.CallResponseError, err
	}

	result, err := db.Query(req.Query, req.Params)
	if err != nil {
		return model.CallResponseError, err
	}

	return conn.Set(ctx, result)
}
