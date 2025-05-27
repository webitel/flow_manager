package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"strconv"
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
		background := argv.GetSpeech.Background
		if background != nil && background.File != nil {
			background.File, _ = r.fm.GetPlaybackFile(call.DomainId(), background.File)
			background.Name = model.NewId()[:6]
			call.BackgroundPlayback(ctx, background.File, background.Name, background.VolumeReduction)
			defer call.BackgroundPlaybackStop(ctx, background.Name)
		}
		if _, err := call.GoogleTranscribe(ctx, argv.GetSpeech); err != nil {
			return nil, err
		}

		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			argv.Files = append(argv.Files, &model.PlaybackFile{
				Type: model.NewString("silence"),
				Name: model.NewString(strconv.Itoa(argv.GetSpeech.Timeout)),
			})
		}

		if _, err := call.Playback(ctx, argv.Files); err != nil {
			return nil, err
		}
		if call.HangupCause() != "" {
			// todo err
		}
		if argv.GetSpeech.Timeout > 0 && argv.GetSpeech.BreakFinalOnTimeout {
			wbtError, _ := call.Get("wbt_stt_error")
			if wbtError != "" {
				return model.CallResponseError, model.NewAppError("Playback.Stt", "call.stt.error",
					nil, wbtError, 500)
			}

			call.Set(ctx, map[string]interface{}{
				"google_play_sleep_timeout": "true",
			})
			isFinal, _ := call.Get("google_final")
			if isFinal != "true" {
				if _, err := call.Playback(ctx, []*model.PlaybackFile{{
					Type: model.NewString("silence"),
					Name: model.NewString(strconv.Itoa(argv.GetSpeech.Timeout)),
				}}); err != nil {
					return nil, err
				}
			}
		}

		if _, err := call.GoogleTranscribeStop(ctx); err != nil {
			return nil, err
		}

		setSttVar(ctx, argv.GetSpeech.SetVar, call)

		return model.CallResponseOK, nil

	} else if argv.GetDigits != nil {
		return call.PlaybackAndGetDigits(ctx, argv.Files, argv.GetDigits)
	}

	return call.Playback(ctx, argv.Files)
}
