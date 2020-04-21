package email

import (
	"github.com/webitel/flow_manager/model"
	"regexp"
)

var compileVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
}

type emailParser struct {
	timezoneName string
	model.EmailConnection
}

func (e *emailParser) ParseText(text string) string {
	txt := compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = e.Get(r[1])
		}

		return
	})

	return txt
}
