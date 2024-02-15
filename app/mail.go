package app

import (
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"golang.org/x/sync/singleflight"
)

var mailGroup singleflight.Group

func (f *FlowManager) SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, *model.AppError) {
	key := fmt.Sprintf("%d-", domainId)
	if search.Id != nil {
		key += fmt.Sprintf("%d-", *search.Id)
	}

	if search.Name != nil {
		key += fmt.Sprintf("%s-", *search.Name)
	}

	settings, err, _ := mailGroup.Do(key, func() (interface{}, error) {
		settings, err := f.Store.Email().SmtpSettings(domainId, search)
		if err != nil {
			return nil, err
		}

		return settings, nil
	})

	if err != nil {
		switch err.(type) {
		case *model.AppError:
			return nil, err.(*model.AppError)
		default:
			return nil, model.NewAppError("Queue", "mail.smtp.settings.get", nil, err.Error(), http.StatusInternalServerError)
		}
	}

	return settings.(*model.SmtSettings), nil
}
