package call

import (
	"context"
	"encoding/json"
	"fmt"
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
	MinSec         int  `json:"minSec"`
	FollowTransfer bool `json:"followTransfer"`
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

	if argv.Name == "" {
		argv.Name = recordSessionTemplate
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

func schemaRecordsSession(schema *model.Schema) bool {
	if schema == nil {
		return false
	}

	data, err := json.Marshal(schema.Schema)
	if err != nil {
		return false
	}

	return strings.Contains(string(data), `"recordSession":`)
}

func (r *Router) forceRecordAllCalls(ctx context.Context, call model.Call) (model.Response, *model.AppError) {
	return call.Set(ctx, model.Variables{
		"RECORD_MIN_SEC":            2,
		"RECORD_STEREO":             true,
		"RECORD_BRIDGE_REQ":         false,
		"media_bug_answer_req":      false,
		"recording_follow_transfer": true,
		"execute_on_answer": fmt.Sprintf("record_session http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s.mp3&.mp3",
			call.DomainId(), call.Id(), recordSessionTemplate),
	})
}
