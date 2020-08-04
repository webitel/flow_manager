package flow

import (
	"bytes"
	"context"
	"github.com/webitel/flow_manager/model"
)

type ClassifierArgs struct {
	Cluster map[string][][]byte `json:"cluster"`
	Input   string              `json:"input"`
	Set     string              `json:"set"`
}

func (r *router) classifierHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ClassifierArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	input := bytes.ToLower([]byte(argv.Input))
	for cluster, elems := range argv.Cluster {
		for _, word := range elems {
			if bytes.Index(input, bytes.ToLower(word)) > -1 {
				return conn.Set(ctx, model.Variables{
					argv.Set: cluster,
				})
			}
		}
	}

	return model.CallResponseOK, nil
}
