// Package call provides native ops for the call (ESL/FreeSWITCH) channel.
package call

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/model"
)

// callConnFromContext retrieves model.Call from the context decorator.
func callConnFromContext(ctx context.Context) (model.Call, bool) {
	conn := legacy.ConnectionFromContext(ctx)
	if conn == nil {
		return nil, false
	}
	c, ok := conn.(model.Call)
	return c, ok
}

// Register adds all simple call ops (no RouterDeps beyond model.Call) to reg.
func Register(reg *ops.Registry) {
	reg.Register("ringReady", syncOp(ringReadyFn))
	reg.Register("preAnswer", syncOp(preAnswerFn))
	reg.Register("answer", syncOp(answerFn))
	reg.Register("hangup", &hangupOp{})
	reg.Register("echo", &echoOp{})
	reg.Register("sleep", &sleepOp{})
	reg.Register("setAll", &setAllOp{})
	reg.Register("setNoLocal", &setNoLocalOp{})
	reg.Register("unSet", &unSetOp{})
	reg.Register("export", &exportOp{})
	reg.Register("flushDtmf", syncOp(flushDtmfFn))
	reg.Register("inBandDTMF", &inBandDTMFOp{})
	reg.Register("park", &parkOp{})
	reg.Register("pickup", &pickupOp{})
	reg.Register("sipRedirect", &sipRedirectOp{})
	reg.Register("markIVR", &markIVROp{})
	reg.Register("setSounds", &setSoundsOp{})
	reg.Register("scheduleHangup", &scheduleHangupOp{})
	reg.Register("amdML", &amdMLOp{})
	reg.Register("backgroundPlaybackStop", &backgroundPlaybackStopOp{})
	reg.Register("voiceBot", &voiceBotOp{})
	reg.Register("update", &updateOp{})
	reg.Register("conference", &conferenceOp{})
	reg.Register("recordFile", &recordFileOp{})
	reg.Register("recordSession", &recordSessionOp{})
	reg.Register("say", &sayOp{})
	reg.Register("cancelQueue", &cancelQueueOp{})
}

// syncOp wraps a no-args call method as an OpKindSync op.
type syncOp func(ctx context.Context, call model.Call) (model.Response, *model.AppError)

func (f syncOp) Kind() ops.OpKind { return ops.OpKindSync }

func (f syncOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("call op: no call connection in context")
	}
	if _, appErr := f(ctx, call); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

func ringReadyFn(ctx context.Context, call model.Call) (model.Response, *model.AppError) {
	return call.RingReady(ctx)
}
func preAnswerFn(ctx context.Context, call model.Call) (model.Response, *model.AppError) {
	return call.PreAnswer(ctx)
}
func answerFn(ctx context.Context, call model.Call) (model.Response, *model.AppError) {
	return call.Answer(ctx)
}
func flushDtmfFn(ctx context.Context, call model.Call) (model.Response, *model.AppError) {
	return call.FlushDTMF(ctx)
}

// ── hangup ────────────────────────────────────────────────────────────────────

type hangupOp struct{}

func (hangupOp) Kind() ops.OpKind { return ops.OpKindSync }

func (hangupOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("hangup: no call connection in context")
	}
	cause, _ := in.Node.RawArgs.(string)
	cause = ops.ExpandStr(cause, in.Variables, in.GlobalVar)
	if _, appErr := call.Hangup(ctx, cause); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── echo ──────────────────────────────────────────────────────────────────────

type echoOp struct{}

func (echoOp) Kind() ops.OpKind { return ops.OpKindSync }

func (echoOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("echo: no call connection in context")
	}
	var delay int
	ops.DecodeArgs(in, &delay) //nolint:errcheck
	if _, appErr := call.Echo(ctx, delay); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── sleep ─────────────────────────────────────────────────────────────────────

type sleepOp struct{}

func (sleepOp) Kind() ops.OpKind { return ops.OpKindSync }

func (sleepOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sleep: no call connection in context")
	}
	var timeout int
	ops.DecodeArgs(in, &timeout) //nolint:errcheck
	if _, appErr := call.Sleep(ctx, timeout); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── setAll ────────────────────────────────────────────────────────────────────
// Values are passed through unexpanded so FreeSWITCH can resolve ${var}
// references internally (same as legacy behaviour).

type setAllOp struct{}

