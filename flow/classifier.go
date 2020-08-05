package flow

import (
	"context"
	"github.com/euskadi31/go-tokenizer"
	"github.com/webitel/flow_manager/model"
	"strings"
)

var tok = tokenizer.New()

type ClassifierArgs struct {
	Cluster map[string][]string `json:"cluster"`
	Input   string              `json:"input"`
	Set     string              `json:"set"`
}

func (r *router) classifierHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ClassifierArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	tokens := tok.Tokenize(strings.ToLower(argv.Input))

	for cluster, elems := range argv.Cluster {
		for _, word := range elems {
			if inArr(tokens, strings.ToLower(word)) {
				return conn.Set(ctx, model.Variables{
					argv.Set: cluster,
				})
			}
		}
	}

	return model.CallResponseOK, nil
}

func inArr(tokens []string, val string) bool {

	for _, v := range tokens {
		if v == val {
			return true
		}
	}

	return false
}
