package call

import (
	"context"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

const recordSessionTemplate = "${strepoch()}_${caller_id_number}_${destination_number}"

type RecordFileArg struct {
	Name          string
	Type          string
	MaxSec        int    `json:"maxSec"`
	SilenceThresh int    `json:"silenceThresh"`
	SilenceHits   int    `json:"silenceHits"`
	Terminators   string `json:"terminators"`
	VoiceMail     bool   `json:"voiceMail"`
}

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

const (
	RecordActionStop           string = "stop"
	DefaultRecordSessionFormat string = "mp3"
	DefaultRecordSessionMinSec int    = 2
)

func (a *RecordSessionArg) IsStop() bool       { return strings.TrimSpace(a.Action) == RecordActionStop }
func (a *RecordSessionArg) HasEmptyName() bool { return strings.TrimSpace(a.Name) == "" }

func (a *RecordSessionArg) SetVarOrDefault() string {
	if a == nil {
		return model.DefaultRecordingVarName
	}

	if strings.TrimSpace(a.SetVar) == "" {
		return model.DefaultRecordingVarName
	}

	return a.SetVar
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
	argv := RecordSessionArg{
		Type:   DefaultRecordSessionFormat,
		MinSec: DefaultRecordSessionMinSec,
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.HasEmptyName() {
		argv.Name = model.RecordSessionTemplate
	}

	normalizeRecordName(&argv.Name)

	if argv.IsStop() {
		return call.RecordSessionStop(ctx, argv.SetVarOrDefault())
	}

	recordSessionRequest := &model.RecordSessionArgs{
		Name:           argv.Name,
		Format:         argv.Type,
		MinSec:         argv.MinSec,
		Stereo:         argv.Stereo,
		Bridged:        argv.Bridged,
		FollowTransfer: argv.FollowTransfer,
		Force:          argv.Force,
		PathVar:        argv.SetVarOrDefault(),
	}

	return call.RecordSession(ctx, recordSessionRequest)
}

func (r *Router) startRecordAllCalls(ctx context.Context, call model.Call) *model.AppError {
	sysRecordRequest := &model.RecordSessionArgs{
		Name:           model.RecordSessionTemplate,
		Format:         DefaultRecordSessionFormat,
		MinSec:         DefaultRecordSessionMinSec,
		Stereo:         true,
		Bridged:        call.IVRQueueId() == nil,
		FollowTransfer: true,
		Force:          false,
		PathVar:        model.DefaultSysRecordingVarName,
	}

	_, err := call.RecordSession(ctx, sysRecordRequest)

	return err
}

func normalizeRecordName(s *string) {
	if strings.Index(*s, " ") != -1 {
		*s = strings.Replace(*s, " ", "_", -1)
	}
}
