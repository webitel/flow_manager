package builtin

import (
	"context"
	"strings"

	"github.com/euskadi31/go-tokenizer"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

var classifierTok = tokenizer.New()

type classifierMatchType string

const (
	classifierMatchFull classifierMatchType = "full"
	classifierMatchPart classifierMatchType = "part"
)

type classifierArgs struct {
	Cluster      map[string][]string `json:"cluster"`
	Input        string              `json:"input"`
	Set          string              `json:"set"`
	PhraseSearch bool                `json:"phraseSearch"`
	MatchType    string              `json:"matchType"`
}

type classifierOp struct{}

func Classifier() ops.Op { return &classifierOp{} }

func (o *classifierOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *classifierOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv classifierArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	result := classifierFindInCluster(argv.Cluster, argv.Input, argv.MatchType, argv.PhraseSearch)
	return ops.OpOutput{
		SetVars: map[string]string{argv.Set: result},
	}, nil
}

func classifierFindInCluster(clusters map[string][]string, userInput string, matchType string, phraseSearch bool) string {
	var (
		tokens []string
		found  bool
	)
	if !phraseSearch {
		tokens = classifierTok.Tokenize(strings.ToLower(userInput))
	}
	for cluster, elems := range clusters {
		for _, phrase := range elems {
			if phraseSearch {
				found = classifierInArr(strings.ToLower(userInput), classifierMatchType(strings.ToLower(matchType)), strings.ToLower(phrase))
			} else {
				found = classifierInArr(strings.ToLower(phrase), classifierMatchType(strings.ToLower(matchType)), tokens...)
			}
			if found {
				return cluster
			}
		}
	}
	return ""
}

func classifierInArr(val string, matchType classifierMatchType, tokens ...string) bool {
	var matchFunc func(str string, sub string) bool
	switch matchType {
	case classifierMatchPart:
		matchFunc = strings.Contains
	default:
		matchFunc = func(str string, sub string) bool { return str == sub }
	}
	for _, v := range tokens {
		if matchFunc(v, val) {
			return true
		}
	}
	return matchFunc(strings.Join(tokens, " "), val)
}
