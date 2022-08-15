package flow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/model"
)

var errTimeout = errors.New("timeout")

type JsArgs struct {
	Data   string
	SetVar string
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

	go func() {
		time.Sleep(2 * time.Second) // Stop after two seconds
		vm.Interrupt <- func() {
			panic(errTimeout)
		}
	}()

	result, err := vm.Run(`
		var LocalDate = function() {
			var t = _LocalDateParameters();
			return new Date(t[0], t[1] - 1, t[2], t[3], t[4], t[5])
		};
		(function(LocalDate) {` + argv.Data + `})(LocalDate)`)
	if err != nil {
		return nil, model.NewAppError("Flow.Js", "flow.js.runtime_err", nil, err.Error(), http.StatusBadRequest)
	}

	return conn.Set(ctx, model.Variables{
		argv.SetVar: result,
	})
}

func (r *router) panic(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv string
	if err := scope.DecodeSrc(args, &argv); err != nil {
		panic(err.Error())
	}

	panic(argv)
}
