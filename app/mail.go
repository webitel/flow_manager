package app

import (
	"fmt"

	"github.com/webitel/flow_manager/model"
	"golang.org/x/sync/singleflight"
)

var mailGroup singleflight.Group

func (f *FlowManager) SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, error) {
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
		return nil, err
	}

	return settings.(*model.SmtSettings), nil
}

