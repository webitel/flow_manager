package call

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/webitel/flow_manager/api/gen/ai_bots"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// MediaDeps is the narrow interface required by the playback and tts ops.
type MediaDeps interface {
	GetMediaFiles(domainId int64, req *[]*calldomain.PlaybackFile) ([]*calldomain.PlaybackFile, error)
	GetPlaybackFile(domainId int64, search *calldomain.PlaybackFile) (*calldomain.PlaybackFile, error)
	GetAiBots() *aibridge.Client
}

// RegisterMedia adds playback and tts ops to reg.
func RegisterMedia(reg *ops.Registry, deps MediaDeps) {
	reg.Register("playback", &playbackOp{deps: deps})
	reg.Register("tts", &ttsOp{deps: deps})
}

// ── playback ──────────────────────────────────────────────────────────────────

type playbackOp struct{ deps MediaDeps }

func (playbackOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *playbackOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("playback: no call connection in context")
	}

	var argv calldomain.PlaybackArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.Files == nil {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "playback: files required")
	}

	var appErr error
	argv.Files, appErr = o.deps.GetMediaFiles(call.DomainId(), &argv.Files)
	if appErr != nil {
		return ops.OpOutput{}, appErr
	}

	if argv.GetSpeech != nil {
		bg := argv.GetSpeech.Background
		if bg != nil && bg.File != nil {
			bg.File, _ = o.deps.GetPlaybackFile(call.DomainId(), bg.File)
			bg.Name = utils.NewId()[:6]
			call.BackgroundPlayback(ctx, bg.File, bg.Name, bg.VolumeReduction) //nolint:errcheck
			defer call.BackgroundPlaybackStop(ctx, bg.Name)                    //nolint:errcheck
		}

		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			argv.Files = append(argv.Files, &calldomain.PlaybackFile{
				Type: utils.NewString("silence"),
				Name: utils.NewString(strconv.Itoa(argv.GetSpeech.Timeout)),
			})
		}

		if argv.GetSpeech.Version == "v3" {
			appErr = o.aiBridgeStt(ctx, call, argv)
		} else {
			appErr = googleStt(ctx, call, argv)
		}
		if appErr != nil {
			return ops.OpOutput{}, appErr
		}
		return ops.OpOutput{}, nil
	}

	if argv.GetDigits != nil {
		if _, appErr = call.PlaybackAndGetDigits(ctx, argv.Files, argv.GetDigits); appErr != nil {
			return ops.OpOutput{}, appErr
		}
		return ops.OpOutput{}, nil
	}

	if _, appErr = call.Playback(ctx, argv.Files); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

