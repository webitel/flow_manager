package flow

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/flow/sql"
	"github.com/webitel/flow_manager/model"
)

func (r *router) SqlHandler(ctx context.Context, scope *Flow, conn model.Connection, args any) (model.Response, *model.AppError) {
	req := sql.DefaultSqlArgs()

	if err := scope.DecodeSrc(args, req); err != nil {
		return nil, err
	}

	varsLookup := func(name string) string { return r.GlobalVariable(conn.DomainId(), name) }

	if err := req.Prepare(varsLookup); err != nil {
		return nil, model.NewAppError("Flow.SQL", "flow.sql.sql_handler.prepare", nil, err.Error(), http.StatusBadRequest)
	}

	if err := scope.Decode(req.Params, &req.Params); err != nil {
		return nil, err
	}

	req.Normalize()

	if err := req.Validate(); err != nil {
		return nil, model.NewAppError("Flow.SQL", "flow.sql.sql_handler.validate", nil, err.Error(), http.StatusBadRequest)
	}

	db, err := r.fm.GetSqlDb(ctx, req.Driver, req.DSN)
	if err != nil {
		return model.CallResponseError, err
	}

	c, cancel := context.WithTimeout(ctx, req.QueryTimeout())
	defer cancel()

	result, err := db.Query(c, req.Query, req.Params)
	if err != nil {
		return model.CallResponseError, err
	}

	return conn.Set(ctx, result)
}