func (setAllOp) Kind() ops.OpKind { return ops.OpKindSync }

func (setAllOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("setAll: no call connection in context")
	}
	vars, ok2 := in.Node.RawArgs.(map[string]any)
	if !ok2 {
		return ops.OpOutput{}, model.NewAppError("setAll", "call.set_all.valid.args", nil,
			fmt.Sprintf("bad arguments %v", in.Node.RawArgs), http.StatusBadRequest)
	}
	if _, appErr := call.SetAll(ctx, vars); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── setNoLocal ────────────────────────────────────────────────────────────────

type setNoLocalOp struct{}

func (setNoLocalOp) Kind() ops.OpKind { return ops.OpKindSync }

func (setNoLocalOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("setNoLocal: no call connection in context")
	}
	vars, ok2 := in.Node.RawArgs.(map[string]any)
	if !ok2 {
		return ops.OpOutput{}, model.NewAppError("setNoLocal", "call.set_no_local.valid.args", nil,
			fmt.Sprintf("bad arguments %v", in.Node.RawArgs), http.StatusBadRequest)
	}
	if _, appErr := call.SetNoLocal(ctx, vars); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── unSet ─────────────────────────────────────────────────────────────────────

type unSetOp struct{}

func (unSetOp) Kind() ops.OpKind { return ops.OpKindSync }

func (unSetOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("unSet: no call connection in context")
	}
	var argv []string
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if len(argv) == 0 {
		return ops.OpOutput{}, model.NewAppError("unSet", "call.unset.valid", nil, "value required", http.StatusBadRequest)
	}
	for _, v := range argv {
		if _, appErr := call.UnSet(ctx, v); appErr != nil {
			return ops.OpOutput{}, appErr
		}
	}
	return ops.OpOutput{}, nil
}

// ── export ────────────────────────────────────────────────────────────────────

type exportOp struct{}

func (exportOp) Kind() ops.OpKind { return ops.OpKindSync }

func (exportOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("export: no call connection in context")
	}
	var argv []string
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.Export(ctx, argv); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── inBandDTMF ────────────────────────────────────────────────────────────────

type inBandDTMFOp struct{}

func (inBandDTMFOp) Kind() ops.OpKind { return ops.OpKindSync }

func (inBandDTMFOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("inBandDTMF: no call connection in context")
	}
	conf, _ := in.Node.RawArgs.(string)
	conf = ops.ExpandStr(conf, in.Variables, in.GlobalVar)
	if conf == "stop" {
		_, appErr := call.StopDTMF(ctx)
		return ops.OpOutput{}, appErr
	}
	_, appErr := call.StartDTMF(ctx)
	return ops.OpOutput{}, appErr
}

// ── park ──────────────────────────────────────────────────────────────────────

type parkOp struct{}

func (parkOp) Kind() ops.OpKind { return ops.OpKindSync }

type parkArgs struct {
	Name string `json:"name"`
	Lot  string `json:"lot"`
	Auto string `json:"auto"`
}

