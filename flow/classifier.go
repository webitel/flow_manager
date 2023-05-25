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
	MatchType string              `json:"matchType"`
}

type MatchType string

const (
	Full MatchType = "full"
	Part MatchType = "part"
)

func (r *router) classifierHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var (
		argv     ClassifierArgs
		variable string
	)
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	tokens := tok.Tokenize(strings.ToLower(argv.Input))

	for cluster, elems := range argv.Cluster {
		for _, word := range elems {
			if inArr(tokens, strings.ToLower(word), MatchType(argv.MatchType)) {
				variable = cluster
				break
			}
		}
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: variable,
	})
}

func inArr(tokens []string, val string, matchType MatchType) bool {
	var matchFunc func(str string, sub string) bool
	switch matchType {
	case Part:
		matchFunc = strings.Contains
	default:
		matchFunc = func(str string, sub string) bool {
			return str == sub
		}
	}

	for _, v := range tokens {
		if matchFunc(val, v) {
			return true
		}
	}
	return false
}
