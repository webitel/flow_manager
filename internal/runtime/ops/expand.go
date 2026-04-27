package ops

import (
	"regexp"
	"strings"
)

var (
	reExpandVar    = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
	reExpandGlobal = regexp.MustCompile(`\$\$\{([\s\S]*?)\}`)
)

// ExpandStr expands variable placeholders in s:
//
//	${name}   → vars[name]
//	$${name}  → globalVar(name)  (skipped when globalVar is nil)
//
// Missing local variables are replaced with "".
func ExpandStr(s string, vars map[string]string, globalVar func(string) string) string {
	if !strings.ContainsAny(s, "$") {
		return s
	}
	if globalVar != nil && strings.Contains(s, "$${") {
		s = reExpandGlobal.ReplaceAllStringFunc(s, func(m string) string {
			sub := reExpandGlobal.FindStringSubmatch(m)
			return globalVar(sub[1])
		})
	}
	if strings.Contains(s, "${") {
		s = reExpandVar.ReplaceAllStringFunc(s, func(m string) string {
			sub := reExpandVar.FindStringSubmatch(m)
			return vars[sub[1]]
		})
	}
	return s
}
