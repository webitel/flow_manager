package flow

import (
	"context"
	"time"

	"github.com/webitel/flow_manager/model"
)

type SqlArgs struct {
	Driver   string        `json:"driver"`
	Dns      string        `json:"dns"`
	Query    string        `json:"query"`
	Params   []interface{} `json:"params"`
	Timeout  int           `json:"timeout"`
	Writable bool          `json:"writable"`
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

	db, err := r.fm.GetSqlDb(ctx, req.Driver, req.Dns, req.Writable)
	if err != nil {
		return model.CallResponseError, err
	}

	c, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
	defer cancel()

	result, err := db.Query(c, req.Query, req.Params)
	if err != nil {
		return model.CallResponseError, err
	}

	return conn.Set(ctx, result)
}
