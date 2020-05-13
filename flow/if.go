package flow

import (
	"context"
	"errors"
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	compileProtectedFunctions = regexp.MustCompile(`\b(function|case|if|return|new|switch|var|this|typeof|for|while|break|do|continue)\b`)
	compileVars               = regexp.MustCompile(`\$\{([\s\S]*?)\}`)
	compileFunctions          = regexp.MustCompile(`\&(year|yday|mon|mday|week|mweek|wday|hour|minute|minute_of_day|time_of_day|date_time)\(([\s\S]*?)\)`)
	regSpace                  = regexp.MustCompile(`\s`)
)

type conditionArgs struct {
	expression string
	then_      *Node
	else_      *Node
	vm_        *otto.Otto
	flow       *Flow
}

func newConditionArgs(i *Flow, parent *Node, props interface{}) *conditionArgs {
	args := &conditionArgs{
		then_: NewNode(parent),
		else_: NewNode(parent),
		flow:  i,
	}

	if tmp, ok := props.(map[string]interface{}); ok {
		if th, ok := tmp["then"].([]interface{}); ok {
			parseFlowArray(i, args.then_, ArrInterfaceToArrayApplication(th))
		}

		if el, ok := tmp["else"].([]interface{}); ok {
			parseFlowArray(i, args.else_, ArrInterfaceToArrayApplication(el))
		}

		if ex, ok := tmp["expression"].(string); ok {
			args.expression = parseExpression(ex)
		}
	}

	return args
}

func (r *router) conditionHandler(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var req *conditionArgs
	var ok bool
	if req, ok = args.(*conditionArgs); !ok {
		return nil, model.NewAppError("Flow.ConditionHandler", "flow.condition_if.not_found", nil, "bad arguments", http.StatusBadRequest)
	}

	if req.vm_ == nil {
		req.vm_ = otto.New()
	}

	injectJsSysObject(conn, req.vm_, req.flow)

	if value, err := req.vm_.Run(`_result = ` + conn.ParseText(req.expression)); err == nil {
		if boolVal, err := value.ToBoolean(); err == nil && boolVal == true {
			wlog.Debug(fmt.Sprintf("condition (%s) is true", req.expression))
			req.flow.SetRoot(req.then_)
		} else {
			wlog.Debug(fmt.Sprintf("condition (%s) is false", req.expression))
			req.flow.SetRoot(req.else_)
		}
	} else {
		return nil, model.NewAppError("Flow.ConditionHandler", "flow.condition_if.vm_err", nil, err.Error(), http.StatusBadRequest)
	}

	return ResponseOK, nil
}

func parseExpression(expression string) string {

	expression = compileVars.ReplaceAllStringFunc(expression, func(varName string) string {
		l := compileVars.FindStringSubmatch(varName)
		return fmt.Sprintf(`sys.getVariable("%s")`, l[1])
	})

	expression = compileFunctions.ReplaceAllStringFunc(expression, func(s string) string {
		l := compileFunctions.FindStringSubmatch(s)

		return fmt.Sprintf(`sys.%s("%s")`, l[1], l[2])
	})

	expression = compileProtectedFunctions.ReplaceAllLiteralString(expression, "")

	return expression
}

