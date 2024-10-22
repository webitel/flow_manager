package model

import "regexp"

var compileVar *regexp.Regexp

type Variables map[string]interface{}

type ParseOption uint

const (
	ParseOptionJson ParseOption = 1 << iota
)

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

func ParseText(c Connection, text string, ops ...ParseOption) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		if len(out) > 0 {
			d := JsonString(nil, out, true)
			return string(d)
		}

		return
	})

	return text
}
