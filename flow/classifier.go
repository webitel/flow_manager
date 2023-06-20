package flow

import (
	"context"
	"strings"

	"github.com/euskadi31/go-tokenizer"
	"github.com/webitel/flow_manager/model"
)

var tok = tokenizer.New()

type ClassifierArgs struct {
	Cluster      map[string][]string `json:"cluster"`
	Input        string              `json:"input"`
	Set          string              `json:"set"`
	PhraseSearch bool                `json:"phraseSearch"`
	MatchType    string              `json:"matchType"`
}

type MatchType string

const (
	Full MatchType = "full"
	Part MatchType = "part"
)

func (r *router) classifierHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var (
		argv ClassifierArgs
	)
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	return conn.Set(ctx, model.Variables{
		argv.Set: findInCluster(argv.Cluster, argv.Input, argv.MatchType, argv.PhraseSearch),
	})
}

// findInCluster finds user input in cluster. Variating by match type and phrase search. Match type determines what
// we can identify as match can be partial or full. Phrase search changes algorithm of search by comparing cluster values to
// the user input while regular search compares user input to the cluster values.
func findInCluster(clusters map[string][]string, userInput string, matchType string, phraseSearch bool) string {
	var (
		tokens []string
		found  bool
	)
	if !phraseSearch {
		tokens = tok.Tokenize(strings.ToLower(userInput))
	}
	for cluster, elems := range clusters {
		for _, phrase := range elems {
			if phraseSearch {
				found = inArr(userInput, MatchType(strings.ToLower(matchType)), strings.ToLower(phrase))
			} else {
				found = inArr(strings.ToLower(phrase), MatchType(strings.ToLower(matchType)), tokens...)
			}
			if found {
				return cluster
			}
		}
	}
	return ""
}

func inArr(val string, matchType MatchType, tokens ...string) bool {
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
	return matchFunc(strings.Join(tokens, " "), val)
}
