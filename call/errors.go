package call

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func ErrorRequiredParameter(appId string, param string) *model.AppError {
	return model.NewAppError("Valid", "valid.app."+appId, nil, fmt.Sprintf("App=%s %s is required", appId, param), http.StatusBadRequest)
}
