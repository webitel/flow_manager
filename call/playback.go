package call

import (
	"github.com/webitel/flow_manager/model"
)

func (r *Router) Playback(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv model.PlaybackArgs

	err := r.Decode(call, args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Files == nil {
		return nil, ErrorRequiredParameter("playback", "files")
	}

	argv.Files, err = r.fm.GetMediaFiles(call.DomainId(), &argv.Files)
	if err != nil {
		return nil, err
	}

	if argv.GetDigits != nil {
		return call.PlaybackAndGetDigits(argv.Files, argv.GetDigits)
	}

	return call.Playback(argv.Files)
}