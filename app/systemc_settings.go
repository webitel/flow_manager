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
	systemCache = utils.NewLruWithParams(300, "system_settings", 60, "")
	systemGroup = singleflight.Group{}
)

func (fm *FlowManager) GetSystemSettingsString(ctx context.Context, domainId int64, name string) (string, *model.AppError) {
	s, err := fm.GetSystemSettings(ctx, domainId, name)
	if err != nil {
		return "", err
	}

	return s.StringValue, nil
}

func (fm *FlowManager) GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, *model.AppError) {
	key := fmt.Sprintf("%d-%s", domainId, name)
	c, ok := systemCache.Get(key)
	if ok {
		return c.(model.SysValue), nil
	}

	v, err, share := systemGroup.Do(fmt.Sprintf("%d-%s", domainId, name), func() (interface{}, error) {
		res, err := fm.Store.SystemcSettings().Get(ctx, domainId, name)
		if err != nil {
			return model.SysValue{}, err
		}
		var val interface{}
		var s model.SysValue
		json.Unmarshal(res, &val)
		switch b := val.(type) {
		case bool:
			s.BoolValue = b
		case string:
			s.StringValue = b
		}

		return s, nil
	})

	if err != nil {
		switch err.(type) {
		case *model.AppError:
			return model.SysValue{}, err.(*model.AppError)
		default:
			return model.SysValue{}, model.NewInternalError("app.sys_settings.get", err.Error())
		}
	}

	if !share {
		systemCache.Add(key, v.(model.SysValue))
	}

	return v.(model.SysValue), nil
}
