package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func ErrorRequiredParameter(appId string, param string) *model.AppError {
	return model.NewAppError("Valid", "valid.app."+appId, nil, fmt.Sprintf("App=%s %s is required", appId, param), http.StatusBadRequest)
}
func Error(appId string, err error) *model.AppError {
	return model.NewAppError("Error", "flow.app."+appId, nil, err.Error(), http.StatusBadRequest)
}
