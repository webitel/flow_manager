package call

import (
	"cmp"
	"context"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type RecordFileArg struct {
	Name          string
	Type          string
	MaxSec        int    `json:"maxSec"`
	SilenceThresh int    `json:"silenceThresh"`
	SilenceHits   int    `json:"silenceHits"`
	Terminators   string `json:"terminators"`
	VoiceMail     bool   `json:"voiceMail"`
}

const (
	recordActionStop           string = "stop"
	defaultRecordSessionFormat string = "mp3"
	defaultRecordSessionMinSec int    = 2
	recordSessionTemplate      string = "${strepoch()}_${caller_id_number}_${destination_number}"
	defaultRecordingVarName    string = "wbt_record_session_path"
	defaultSysRecordingVarName string = "wbt_record_all_calls_path"
	sysRecordSessionSuffix     string = "_auto"
)

func formatSysRecordingName() string { return recordSessionTemplate + sysRecordSessionSuffix }

type RecordSessionArg struct {
	Action         string
	Name           string
	Type           string
	Stereo         bool
	Bridged        bool
	MinSec         int    `json:"minSec"`
	FollowTransfer bool   `json:"followTransfer"`
	Force          bool   `json:"force"`
	SetVar         string `json:"setVar"`
}

func (a *RecordSessionArg) Normalize() {
	if strings.TrimSpace(a.Type) == "" {
		a.Type = defaultRecordSessionFormat
	}

	if strings.TrimSpace(a.Name) == "" {
		a.Name = recordSessionTemplate
	}

	normalizeRecordName(&a.Name)
}

func (a *RecordSessionArg) IsStop() bool { return strings.TrimSpace(a.Action) == recordActionStop }

// PathVar returns the set path variable name if specified, or defaultRecordingVarName as a fallback.
func (a *RecordSessionArg) PathVar() string {
	return cmp.Or(strings.TrimSpace(a.SetVar), defaultRecordingVarName)
}

func (a *RecordSessionArg) ToRequest() *model.RecordSessionArgs {
	return &model.RecordSessionArgs{
		Name:           a.Name,
		Format:         a.Type,
		MinSec:         a.MinSec,
		Stereo:         a.Stereo,
		Bridged:        a.Bridged,
		FollowTransfer: a.FollowTransfer,
		Force:          a.Force,
		PathVar:        a.PathVar(),
	}
}

func (r *Router) recordFile(ctx context.Context, scope *flow.Flow, call model.Call, args any) (model.Response, *model.AppError) {
	argv := RecordFileArg{
		Name:          "recordFile_${strepoch()}",
		Type:          "mp3",
		MaxSec:        60,
		SilenceThresh: 200,
		SilenceHits:   5,
		Terminators:   "",
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Terminators != "" {
		if _, err := call.Set(ctx, map[string]any{
			"playback_terminators": argv.Terminators,
		}); err != nil {
			return nil, err
		}
	}

	if argv.VoiceMail {
		call.Push(ctx, "wbt_tags", "vm")
	}

	normalizeRecordName(&argv.Name)

	return call.RecordFile(ctx, argv.Name, argv.Type, argv.MaxSec, argv.SilenceThresh, argv.SilenceHits)
}

func (r *Router) recordSession(ctx context.Context, scope *flow.Flow, call model.Call, args any) (model.Response, *model.AppError) {
	argv := RecordSessionArg{MinSec: defaultRecordSessionMinSec}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	argv.Normalize()

	if argv.IsStop() {
		return call.RecordSessionStop(ctx, argv.PathVar())
	}

	return call.RecordSession(ctx, argv.ToRequest())
}

func (r *Router) startRecordAllCalls(ctx context.Context, call model.Call) *model.AppError {
	sysRecordRequest := &model.RecordSessionArgs{
		Name:           formatSysRecordingName(),
		Format:         defaultRecordSessionFormat,
		MinSec:         defaultRecordSessionMinSec,
		Stereo:         true,
		Bridged:        call.IVRQueueId() == nil,
		FollowTransfer: true,
		Force:          false,
		PathVar:        defaultSysRecordingVarName,
	}

	_, err := call.RecordSession(ctx, sysRecordRequest)

	return err
}

func normalizeRecordName(s *string) {
	if s == nil {
		return
	}

	if !strings.Contains(*s, " ") {
		return
	}

	*s = strings.ReplaceAll(*s, " ", "_")
}
