package flow

import (
	"context"
	"strings"

	"github.com/euskadi31/go-tokenizer"
	"github.com/webitel/flow_manager/model"
)

var tok = tokenizer.New()

type ClassifierArgs struct {
	Cluster   map[string][]string `json:"cluster"`
	Input     string              `json:"input"`
	Set       string              `json:"set"`
	MatchType MatchType           `json:"matchType"`
}

type MatchType int64

const (
	Full MatchType = 0
	Part MatchType = 1
)

func (r *router) classifierHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv ClassifierArgs
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	tokens := tok.Tokenize(strings.ToLower(argv.Input))

	for cluster, elems := range argv.Cluster {
		for _, word := range elems {
			if inArr(tokens, strings.ToLower(word), argv.MatchType) {
				return conn.Set(ctx, model.Variables{
					argv.Set: cluster,
				})
			}
		}
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: nil,
	})
}

func inArr(tokens []string, val string, matchType MatchType) bool {
	var matchFunc func(str string, sub string) bool
	switch matchType {
	case Full:
		matchFunc = func(str string, sub string) bool {
			return str == sub
		}
	case Part:
		matchFunc = strings.Contains
	}

	for _, v := range tokens {
		if matchFunc(val, v) {
			return true
		}
	}
	return false
}
