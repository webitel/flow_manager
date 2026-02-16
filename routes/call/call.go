package call

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"regexp"
)

var compileOutPattern *regexp.Regexp

func init() {
	compileOutPattern = regexp.MustCompile(`\$(\d+)`)
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

func (call *callParser) ParseText(text string, ops ...model.ParseOption) string {
	txt := model.ParseText(call, text, ops...)

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
