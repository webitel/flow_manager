package flow

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"math/rand"
	"net/http"
)

func (r *Router) Math(c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool
	var fnName, setVar, value string

	var argsElem []interface{}
	var _args interface{}
	var vm *otto.Otto

	if props, ok = args.(map[string]interface{}); !ok {
		return nil, model.NewAppError("Flow.Math", "flow.app.math.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
	}

	if setVar = model.StringValueFromMap("setVar", props, ""); setVar == "" {
		return nil, model.NewAppError("Flow.String", "flow.app.math.valid.setVar", nil, fmt.Sprintf("setVar is required %v", args), http.StatusBadRequest)
	}

	fnName = model.StringValueFromMap("fn", props, "random")

	if _args, ok = props["data"]; ok {
		//TODO
		//argsElem = model.ArgsToArrayInterface(c, _args)
	} else {
		argsElem = []interface{}{}
	}

	if fnName == "random" || fnName == "" {
		_args = random(argsElem)
	} else {
		vm = otto.New()
		vm.Set("fnName", fnName)
		vm.Set("args", argsElem)
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

	value = model.InterfaceToString(_args)
	return c.Set(model.Variables{
		setVar: value,
	})
}

func random(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return ""
	}

	n := rand.Int() % len(arr)
	return arr[n]
}
