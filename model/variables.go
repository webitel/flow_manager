package model

import "regexp"

var compileVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

func ParseJsonText(c Connection, text string) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		if len(out) > 0 {
			out = string(JsonString(nil, out, true))
		}

		return
	})

	return text
}
