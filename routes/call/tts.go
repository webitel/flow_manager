package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/url"
	"strconv"
)

type TTSArgs struct {
	model.TTSSettings
	Key      string `json:"key"`      // todo deprecated
	Token    string `json:"token"`    // todo deprecated
	Provider string `json:"provider"` // todo deprecated
	Region   string `json:"region"`   // todo deprecated

	Terminator string                `json:"terminator"`
	GetDigits  *model.PlaybackDigits `json:"getDigits"`
	GetSpeech  *model.GetSpeech      `json:"getSpeech"`
}

func (r *Router) TTS(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv TTSArgs
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}
	if err := r.Decode(scope, args, &argv.TTSSettings); err != nil {
		return nil, err
	}

	if argv.Text == "" {
		return model.CallResponseError, ErrorRequiredParameter("tts", "text")
	}

	var provider string

	switch argv.Provider {
	case "polly", "microsoft", "yandex", "webitel", "google":
		provider = argv.Provider
	}

	params := url.Values{}

	if argv.Key != "" {
		params.Set("key", argv.Key)
	}

	if argv.Token != "" {
		params.Set("token", argv.Token)
	}

	if argv.Region != "" {
		params.Set("region", argv.Region)
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

		timeout := 0
		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			timeout = argv.GetSpeech.Timeout
		}

		if _, err := call.TTS(ctx, provider, params, argv.TTSSettings, argv.GetDigits, timeout); err != nil {
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

		answer := call.GetVariable("variable_google_transcript")
		if argv.GetSpeech.Question != "" {
			call.PushSpeechMessage(model.SpeechMessage{
				Question: argv.GetSpeech.Question,
				Answer:   answer,
			})
		}

		return model.CallResponseOK, nil
	}
	if argv.Provider == "yandex" {
		return call.TTSOpus(ctx, provider, params, argv.GetDigits, 0)
	}
	return call.TTS(ctx, provider, params, argv.TTSSettings, argv.GetDigits, 0)
}

func setSttVar(ctx context.Context, varName string, call model.Call) {
	if varName == "" {
		varName = "google_refresh_vars" // need execute event
	}
	call.Set(ctx, map[string]any{
		varName: "${google_transcript}",
	})
}
