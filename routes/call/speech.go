package call

import (
	"context"
	"encoding/json"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/pkg/processing"
)

type SpeechAiV2 struct {
	File       RecordFileArg
	Connection string              `json:"connection"`
	Context    processing.JsonView // todo
	Addresses  []string
}

func (r *Router) speechAiV2(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv SpeechAiV2
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	m := make(map[string]string)
	var b []byte
	b, _ = json.Marshal(argv.Context)
	m["context"] = string(b)
	b, _ = json.Marshal(argv.Addresses)
	m["addresses"] = string(b)

	_, err := call.SendFileToAi(ctx, argv.Connection, m, argv.File.Type, argv.File.MaxSec, argv.File.SilenceThresh, argv.File.SilenceHits)
	if err != nil {
		call.Set(ctx, model.Variables{
			"ai_error": err.Error(),
		})
		return nil, err
	}

	return model.CallResponseOK, nil

}
