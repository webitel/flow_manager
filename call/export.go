package call

import (
	"github.com/webitel/flow_manager/model"
)

type ExportArg []string

func (r *Router) export(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv ExportArg

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}
	return call.Export(argv)
}
