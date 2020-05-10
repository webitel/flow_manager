package call

import "github.com/webitel/flow_manager/model"

type SoundsArgs struct {
	Voice string
	Lang  string
}

func (r *Router) SetSounds(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv SoundsArgs
	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	if argv.Lang == "" {
		return nil, ErrorRequiredParameter("setSounds", "lang")
	}

	if argv.Voice == "" {
		return nil, ErrorRequiredParameter("setSounds", "voice")
	}

	return call.SetSounds(argv.Lang, argv.Voice)
}
