package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"golang.org/x/sync/singleflight"
)

var (
	systemCache = utils.NewLruWithParams(100, "system_settings", 60, "")
	systemGroup = singleflight.Group{}
)

func (fm *FlowManager) GetSystemSettingsString(ctx context.Context, domainId int64, name string) (string, *model.AppError) {
	key := fmt.Sprintf("%d-%s", domainId, name)
	c, ok := systemCache.Get(key)
	if ok {
		return c.(string), nil
	}

	v, err, share := systemGroup.Do(fmt.Sprintf("%d-%s", domainId, name), func() (interface{}, error) {
		res, err := fm.Store.SystemcSettings().Get(ctx, domainId, name)
		if err != nil {
			return "", err
		}
		var val string
		json.Unmarshal(res, &val)
		return val, nil
	})

	if err != nil {
		switch err.(type) {
		case *model.AppError:
			return "", err.(*model.AppError)
		default:
			return "", model.NewInternalError("app.sys_settings.get", err.Error())
		}
	}

	if !share {
		systemCache.Add(key, v.(string))
	}

	return v.(string), nil
}
