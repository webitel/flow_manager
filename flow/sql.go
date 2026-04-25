package flow

import (
	"context"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/model"
)

type SqlArgs struct {
	Driver  string        `json:"driver"`
	Dns     string        `json:"dns"`
	Query   string        `json:"query"`
	Params  []interface{} `json:"params"`
	Timeout int           `json:"timeout"`
}

func (r *router) SqlHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	req := &SqlArgs{
		Timeout: 1000,
	}

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

	db, err := r.fm.GetSqlDb(req.Driver, req.Dns)
	if err != nil {
		return model.CallResponseError, model.NewAppError("SqlHandler", "flow.sql.connect", nil, err.Error(), http.StatusInternalServerError)
	}

	c, _ := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)

	result, qErr := db.Query(c, req.Query, req.Params)
	if qErr != nil {
		return model.CallResponseError, model.NewAppError("SqlHandler", "flow.sql.query", nil, qErr.Error(), http.StatusInternalServerError)
	}

	return conn.Set(ctx, result)
}
