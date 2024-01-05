package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
	"golang.org/x/sync/singleflight"
)

var hookGroup singleflight.Group

func (f *FlowManager) GetHookById(key string) (hook model.WebHook, err *model.AppError) {

	v, err2, _ := hookGroup.Do(key, func() (interface{}, error) {
		h, e := f.Store.WebHook().Get(key)
		if e != nil {
			return h, e
		}

		return h, nil
	})

	if err2 != nil {
		switch err2.(type) {
		case *model.AppError:
			return hook, err2.(*model.AppError)
		default:
			return hook, model.NewAppError("Hook", "hook.settings.get", nil, err2.Error(), http.StatusInternalServerError)
		}
	}

	return v.(model.WebHook), nil

}
