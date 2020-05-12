package flow

import (
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"math/rand"
	"net/http"
)

type MathArgs struct {
	Data   []interface{}
	SetVar string `json:"setVar"`
	Fn     string `json:"fn"`
}

func (r *router) Math(c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vm *otto.Otto
	var _args interface{}

	var argv = MathArgs{
		Fn: "random",
	}

	err := Decode(c, args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return nil, ErrorRequiredParameter("math", "setVar")
	}

	if argv.Fn == "random" || argv.Fn == "" {
		_args = random(argv.Data)
	} else {
		vm = otto.New()
		vm.Set("fnName", argv.Fn)
		vm.Set("args", argv.Data)
		v, err := vm.Run(`
				var value;
	
				if (typeof Math[fnName] === "function") {
					value = Math[fnName].apply(null, args);
				} else if (Math.hasOwnProperty(fnName)) {
					value = Math[fnName]
				} else {
					throw "Bad Math function " + fnName
				}
	
				if (isNaN(value)) {
					value = ""
				}
	
				value += "";
			`)

		if err != nil {
			return nil, model.NewAppError("Flow.String", "flow.app.string.error.args", nil, err.Error(), http.StatusBadRequest)
		}

		_args = v.String()
	}

	value := model.InterfaceToString(_args)
	return c.Set(model.Variables{
		argv.SetVar: value,
	})
}

func random(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return ""
	}

	n := rand.Int() % len(arr)
	return arr[n]
}
