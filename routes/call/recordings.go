package call

import (
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

type RecordSessionArg struct {
	Action         string
	Name           string
	Type           string
	Stereo         bool
	Bridged        bool
	MinSec         int  `json:"minSec"`
	FollowTransfer bool `json:"followTransfer"`
}

func (r *Router) recordFile(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv = RecordFileArg{
		Name:          "recordFile",
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
		if _, err := call.Set(ctx, map[string]interface{}{
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

// FIXME test record stop
func (r *Router) recordSession(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {

	var argv = RecordSessionArg{
		Name:   "${caller_id_number}_${destination_number}",
		Type:   "mp3",
		MinSec: 2,
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	normalizeRecordName(&argv.Name)

	if argv.Action == "stop" {
		return call.RecordSessionStop(ctx, argv.Name, argv.Type)
	}

	return call.RecordSession(ctx, argv.Name, argv.Type, argv.MinSec, argv.Stereo, argv.Bridged, argv.FollowTransfer)
}

func normalizeRecordName(s *string) {
	if strings.Index(*s, " ") != -1 {
		*s = strings.Replace(*s, " ", "_", -1)
	}
}
