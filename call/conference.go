package call

import (
	"github.com/webitel/flow_manager/model"
)

type ConferenceArgs struct {
	Name    string
	Profile string
	Tags    []string
	Pin     string
}

func (r *Router) conference(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var conf = ConferenceArgs{
		Name:    "global",
		Profile: "default",
		Tags:    nil,
	}

	if err := r.Decode(call, args, &conf); err != nil {
		return nil, err
	}

	return call.Conference(conf.Name, conf.Profile, conf.Pin, conf.Tags)
}