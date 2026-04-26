package builtin

import (
	"regexp"
	"strings"
)

var reVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)

// expand replaces all ${varName} placeholders in s with the corresponding
// value from vars. Missing variables are replaced with an empty string.
func expand(s string, vars map[string]string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return reVar.ReplaceAllStringFunc(s, func(match string) string {
		sub := reVar.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		return vars[sub[1]]
	})
}
