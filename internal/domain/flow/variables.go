package flow

import (
	"regexp"
	"strings"

	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

// Variables is a generic key-value map used as flow scope variables.
type Variables map[string]interface{}

// ParseOption controls optional behaviour of ParseText.
type ParseOption uint

const (
	ParseOptionJson ParseOption = 1 << iota
)

var compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)

// ParseText replaces ${varName} placeholders in text using the connection's variables.
func ParseText(c Connection, text string, ops ...ParseOption) string {
	jsonString := hasOption(ParseOptionJson, ops...)
	uri := false

	return compileVar.ReplaceAllStringFunc(text, func(varName string) string {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) == 0 {
			return varName
		}
		key := r[1]
		if strings.HasSuffix(key, ".uri()") {
			key = key[:len(key)-6]
			uri = true
		}
		out, _ := c.Get(key)
		if uri && out != "" {
			out = utils.UrlEncoded(out)
		}
		if jsonString && len(out) > 0 {
			return string(utils.JsonString(nil, out, true))
		}
		return out
	})
}

func hasOption(o ParseOption, ops ...ParseOption) bool {
	for _, v := range ops {
		if o == v {
			return true
		}
	}
	return false
}