func injectJsSysObject(conn model.Connection, vm *otto.Otto, flow *Flow) *otto.Object {
	sys, _ := vm.Object("sys = {}")
	sys.Set("getVariable", func(call otto.FunctionCall) otto.Value {
		val, _ := conn.Get(call.Argument(0).String())
		res, err := vm.ToValue(val)
		if err != nil {
			return otto.Value{}
		}
		return res
	})

	sys.Set("year", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()
		if param == "" {
			v, _ = otto.ToValue(flow.Now().Year())
		} else {
			v, _ = vm.ToValue(parseDate(param, flow.Now().Year(), 9999))
		}
		return v
	})

	sys.Set("yday", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()
		if param == "" {
			v, _ = otto.ToValue(flow.Now().YearDay())
		} else {
			v, _ = vm.ToValue(parseDate(param, flow.Now().YearDay(), 366))
		}
		return v
	})

	sys.Set("mon", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(flow.Now().Month())
		} else {
			v, _ = vm.ToValue(parseDate(param, int(flow.Now().Month()), 12))
		}
		return v
	})

	sys.Set("mday", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(flow.Now().Day())
		} else {
			v, _ = vm.ToValue(parseDate(param, int(flow.Now().Day()), 31))
		}
		return v
	})

	sys.Set("week", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()
		_, week := flow.Now().ISOWeek()

		if param == "" {
			v, _ = otto.ToValue(week)
		} else {
			v, _ = vm.ToValue(parseDate(param, week, 53))
		}
		return v
	})

	sys.Set("mweek", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(numberOfTheWeekInMonth(flow.Now()))
		} else {
			v, _ = vm.ToValue(parseDate(param, numberOfTheWeekInMonth(flow.Now()), 6))
		}
		return v
	})

	sys.Set("wday", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(flow.Now().Weekday() + 1)
		} else {
			v, _ = vm.ToValue(parseDate(param, getWeekday(flow.Now()), 7))
		}
		return v
	})

	sys.Set("hour", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(flow.Now().Hour())
		} else {
			v, _ = vm.ToValue(parseDate(param, int(flow.Now().Hour()), 23))
		}

		return v
	})

	sys.Set("minute", func(call otto.FunctionCall) otto.Value {
		var v otto.Value
		param := call.Argument(0).String()

		if param == "" {
			v, _ = otto.ToValue(flow.Now().Minute())
		} else {
			v, _ = vm.ToValue(parseDate(param, int(flow.Now().Minute()), 59))
		}

		return v
	})

	sys.Set("minute_of_day", func(call otto.FunctionCall) otto.Value {
		var v otto.Value

		param := call.Argument(0).String()
		date := flow.Now()
		minOfDay := date.Hour()*60 + date.Minute()

		if param == "" {
			v, _ = otto.ToValue(minOfDay)
		} else {
			v, _ = vm.ToValue(parseDate(param, minOfDay, 1440))
		}

		return v
	})

	sys.Set("time_of_day", func(call otto.FunctionCall) (result otto.Value) {
		var tmp []string

		date := flow.Now()

		if call.Argument(0).String() == "" {
			result, _ = otto.ToValue(leadingZeros(date.Hour()) + ":" + leadingZeros(date.Minute()))
			return
		}

		current := (date.Hour() * 10000) + (int(date.Minute()) * 100) + date.Second()
		times := strings.Split(call.Argument(0).String(), ",")

		for _, v := range times {
			tmp = strings.Split(v, "-")
			if len(tmp) != 2 {
				wlog.Warn(fmt.Sprintf("skip parse: %v", v))
				continue
			}
			if current >= parseTime(tmp[0]) && current <= parseTime(tmp[1]) {
				result, _ = vm.ToValue(true)
				return
			}
		}
		result, _ = vm.ToValue(false)
		return
	})

	sys.Set("date_time", func(call otto.FunctionCall) (result otto.Value) {
		var tmp []string
		var err error
		var t1, t2 int64

		date := flow.Now()

		if call.Argument(0).String() == "" {
			result, _ = otto.ToValue(date.Format("2006-01-02 15:04:05"))
			fmt.Println(result)
			return
		}

		currentNano := date.UnixNano()
		times := strings.Split(call.Argument(0).String(), ",")

		for _, v := range times {
			tmp = strings.Split(v, "~")
			if len(tmp) != 2 {
				wlog.Warn(fmt.Sprintf("skip parse: %v", v))
				continue
			}
			strings.Trim(tmp[0], tmp[0])
			strings.Trim(tmp[1], tmp[1])

			t1, err = stringDateTimeToNano(tmp[0], flow.timezone)
			if err != nil {
				wlog.Error(fmt.Sprintf("call %s parse date: %s", conn.Id(), err.Error()))
				continue
			}

			t2, err = stringDateTimeToNano(tmp[1], flow.timezone)
			if err != nil {
				wlog.Error(fmt.Sprintf("call %s parse date: %s", conn.Id(), err.Error()))
				continue
			}

			if currentNano >= t1 && currentNano <= t2 {
				result, _ = vm.ToValue(true)
				return
			}

		}
		return
	})

	return sys
}

func parseTime(str string) (result int) {
	var err error
	var tmp int

	for i, v := range strings.Split(str, ":") {
		tmp, err = strconv.Atoi(strings.Trim(v, ` `))
		if err != nil {
			wlog.Error(fmt.Sprintf("bad parse time: %s", err.Error()))
			return
		}
		if i == 0 {
			result += (tmp * 10000)
		} else if i == 1 {
			result += (tmp * 100)
		} else {
			result += tmp
		}
	}
	return
}

func numberOfTheWeekInMonth(now time.Time) int {
	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 1, 1, 1, 1, time.UTC)
	_, thisWeek := now.ISOWeek()
	_, beginningWeek := beginningOfTheMonth.ISOWeek()
	return 1 + thisWeek - beginningWeek
}

func parseDate(params string, datetime int, maxVal int) (result bool) {
	rows := strings.Split(regSpace.ReplaceAllString(params, ""), ",")

	if len(rows) == 0 {
		wlog.Warn("bad parameters: " + params)
		return
	}

	for _, v := range rows {
		if strings.Index(v, "-") != -1 {
			result = equalsDateTimeRange(datetime, strings.Trim(v, ` `), maxVal)
		} else {
			if i, err := strconv.Atoi(strings.Trim(v, ` `)); err == nil {
				result = i == datetime
			}
		}

		if result {
			return
		}
	}

	return
}

func equalsDateTimeRange(datetime int, strRange string, maxVal int) (result bool) {
	var min, max int
	var err error

	rows := strings.Split(strRange, "-")
	min, err = strconv.Atoi(rows[0])
	if err != nil {
		wlog.Error(fmt.Sprintf("bad parse date: %s", err.Error()))
		return
	}

	if len(rows) >= 2 {
		max, err = strconv.Atoi(rows[1])
		if err != nil {
			wlog.Error(fmt.Sprintf("bad parse date:  %s", err.Error()))
			return
		}
	} else {
		max = maxVal
	}

	if min > max {
		tmp := min
		min = max
		max = tmp
	}

	result = datetime >= min && datetime <= max
	return
}

var weakdays = []int{1, 2, 3, 4, 5, 6, 7}

//todo move helper (calendar use)
func getWeekday(in time.Time) int {
	return weakdays[in.Weekday()]
}

func stringDateTimeToNano(data string, loc *time.Location) (int64, error) {
	var t time.Time
	var err error
	var length = len(data)

	if length == 19 {
		if loc != nil {
			t, err = time.ParseInLocation("2006-01-02 15:04:05", data, loc)
		} else {
			t, err = time.Parse("2006-01-02 15:04:05", data)
		}
	} else if length == 16 {
		if loc != nil {
			t, err = time.ParseInLocation("2006-01-02 15:04", data, loc)
		} else {
			t, err = time.Parse("2006-01-02 15:04", data)
		}
	} else {
		return 0, errors.New("Bad parse string:" + data)
	}

	if err != nil {
		return 0, err
	}

	return t.UnixNano(), nil
}

func leadingZeros(data int) string {
	if data < 10 {
		return "0" + strconv.Itoa(data)
	} else {
		return strconv.Itoa(data)
	}
}
