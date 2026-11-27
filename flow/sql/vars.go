package sql

import (
	"regexp"
	"strings"
)

var (
	compileVarsGlobal = regexp.MustCompile(`\$\$\{([\s\S]*?)\}`)
	compileVars       = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
)

type globalVarLookup func(name string) string

func formatGlobalVarName(s string) string {
	return strings.TrimSpace(s[3 : len(s)-1])
}

func resolveGlobalVars(raw string, lookup globalVarLookup) (string, error) {
	if compileVars.MatchString(compileVarsGlobal.ReplaceAllString(raw, "")) {
		return "", ErrOnlyGlobalVars
	}

	return compileVarsGlobal.ReplaceAllStringFunc(raw, func(m string) string {
		return lookup(strings.TrimSpace(m[3 : len(m)-1])) // "$${" ... "}"
	}), nil
}
