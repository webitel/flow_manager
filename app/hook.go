package app

import (
	"github.com/webitel/flow_manager/model"
	"golang.org/x/sync/singleflight"
)

var hookGroup singleflight.Group

func (f *FlowManager) GetHookById(id string) (hook model.WebHook, err *model.AppError) {
	return f.Store.WebHook().Get(id)
}
