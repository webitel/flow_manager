package call

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/webitel/flow_manager/model"
)

var compileVar *regexp.Regexp
var compileOutPattern *regexp.Regexp
var compileObjVar *regexp.Regexp

func init() {
	compileVar = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
	compileOutPattern = regexp.MustCompile(`\$(\d+)`)
	compileObjVar = regexp.MustCompile(`\{\{([\s\S]*?)\}\}`)
}

type callParser struct {
	outboundVars map[string]string
	timezoneName string
	model.Call
}

func getOutboundReg(pattern, destination string) (map[string]string, *model.AppError) {
	r, err := regexp.Compile(pattern)
	if err != nil {

		return nil, model.NewAppError("Call", "call.router.valid.outbound_pattern", nil, err.Error(), http.StatusBadRequest)
	}

	out := make(map[string]string)
	for k, v := range r.FindStringSubmatch(destination) {
		out[fmt.Sprintf("%d", k)] = v
	}
	return out, nil
}

func (call *callParser) ParseText(text string) string {
	txt := compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = call.Get(r[1])
		}

		return
	})

	txt = compileObjVar.ReplaceAllStringFunc(txt, func(varName string) (out string) {
		r := compileObjVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = call.Get(r[1])
		}

		return
	})

	if call.outboundVars != nil {
		txt = compileOutPattern.ReplaceAllStringFunc(txt, func(s string) string {
			r := compileOutPattern.FindStringSubmatch(s)
			if len(r) == 2 {
				if v, ok := call.outboundVars[r[1]]; ok {
					return v
				}
			}

			return ""
		})
	}

	return call.Call.ParseText(txt)
}
