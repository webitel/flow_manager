package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
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

	q := ""

	switch argv.Provider {
	case "polly":
		q = "/polly?"
	case "microsoft":
		q = "/microsoft?"
	case "yandex":
		q = "/yandex?"
	case "webitel":
		q = "/webitel?"
	case "google":
		q = "/google?"

	default:
		q = "/?"
	}

	if argv.Key != "" {
		q += "&key=" + model.UrlEncoded(argv.Key)
	}

	if argv.Token != "" {
		q += "&token=" + model.UrlEncoded(argv.Token)
	}

	if argv.Region != "" {
		q += "&region=" + argv.Region
	}

	q += "&"

	if argv.GetSpeech != nil {
		if _, err := call.GoogleTranscribe(ctx, argv.GetSpeech); err != nil {
			return nil, err
		}

		timeout := 0
		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			timeout = argv.GetSpeech.Timeout
		}

		if _, err := call.TTS(ctx, q, argv.TTSSettings, argv.GetDigits, timeout); err != nil {
			return nil, err
		}

		if call.HangupCause() != "" {
			// todo err
		}
		if argv.GetSpeech.Timeout > 0 && argv.GetSpeech.BreakFinalOnTimeout {
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

		call.Set(ctx, map[string]interface{}{
			"google_refresh_vars": "todo",
		}) // TODO refresh vars

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
		return call.TTSOpus(ctx, q, argv.GetDigits, 0)
	}
	return call.TTS(ctx, q, argv.TTSSettings, argv.GetDigits, 0)
}
