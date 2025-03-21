package model

import (
	"regexp"
	"strings"
)

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
	jsonString := hasOption(ParseOptionJson, ops...)
	uri := false

	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			if strings.HasSuffix(r[1], ".uri()") {
				r[1] = r[1][:len(r[1])-6]
				uri = true
			}
			out, _ = c.Get(r[1])

			if uri && out != "" {
				out = UrlEncoded(out)
			}
		}

		if jsonString && len(out) > 0 {
			d := JsonString(nil, out, true)
			return string(d)
		}

		return
	})

	return text
}

func hasOption(o ParseOption, ops ...ParseOption) bool {
	for _, v := range ops {
		if o == v {
			return true
		}
	}

	return false
}
