package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) Playback(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv model.PlaybackArgs

	err := r.Decode(scope, args, &argv)
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

	if argv.GetSpeech != nil {
		if _, err := call.GoogleTranscribe(ctx); err != nil {
			return nil, err
		}
		if _, err := call.Playback(ctx, argv.Files); err != nil {
			return nil, err
		}

		if err = r.fm.Store.Call().SaveTranscribe(call.Id(), call.GetVariable("variable_google_transcript")); err != nil {
			return nil, err
		}
		return model.CallResponseOK, nil

	} else if argv.GetDigits != nil {
		return call.PlaybackAndGetDigits(ctx, argv.Files, argv.GetDigits)
	}

	return call.Playback(ctx, argv.Files)
}
