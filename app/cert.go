package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

//todo export from storage

func (fm *FlowManager) GenerateSignature(msg []byte) (string, *model.AppError) {
	signature, err := fm.cert.Generate(msg)
	if err != nil {
		return "", model.NewAppError("GenerateSignature", "app.signature.generate.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return signature, nil
}

func (fm *FlowManager) GeneratePreSignetResourceSignature(resource, action string, uuid string, domainId int64, timeout int64) (string, *model.AppError) {
	key := fmt.Sprintf("%s/%s?domain_id=%d&uuid=%s&expires=%d", resource, action, domainId, uuid,
		(model.GetMillis() + timeout))

	signature, err := fm.GenerateSignature([]byte(key))
	if err != nil {
		return "", err
	}

	return key + "&signature=" + signature, nil

}