func (parkOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("park: no call connection in context")
	}
	argv := parkArgs{Auto: "in"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Name == "" {
		return ops.OpOutput{}, model.NewAppError("park", "call.park.valid", nil, "name required", http.StatusBadRequest)
	}
	if argv.Lot == "" {
		return ops.OpOutput{}, model.NewAppError("park", "call.park.valid", nil, "lot required", http.StatusBadRequest)
	}
	lots := strings.SplitN(argv.Lot, "-", 2)
	from, to := lots[0], ""
	if len(lots) > 1 {
		to = lots[1]
	}
	if _, appErr := call.Park(ctx, argv.Name, argv.Auto == "in", from, to); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── pickup ────────────────────────────────────────────────────────────────────

type pickupOp struct{}

func (pickupOp) Kind() ops.OpKind { return ops.OpKindSync }

func (pickupOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("pickup: no call connection in context")
	}
	var argv struct {
		Name string `json:"name"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Name == "" {
		return ops.OpOutput{}, model.NewAppError("pickup", "call.pickup.valid", nil, "name required", http.StatusBadRequest)
	}
	if _, appErr := call.Pickup(ctx, argv.Name); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── sipRedirect ───────────────────────────────────────────────────────────────

type sipRedirectOp struct{}

func (sipRedirectOp) Kind() ops.OpKind { return ops.OpKindSync }

func (sipRedirectOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("sipRedirect: no call connection in context")
	}
	var argv []string
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.Redirect(ctx, argv); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── markIVR ───────────────────────────────────────────────────────────────────

type markIVROp struct{}

func (markIVROp) Kind() ops.OpKind { return ops.OpKindSync }

func (markIVROp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("markIVR: no call connection in context")
	}
	var argv struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Name == "" || argv.Value == "" {
		return ops.OpOutput{}, model.NewAppError("markIVR", "call.mark_ivr.valid", nil, "name and value required", http.StatusBadRequest)
	}
	key := fmt.Sprintf("usr_%s", strings.ReplaceAll(argv.Name, "'", ""))
	if _, appErr := call.Push(ctx, key, argv.Value); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── setSounds ─────────────────────────────────────────────────────────────────

type setSoundsOp struct{}

func (setSoundsOp) Kind() ops.OpKind { return ops.OpKindSync }

func (setSoundsOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("setSounds: no call connection in context")
	}
	var argv struct {
		Voice string `json:"voice"`
		Lang  string `json:"lang"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Lang == "" {
		return ops.OpOutput{}, model.NewAppError("setSounds", "call.set_sounds.valid", nil, "lang required", http.StatusBadRequest)
	}
	if argv.Voice == "" {
		return ops.OpOutput{}, model.NewAppError("setSounds", "call.set_sounds.valid", nil, "voice required", http.StatusBadRequest)
	}
	if _, appErr := call.SetSounds(ctx, argv.Lang, argv.Voice); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── scheduleHangup ────────────────────────────────────────────────────────────

type scheduleHangupOp struct{}

func (scheduleHangupOp) Kind() ops.OpKind { return ops.OpKindSync }

func (scheduleHangupOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("scheduleHangup: no call connection in context")
	}
	argv := struct {
		Seconds int    `json:"seconds"`
		Cause   string `json:"cause"`
	}{Seconds: 2}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.ScheduleHangup(ctx, argv.Seconds, argv.Cause); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── amdML ─────────────────────────────────────────────────────────────────────

type amdMLOp struct{}

func (amdMLOp) Kind() ops.OpKind { return ops.OpKindSync }

func (amdMLOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("amdML: no call connection in context")
	}
	var argv model.AmdMLParameters
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.AmdML(ctx, argv); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── backgroundPlaybackStop ────────────────────────────────────────────────────

type backgroundPlaybackStopOp struct{}

func (backgroundPlaybackStopOp) Kind() ops.OpKind { return ops.OpKindSync }

func (backgroundPlaybackStopOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("backgroundPlaybackStop: no call connection in context")
	}
	var argv struct {
		Name string `json:"name"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.BackgroundPlaybackStop(ctx, argv.Name); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── voiceBot ──────────────────────────────────────────────────────────────────

type voiceBotOp struct{}

func (voiceBotOp) Kind() ops.OpKind { return ops.OpKindSync }

func (voiceBotOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("voiceBot: no call connection in context")
	}
	var argv struct {
		Connection       string            `json:"connection"`
		Rate             string            `json:"rate"`
		InitialAiMessage string            `json:"initialAiMessage"`
		Variables        map[string]string `json:"variables"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Connection == "" {
		return ops.OpOutput{}, model.NewAppError("voiceBot", "call.voice_bot.valid", nil, "connection required", http.StatusBadRequest)
	}
	rate := 0
	switch argv.Rate {
	case "8kHz":
		rate = 8000
	case "16kHz":
		rate = 16000
	}
	if _, appErr := call.Bot(ctx, argv.Connection, rate, argv.InitialAiMessage, argv.Variables); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── update ────────────────────────────────────────────────────────────────────

type updateOp struct{}

func (updateOp) Kind() ops.OpKind { return ops.OpKindSync }

func (updateOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("update: no call connection in context")
	}
	if call.UserId() == 0 {
		return ops.OpOutput{}, model.NewRequestError("update", "this call is not an outbound")
	}
	var argv struct {
		Variables model.Variables `json:"variables"`
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if len(argv.Variables) > 0 {
		cp := make(model.Variables, len(argv.Variables))
		for k, v := range argv.Variables {
			if strings.HasPrefix(fmt.Sprintf("%v", k), "wbt_") {
				cp[k] = v
			} else {
				cp["usr_"+fmt.Sprintf("%v", k)] = v
			}
		}
		if _, appErr := call.Set(ctx, cp); appErr != nil {
			return ops.OpOutput{}, appErr
		}
	}
	if _, appErr := call.Update(ctx); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── conference ────────────────────────────────────────────────────────────────

type conferenceOp struct{}

func (conferenceOp) Kind() ops.OpKind { return ops.OpKindSync }

func (conferenceOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("conference: no call connection in context")
	}
	argv := struct {
		Name    string   `json:"name"`
		Profile string   `json:"profile"`
		Tags    []string `json:"tags"`
		Pin     string   `json:"pin"`
	}{Name: "global", Profile: "default"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if _, appErr := call.Conference(ctx, argv.Name, argv.Profile, argv.Pin, argv.Tags); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── recordFile ────────────────────────────────────────────────────────────────

const recordSessionTemplate = "${strepoch()}_${caller_id_number}_${destination_number}"

type recordFileOp struct{}

func (recordFileOp) Kind() ops.OpKind { return ops.OpKindSync }

func (recordFileOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("recordFile: no call connection in context")
	}
	argv := struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		MaxSec        int    `json:"maxSec"`
		SilenceThresh int    `json:"silenceThresh"`
		SilenceHits   int    `json:"silenceHits"`
		Terminators   string `json:"terminators"`
		VoiceMail     bool   `json:"voiceMail"`
	}{Name: "recordFile_${strepoch()}", Type: "mp3", MaxSec: 60, SilenceThresh: 200, SilenceHits: 5}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Terminators != "" {
		if _, appErr := call.Set(ctx, map[string]any{"playback_terminators": argv.Terminators}); appErr != nil {
			return ops.OpOutput{}, appErr
		}
	}
	if argv.VoiceMail {
		call.Push(ctx, "wbt_tags", "vm") //nolint:errcheck
	}
	normalizeRecordName(&argv.Name)
	if _, appErr := call.RecordFile(ctx, argv.Name, argv.Type, argv.MaxSec, argv.SilenceThresh, argv.SilenceHits); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── recordSession ─────────────────────────────────────────────────────────────

type recordSessionOp struct{}

func (recordSessionOp) Kind() ops.OpKind { return ops.OpKindSync }

func (recordSessionOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("recordSession: no call connection in context")
	}
	argv := struct {
		Action         string `json:"action"`
		Name           string `json:"name"`
		Type           string `json:"type"`
		Stereo         bool   `json:"stereo"`
		Bridged        bool   `json:"bridged"`
		MinSec         int    `json:"minSec"`
		FollowTransfer bool   `json:"followTransfer"`
	}{Type: "mp3", MinSec: 2}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Name == "" {
		argv.Name = recordSessionTemplate
	}
	normalizeRecordName(&argv.Name)
	if argv.Action == "stop" {
		if _, appErr := call.RecordSessionStop(ctx, argv.Name, argv.Type); appErr != nil {
			return ops.OpOutput{}, appErr
		}
		return ops.OpOutput{}, nil
	}
	if _, appErr := call.RecordSession(ctx, argv.Name, argv.Type, argv.MinSec, argv.Stereo, argv.Bridged, argv.FollowTransfer); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

func normalizeRecordName(s *string) {
	if strings.Contains(*s, " ") {
		*s = strings.ReplaceAll(*s, " ", "_")
	}
}

// ── say ───────────────────────────────────────────────────────────────────────

type sayOp struct{}

func (sayOp) Kind() ops.OpKind { return ops.OpKindSync }

func (sayOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("say: no call connection in context")
	}
	text, _ := in.Node.RawArgs.(string)
	text = ops.ExpandStr(text, in.Variables, in.GlobalVar)
	if _, appErr := call.Say(ctx, text); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}

// ── cancelQueue ───────────────────────────────────────────────────────────────

type cancelQueueOp struct{}

func (cancelQueueOp) Kind() ops.OpKind { return ops.OpKindSync }

func (cancelQueueOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	call, ok := callConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("cancelQueue: no call connection in context")
	}
	if _, appErr := call.Set(ctx, model.Variables{
		"cc_cancel": fmt.Sprintf("%v", call.CancelQueue()),
	}); appErr != nil {
		return ops.OpOutput{}, appErr
	}
	return ops.OpOutput{}, nil
}
