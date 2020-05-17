package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (i *Flow) addFunction(args interface{}) *model.AppError {
	var name string
	var ok bool
	var tmp map[string]interface{}

	if tmp, ok = args.(map[string]interface{}); ok {
		if name, ok = tmp["name"].(string); !ok {
			return model.NewAppError("Iterator", "iterator.parse_app.function.valid_name", nil, "bad arguments, name is required", http.StatusBadRequest)
		}

		if actions, ok := tmp["actions"].([]interface{}); !ok {
			return model.NewAppError("Iterator", "iterator.parse_app.function.valid_actions", nil, "bad arguments, actions is required", http.StatusBadRequest)
		} else {
			i.Functions[name] = ArrInterfaceToArrayApplication(actions)
		}

		return nil
	} else {
		return model.NewAppError("Iterator", "iterator.parse_app.function.valid_args", nil, "bad arguments", http.StatusBadRequest)
	}
}

func (f *Flow) FunctionScope(name string) (*Flow, *model.AppError) {
	if apps, ok := f.Functions[name]; ok {
		return f.Fork(fmt.Sprintf("function-%s", name), apps), nil
	}

	return nil, model.NewAppError("Iterator", "iterator.function.new_scope", nil, "not found "+name, http.StatusBadRequest)
}
