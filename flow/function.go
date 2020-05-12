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
			i.Functions[name] = New(Config{
				Name:    fmt.Sprintf("function-%s", name),
				Handler: i.handler,
				Schema:  ArrInterfaceToArrayApplication(actions),
				Conn:    i.conn,
			})
		}

		return nil
	} else {
		return model.NewAppError("Iterator", "iterator.parse_app.function.valid_args", nil, "bad arguments", http.StatusBadRequest)
	}
}
