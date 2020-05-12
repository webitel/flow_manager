package call

import (
	"github.com/webitel/flow_manager/model"
)

type RecordFileArg struct {
	Name          string
	Type          string
	MaxSec        int    `json:"maxSec"`
	SilenceThresh int    `json:"silenceThresh"`
	SilenceHits   int    `json:"silenceHits"`
	Terminators   string `json:"terminators"`
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

func (r *Router) recordFile(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv = RecordFileArg{
		Name:          "recordFile",
		Type:          "mp3",
		MaxSec:        60,
		SilenceThresh: 200,
		SilenceHits:   5,
		Terminators:   "",
	}

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	if argv.Terminators != "" {
		if _, err := call.Set(map[string]interface{}{
			"playback_terminators": argv.Terminators,
		}); err != nil {
			return nil, err
		}
	}

	return call.RecordFile(argv.Name, argv.Type, argv.MaxSec, argv.SilenceThresh, argv.SilenceHits)
}

// FIXME test record stop
func (r *Router) recordSession(call model.Call, args interface{}) (model.Response, *model.AppError) {

	var argv = RecordSessionArg{
		Name:   "recordSession",
		Type:   "mp3",
		MinSec: 2,
	}

	if err := r.Decode(call, args, &argv); err != nil {
		return nil, err
	}

	if argv.Action == "stop" {
		return call.RecordSessionStop(argv.Name, argv.Type)
	}

	return call.RecordSession(argv.Name, argv.Type, argv.MinSec, argv.Stereo, argv.Bridged, argv.FollowTransfer)
}