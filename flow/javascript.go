package flow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/wlog"

	"github.com/robertkrimen/otto"

	"github.com/webitel/flow_manager/model"
)

var errTimeout = errors.New("timeout")

type JsArgs struct {
	Data   string
	SetVar string
}

type jsResult struct {
	val otto.Value
	err error
}

func (r *router) Js(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv JsArgs
	if err := scope.DecodeSrc(args, &argv); err != nil {
		return nil, err
	}

	argv.Data = compileVars.ReplaceAllStringFunc(argv.Data, func(varName string) string {
		l := compileVars.FindStringSubmatch(varName)
		return fmt.Sprintf(`_getChannelVar("%s")`, l[1])
	})

	vm := scope.GetVm()

	runtime := make(chan jsResult, 1)
	var result jsResult

	go func() {
		defer func() {
			if caught := recover(); caught != nil {
				wlog.Error(errTimeout.Error())
			}
		}()
		result := jsResult{}
		result.val, result.err = vm.Run(`
		var LocalDate = function() {
			var t = _LocalDateParameters();
			return new Date(t[0], t[1] - 1, t[2], t[3], t[4], t[5])
		};
		(function(LocalDate) {` + argv.Data + `})(LocalDate)`)
		runtime <- result
	}()

	select {
	case <-time.After(1 * time.Second):
		vm.Interrupt <- func() {
			panic(errTimeout)
		}
		return nil, model.NewAppError("Flow.Js", "flow.js.runtime_err", nil, errTimeout.Error(), http.StatusBadRequest)
	case result = <-runtime:

	}

	if result.err != nil {
		return nil, model.NewAppError("Flow.Js", "flow.js.runtime_err", nil, result.err.Error()+" js: "+argv.Data, http.StatusBadRequest)
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: result.val,
	})
}

func (r *router) panic(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv string
	if err := scope.DecodeSrc(args, &argv); err != nil {
		panic(err.Error())
	}

	panic(argv)
}
