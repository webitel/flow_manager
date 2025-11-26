package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/ai_bots"
	"github.com/webitel/flow_manager/model"
	"strconv"
)

func doStopStt(ctx context.Context, call model.Call, gs *model.GetSpeech, vSleepTimeout, vStatus, vFinal string) *model.AppError {
	if gs.Timeout > 0 && gs.BreakFinalOnTimeout {
		wbtError, _ := call.Get("wbt_stt_error")
		if wbtError != "" {
			return model.NewAppError("Playback.Stt", "call.stt.error",
				nil, wbtError, 500)
		}

		call.Set(ctx, map[string]interface{}{
			vSleepTimeout: "true",
		})
		isFinal, _ := call.Get(vStatus)
		if isFinal != vFinal {
			if _, err := call.Playback(ctx, []*model.PlaybackFile{{
				Type: model.NewString("silence"),
				Name: model.NewString(strconv.Itoa(gs.Timeout)),
			}}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Router) aiBridgeStt(ctx context.Context, call model.Call, argv model.PlaybackArgs) *model.AppError {

	gs := argv.GetSpeech

	if gs.SetVar == "" {
		gs.SetVar = "wbt_stt_text"
	}

	res, errGrpc := r.fm.AiBots.Bot().STT(ctx, &ai_bots.STTRequest{
		ProfileId:         gs.Profile.Id,
		DomainId:          call.DomainId(),
		CallId:            call.Id(),
		BreakFinalTimeout: gs.BreakFinalOnTimeout,
		DisableBreakFinal: gs.DisableBreakFinal,
		Lang:              gs.Lang,
		AlternativeLang:   gs.AlternativeLang,
		SetVar:            gs.SetVar,
		MinWords:          int32(gs.MinWords),
		MaxWords:          int32(gs.MaxWords),
		ExtraParams:       gs.ExtraParams,
	})

	if errGrpc != nil {
		return model.NewAppError("stt", "stt.ai_bridge", nil, errGrpc.Error(), 500)
	}

	con := res.GetConnected()

	_, err := call.StartRecognize(ctx, con.Connection, con.DialogId, int(con.InputRate), gs.VadTimeout)
	if err != nil {
		return err
	}

	if _, err = call.Playback(ctx, argv.Files); err != nil {
		return err
	}

	if gs.Timeout > 0 && gs.BreakFinalOnTimeout && gs.DisableBreakFinal {
		r.fm.AiBots.Bot().STTUpdateSession(ctx, &ai_bots.STTUpdateSessionRequest{
			DialogId:          con.DialogId,
			DisableBreakFinal: false,
		})
	}

	err = doStopStt(ctx, call, argv.GetSpeech, "wbt_play_sleep_timeout", "wbt_stt_status", "recognized")
	if err != nil {
		return err
	}

	_, err = call.StopRecognize(ctx)
	if err != nil {
		return err
	}

	return nil
}

func googleStt(ctx context.Context, call model.Call, argv model.PlaybackArgs) *model.AppError {
	if _, err := call.GoogleTranscribe(ctx, argv.GetSpeech); err != nil {
		return err
	}

	if _, err := call.Playback(ctx, argv.Files); err != nil {
		return err
	}
	if call.HangupCause() != "" {
		// todo err
	}

	err := doStopStt(ctx, call, argv.GetSpeech, "google_play_sleep_timeout", "google_final", "true")
	if err != nil {
		return err
	}

	if _, err := call.GoogleTranscribeStop(ctx); err != nil {
		return err
	}

	setSttVar(ctx, argv.GetSpeech.SetVar, call)

	return nil
}

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

		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			argv.Files = append(argv.Files, &model.PlaybackFile{
				Type: model.NewString("silence"),
				Name: model.NewString(strconv.Itoa(argv.GetSpeech.Timeout)),
			})
		}

		if argv.GetSpeech.Version == "v3" {
			err = r.aiBridgeStt(ctx, call, argv)
		} else {
			err = googleStt(ctx, call, argv)
		}

		if err != nil {
			return nil, err
		}

		return model.CallResponseOK, nil

	} else if argv.GetDigits != nil {
		return call.PlaybackAndGetDigits(ctx, argv.Files, argv.GetDigits)
	}

	return call.Playback(ctx, argv.Files)
}
