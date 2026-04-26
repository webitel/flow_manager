package builtin

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robertkrimen/otto"
)

var (
	reExprVar       = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
	reExprGlobalVar = regexp.MustCompile(`\$\$\{([\s\S]*?)\}`)
	reExprFuncs     = regexp.MustCompile(`&(year|yday|mon|mday|week|mweek|wday|hour|minute|minute_of_day|time_of_day)\(([\s\S]*?)\)`)
	reExprProtected = regexp.MustCompile(`\b(function|case|if|return|new|switch|var|this|typeof|for|while|break|do|continue)\b`)
	reSpace         = regexp.MustCompile(`\s`)
)

// parseExpression mirrors flow/if.go parseExpression: transforms ${var} into
// sys.getVariable("var") and date helpers into sys.year() etc. so the expression
// is safe to evaluate in an otto VM built by buildVM.
func parseExpression(expr string) string {
	// $${ } global variables first (longer prefix, must precede ${ } replace).
	expr = reExprGlobalVar.ReplaceAllStringFunc(expr, func(s string) string {
		m := reExprGlobalVar.FindStringSubmatch(s)
		return `sys.getGlobalVariable("` + m[1] + `")`
	})

	expr = reExprVar.ReplaceAllStringFunc(expr, func(s string) string {
		m := reExprVar.FindStringSubmatch(s)
		return `sys.getVariable("` + m[1] + `")`
	})

	expr = reExprFuncs.ReplaceAllStringFunc(expr, func(s string) string {
		m := reExprFuncs.FindStringSubmatch(s)
		return `sys.` + m[1] + `("` + m[2] + `")`
	})

	expr = reExprProtected.ReplaceAllLiteralString(expr, "")
	return expr
}

// buildVM creates an otto VM with a sys object that reads variables from vars.
// Global variables are not supported here (returns ""); callers that have a
// model.Connection can extend the sys object after calling buildVM.
func buildVM(vars map[string]string) *otto.Otto {
	vm := otto.New()
	sys, _ := vm.Object("sys = {}")

	sys.Set("getVariable", func(call otto.FunctionCall) otto.Value {
		key := call.Argument(0).String()
		v, _ := vm.ToValue(vars[key])
		return v
	})

	// Global variables are not available without a connection; return empty string.
	sys.Set("getGlobalVariable", func(call otto.FunctionCall) otto.Value {
		v, _ := vm.ToValue("")
		return v
	})

	now := time.Now()

	sys.Set("year", dateIntFunc(vm, now.Year(), now.Year()))
	sys.Set("yday", dateIntFunc(vm, now.YearDay(), 366))
	sys.Set("mon", dateIntFunc(vm, int(now.Month()), 12))
	sys.Set("mday", dateIntFunc(vm, now.Day(), 31))
	sys.Set("hour", dateIntFunc(vm, now.Hour(), 23))
	sys.Set("minute", dateIntFunc(vm, now.Minute(), 59))
	sys.Set("minute_of_day", dateIntFunc(vm, now.Hour()*60+now.Minute(), 1440))

	_, week := now.ISOWeek()
	sys.Set("week", dateIntFunc(vm, week, 53))

	begin := time.Date(now.Year(), now.Month(), 1, 1, 1, 1, 1, time.UTC)
	_, bw := begin.ISOWeek()
	sys.Set("mweek", dateIntFunc(vm, 1+week-bw, 6))

	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	sys.Set("wday", dateIntFunc(vm, wd, 7))

	sys.Set("time_of_day", func(call otto.FunctionCall) otto.Value {
		param := call.Argument(0).String()
		if param == "" {
			v, _ := vm.ToValue(leadingZero(now.Hour()) + ":" + leadingZero(now.Minute()))
			return v
		}
		cur := now.Hour()*10000 + now.Minute()*100 + now.Second()
		for _, seg := range strings.Split(param, ",") {
			parts := strings.SplitN(seg, "-", 2)
			if len(parts) != 2 {
				continue
			}
			if cur >= parseTimeHHMM(parts[0]) && cur <= parseTimeHHMM(parts[1]) {
				v, _ := vm.ToValue(true)
				return v
			}
		}
		v, _ := vm.ToValue(false)
		return v
	})

	return vm
}

// dateIntFunc returns an otto function: no arg → current value; with arg → range
// comparison (comma-separated integers or "min-max" ranges, same as flow/if.go).
func dateIntFunc(vm *otto.Otto, current, max int) func(otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		param := call.Argument(0).String()
		if param == "" {
			v, _ := vm.ToValue(current)
			return v
		}
		v, _ := vm.ToValue(parseDateRange(param, current, max))
		return v
	}
}

func parseDateRange(params string, current, maxVal int) bool {
	for _, seg := range strings.Split(reSpace.ReplaceAllString(params, ""), ",") {
		if strings.Contains(seg, "-") {
			parts := strings.SplitN(seg, "-", 2)
			lo, err1 := strconv.Atoi(parts[0])
			hi, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				continue
			}
			if lo > hi {
				lo, hi = hi, lo
			}
			if current >= lo && current <= hi {
				return true
			}
		} else {
			if i, err := strconv.Atoi(seg); err == nil && i == current {
				return true
			}
		}
	}
	return false
}

func parseTimeHHMM(s string) int {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	var result int
	for i, p := range parts {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		switch i {
		case 0:
			result += n * 10000
		case 1:
			result += n * 100
		case 2:
			result += n
		}
	}
	return result
}

func leadingZero(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
