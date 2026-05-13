package call

import (
	"fmt"
	"net/http"
	"regexp"

	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
)

var compileOutPattern *regexp.Regexp

func init() {
	compileOutPattern = regexp.MustCompile(`\$(\d+)`)
}

type callParser struct {
	outboundVars map[string]string
	timezoneName string
	calldomain.Call
}

func getOutboundReg(pattern, destination string) (map[string]string, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, apperrs.Newf(http.StatusBadRequest, "Call: call.router.valid.outbound_pattern: %s", err.Error())
	}

	out := make(map[string]string)
	for k, v := range r.FindStringSubmatch(destination) {
		out[fmt.Sprintf("%d", k)] = v
	}
	return out, nil
}

// OnInboundMessage satisfies sessionmgr.Connection. Call connections never
// receive inbound text messages; resume is driven by ESL events, not messages.
func (call *callParser) OnInboundMessage(_ func(string)) (unregister func()) {
	return func() {}
}

func (call *callParser) ParseText(text string, ops ...flow.ParseOption) string {
	txt := flow.ParseText(call, text, ops...)

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
