package call

import (
	"context"

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

type RecordSessionArg struct {
	Action         string
	Name           string
	Type           string
	Stereo         bool
	Bridged        bool
	Force          bool   `json:"force"`
	MinSec         int    `json:"minSec"`
	FollowTransfer bool   `json:"followTransfer"`
	SetVar         string `json:"setVar"`
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

	model.NormalizeRecordName(&argv.Name)

	return call.RecordFile(ctx, argv.Name, argv.Type, argv.MaxSec, argv.SilenceThresh, argv.SilenceHits)
}

// FIXME test record stop
func (r *Router) recordSession(ctx context.Context, scope *flow.Flow, call model.Call, args any) (model.Response, *model.AppError) {
	argv := RecordSessionArg{
		Name:   "",
		Type:   "mp3",
		MinSec: 2,
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	sessionRecord := &model.SessionRecord{
		RecordingNameVar: argv.SetVar,
		ActionStr:        argv.Action,
		DomainID:         call.DomainId(),
		CallID:           call.Id(),
		Name:             argv.Name,
		Format:           argv.Type,
		Force:            argv.Force,
		AudioConfig: model.SessionRecordAudioConfig{
			RecordMinSec:           argv.MinSec,
			RecordStereo:           argv.Stereo,
			RecordBridgeReq:        argv.Bridged,
			MediaBugAnswerReq:      argv.Bridged,
			RecordingFolowTransfer: argv.FollowTransfer,
		},
	}

	sessionRecord.Prepare()

	return call.RecordAction(ctx, sessionRecord)
}

func (r *Router) startRecordAllCalls(ctx context.Context, call model.Call) *model.AppError {
	cfg := model.DefaultSessionRecordConfig(
		model.WithDomainID(call.DomainId()),
		model.WithCallID(call.Id()),
		model.WithSessionRecordBridged(call.IVRQueueId() == nil),
	)

	cfg.Prepare()

	_, err := call.RecordAction(ctx, cfg)

	return err
}
