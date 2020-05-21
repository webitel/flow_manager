package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

const TriggerDisconnected = "disconnected"

func (i *Flow) addTrigger(args interface{}) *model.AppError {
	var ok bool
	var tmp map[string]interface{}

	if tmp, ok = args.(map[string]interface{}); ok {
		for name, val := range tmp {
			if _, ok = val.([]interface{}); ok {
				i.triggers[name] = ArrInterfaceToArrayApplication(val.([]interface{}))
			}
		}
		return nil
	} else {
		return model.NewAppError("Iterator", "iterator.parse_app.trigger.valid_args", nil, "bad arguments", http.StatusBadRequest)
	}
}

func (f *Flow) TriggerScope(name string) (*Flow, *model.AppError) {
	if apps, ok := f.triggers[name]; ok {
		return f.Fork(fmt.Sprintf("trigger-%s", name), apps), nil
	}

	return nil, model.NewAppError("Iterator", "iterator.trigger.new_scope", nil, "not found "+name, http.StatusBadRequest)
}
