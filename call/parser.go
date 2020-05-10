package call

import (
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

/*
\d := map[string]interface{}{
		"terminator": "ddsada",
		"files": []interface{}{
			map[string]interface{}{
				"name": 123,
				"id":   "${123}",
			},
		},
		"getDigits": map[string]interface{}{
			"setVar":    "getIvrDigit",
			"min":       "3",
			"max":       4,
			"tries":     1,
			"timeout":   2000,
			"flushDTMF": true,
		},
	}
*/

func (r *Router) Decode(conn model.Connection, in interface{}, out interface{}) *model.AppError {
	return flow.Decode(conn, in, out)
}
