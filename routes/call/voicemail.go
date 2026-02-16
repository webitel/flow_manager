package call

import "github.com/webitel/flow_manager/model"

type VoicemailArgs struct {
	User             string
	Announce         string
	Check            bool
	SkipGreeting     bool
	SkipInstructions bool
	Auth             *bool
	CC               []string
}

// TODO
func (r *Router) Voicemail(call model.Call, args interface{}) (model.Response, *model.AppError) {
	return model.CallResponseError, nil
}
