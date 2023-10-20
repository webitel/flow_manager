package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

const (
	TriggerDisconnected = "disconnected"
	TriggerCommands     = "commands"
)

// todo
func (i *Flow) addTrigger(args interface{}) *model.AppError {
	var ok bool
	var tmp map[string]interface{}

	if tmp, ok = args.(map[string]interface{}); ok {
		for name, val := range tmp {
			switch name {
			case TriggerDisconnected:
				if _, ok = val.([]interface{}); ok {
					i.triggers[name] = ArrInterfaceToArrayApplication(val.([]interface{}))
				}
			case TriggerCommands:
				if tmp, ok = val.(map[string]interface{}); ok {
					for c, v := range tmp {
						if _, ok = v.([]interface{}); ok {
							i.triggers[TriggerCommandsName(c)] = ArrInterfaceToArrayApplication(v.([]interface{}))
						}
					}
				}
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

func (f *Flow) TriggerScopeAsync(ctx context.Context, name string, h Handler) *model.AppError {
	if apps, ok := f.triggers[name]; ok {
		s := f.Fork(fmt.Sprintf("trigger-%s", name), apps)
		go Route(ctx, s, h)
		return nil
	}

	return model.NewAppError("Iterator", "iterator.trigger.new_scope", nil, "not found "+name, http.StatusBadRequest)
}

func (f *Flow) CountTriggers() int {
	return len(f.triggers)
}

func (f *Flow) HasTrigger(c string) bool {
	_, ok := f.triggers[c]

	return ok
}

func TriggerCommandsName(c string) string {
	return fmt.Sprintf("%s-%s", TriggerCommands, c)
}