func (o *playbackOp) aiBridgeStt(ctx context.Context, call calldomain.Call, argv calldomain.PlaybackArgs) error {
	gs := argv.GetSpeech
	if gs.SetVar == "" {
		gs.SetVar = "wbt_stt_text"
	}
	res, errGrpc := o.deps.GetAiBots().Bot().STT(ctx, &ai_bots.STTRequest{
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
		return fmt.Errorf("stt: stt.ai_bridge: %w", errGrpc)
	}
	con := res.GetConnected()
	if _, err := call.StartRecognize(ctx, con.Connection, con.DialogId, int(con.InputRate), gs.VadTimeout); err != nil {
		return err
	}
	if _, err := call.Playback(ctx, argv.Files); err != nil {
		return err
	}
	if gs.Timeout > 0 && gs.BreakFinalOnTimeout && gs.DisableBreakFinal {
		o.deps.GetAiBots().Bot().STTUpdateSession(ctx, &ai_bots.STTUpdateSessionRequest{ //nolint:errcheck
			DialogId:          con.DialogId,
			DisableBreakFinal: false,
		})
	}
	if err := doStopStt(ctx, call, gs, "wbt_play_sleep_timeout", "wbt_stt_final", "true"); err != nil {
		return err
	}
	if _, err := call.StopRecognize(ctx); err != nil {
		return err
	}
	return nil
}

func googleStt(ctx context.Context, call calldomain.Call, argv calldomain.PlaybackArgs) error {
	if _, err := call.GoogleTranscribe(ctx, argv.GetSpeech); err != nil {
		return err
	}
	if _, err := call.Playback(ctx, argv.Files); err != nil {
		return err
	}
	if err := doStopStt(ctx, call, argv.GetSpeech, "google_play_sleep_timeout", "google_final", "true"); err != nil {
		return err
	}
	if _, err := call.GoogleTranscribeStop(ctx); err != nil {
		return err
	}
	setSttVar(ctx, argv.GetSpeech.SetVar, call)
	return nil
}

func doStopStt(ctx context.Context, call calldomain.Call, gs *calldomain.GetSpeech, vSleepTimeout, vStatus, vFinal string) error {
	if gs.Timeout <= 0 || !gs.BreakFinalOnTimeout {
		return nil
	}
	wbtError, _ := call.Get("wbt_stt_error")
	if wbtError != "" {
		return fmt.Errorf("Playback.Stt: call.stt.error: %s", wbtError)
	}
	call.Set(ctx, map[string]any{vSleepTimeout: "true"}) //nolint:errcheck
	isFinal, _ := call.Get(vStatus)
	if isFinal != vFinal {
		if _, err := call.Playback(ctx, []*calldomain.PlaybackFile{{
			Type: utils.NewString("silence"),
			Name: utils.NewString(strconv.Itoa(gs.Timeout)),
		}}); err != nil {
			return err
		}
	}
	return nil
}

func setSttVar(ctx context.Context, varName string, call calldomain.Call) {
	if varName == "" {
		varName = "google_refresh_vars"
	}
	call.Set(ctx, map[string]any{varName: "${google_transcript}"}) //nolint:errcheck
}

// ── tts ───────────────────────────────────────────────────────────────────────

type ttsOp struct{ deps MediaDeps }

func (ttsOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *ttsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("tts: no call connection in context")
	}

	var argv struct {
		calldomain.TTSSettings
		Provider   string                `json:"provider"`
		Key        string                `json:"key"`
		Token      string                `json:"token"`
		Region     string                `json:"region"`
		Terminator string                `json:"terminator"`
		GetDigits  *calldomain.PlaybackDigits `json:"getDigits"`
		GetSpeech  *calldomain.GetSpeech      `json:"getSpeech"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if err := ops.DecodeArgs(in, &argv.TTSSettings); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.Text == "" {
		return ops.OpOutput{}, apperrs.New(http.StatusBadRequest, "tts: text required")
	}

	q := buildTTSQuery(argv.Provider, argv.Key, argv.Token, argv.Region)

	if argv.GetSpeech != nil {
		bg := argv.GetSpeech.Background
		if bg != nil && bg.File != nil {
			bg.File, _ = o.deps.GetPlaybackFile(call.DomainId(), bg.File)
			bg.Name = utils.NewId()[:6]
			call.BackgroundPlayback(ctx, bg.File, bg.Name, bg.VolumeReduction) //nolint:errcheck
			defer call.BackgroundPlaybackStop(ctx, bg.Name)                    //nolint:errcheck
		}

		if _, err := call.GoogleTranscribe(ctx, argv.GetSpeech); err != nil {
			return ops.OpOutput{}, err
		}

		timeout := 0
		if argv.GetSpeech.Timeout > 0 && !argv.GetSpeech.BreakFinalOnTimeout {
			timeout = argv.GetSpeech.Timeout
		}

		if _, err := call.TTS(ctx, q, argv.TTSSettings, argv.GetDigits, timeout); err != nil {
			return ops.OpOutput{}, err
		}

		if argv.GetSpeech.Timeout > 0 && argv.GetSpeech.BreakFinalOnTimeout {
			wbtError, _ := call.Get("wbt_stt_error")
			if wbtError != "" {
				return ops.OpOutput{}, fmt.Errorf("tts.stt: call.stt.error: %s", wbtError)
			}
			call.Set(ctx, map[string]any{"google_play_sleep_timeout": "true"}) //nolint:errcheck
			isFinal, _ := call.Get("google_final")
			if isFinal != "true" {
				if _, err := call.Playback(ctx, []*calldomain.PlaybackFile{{
					Type: utils.NewString("silence"),
					Name: utils.NewString(strconv.Itoa(argv.GetSpeech.Timeout)),
				}}); err != nil {
					return ops.OpOutput{}, err
				}
			}
		}

		if _, err := call.GoogleTranscribeStop(ctx); err != nil {
			return ops.OpOutput{}, err
		}

		setSttVar(ctx, argv.GetSpeech.SetVar, call)

		answer := call.GetVariable("variable_google_transcript")
		if argv.GetSpeech.Question != "" {
			call.PushSpeechMessage(calldomain.SpeechMessage{
				Question: argv.GetSpeech.Question,
				Answer:   answer,
			})
		}
		return ops.OpOutput{}, nil
	}

	if argv.Provider == "yandex" {
		if _, err := call.TTSOpus(ctx, q, argv.GetDigits, 0); err != nil {
			return ops.OpOutput{}, err
		}
		return ops.OpOutput{}, nil
	}

	if _, err := call.TTS(ctx, q, argv.TTSSettings, argv.GetDigits, 0); err != nil {
		return ops.OpOutput{}, err
	}
	return ops.OpOutput{}, nil
}

func buildTTSQuery(provider, key, token, region string) string {
	switch provider {
	case "polly":
		provider = "/polly?"
	case "microsoft":
		provider = "/microsoft?"
	case "yandex":
		provider = "/yandex?"
	case "webitel":
		provider = "/webitel?"
	case "google":
		provider = "/google?"
	default:
		provider = "/?"
	}
	q := provider
	if key != "" {
		q += "&key=" + utils.UrlEncoded(key)
	}
	if token != "" {
		q += "&token=" + utils.UrlEncoded(token)
	}
	if region != "" {
		q += "&region=" + region
	}
	return q + "&"
}
