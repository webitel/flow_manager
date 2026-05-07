package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// SqlDeps is the narrow interface required by the sql op.
// The implementation connects to the external DB, executes the query, and
// returns the first row as a map — all in one call so the op handles no
// connection objects and the return type stays primitive-only.
type SqlDeps interface {
	SqlQuery(ctx context.Context, driver, dns, query string, params []interface{}) (map[string]interface{}, error)
}

type sqlOp struct{ deps SqlDeps }

// SqlOp returns the native sql op: queries an external database and sets the
// first row's columns as schema variables.
func SqlOp(deps SqlDeps) ops.Op { return sqlOp{deps: deps} }

func (sqlOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o sqlOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	driver, _ := in.Node.Args["driver"].(string)
	dns, _ := in.Node.Args["dns"].(string)
	query, _ := in.Node.Args["query"].(string)

	driver = ops.ExpandStr(driver, in.Variables, in.GlobalVar)
	dns = ops.ExpandStr(dns, in.Variables, in.GlobalVar)
	query = ops.ExpandStr(query, in.Variables, in.GlobalVar)

	if query == "" {
		return ops.OpOutput{}, fmt.Errorf("sql: query is required")
	}
	if driver == "" {
		return ops.OpOutput{}, fmt.Errorf("sql: driver is required")
	}
	if dns == "" {
		return ops.OpOutput{}, fmt.Errorf("sql: dns is required")
	}

	timeoutMs := 1000
	switch v := in.Node.Args["timeout"].(type) {
	case float64:
		if v > 0 {
			timeoutMs = int(v)
		}
	}

	var params []interface{}
	if raw, ok := in.Node.Args["params"].([]interface{}); ok {
		params = make([]interface{}, len(raw))
		for i, p := range raw {
			if s, ok := p.(string); ok {
				params[i] = ops.ExpandStr(s, in.Variables, in.GlobalVar)
			} else {
				params[i] = p
			}
		}
	}

	qCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	row, err := o.deps.SqlQuery(qCtx, driver, dns, query, params)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("sql: %w", err)
	}

	vars := make(map[string]string, len(row))
	for k, v := range row {
		vars[k] = fmt.Sprint(v)
	}
	return ops.OpOutput{SetVars: vars}, nil
}
