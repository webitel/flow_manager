package call

import "github.com/webitel/flow_manager/model"

func (r *Router) recordFile(call model.Call, args interface{}) (model.Response, *model.AppError) {
	parameters, _ := args.(map[string]interface{})
	if parameters == nil {
		return model.CallResponseError, nil
	}

	var name = getStringValueFromMap("name", parameters, "recordFile")
	var terminators, _ = args.(string)
	var typeFile = getStringValueFromMap("type", parameters, "mp3")
	var maxSec = getIntValueFromMap("maxSec", parameters, 60)
	var silenceThresh = getIntValueFromMap("silenceThresh", parameters, 200)
	var silenceHits = getIntValueFromMap("silenceHits", parameters, 5)

	if terminators != "" {
		if _, err := call.Set(map[string]interface{}{
			"playback_terminators": terminators,
		}); err != nil {
			return nil, err
		}
	}

	return call.RecordFile(call.ParseText(name), typeFile, maxSec, silenceThresh, silenceHits)
}

func (r *Router) recordSession(call model.Call, args interface{}) (model.Response, *model.AppError) {
	parameters, _ := args.(map[string]interface{})
	if parameters == nil {
		return model.CallResponseError, nil
	}

	var name = getStringValueFromMap("name", parameters, "recordSession")
	var typeFile = getStringValueFromMap("type", parameters, "mp3")
	var minSec = getIntValueFromMap("minSec", parameters, 2)
	var stereo = getBoolValueFromMap("stereo", parameters, false)
	var bridged = getBoolValueFromMap("bridged", parameters, false)
	var followTransfer = getBoolValueFromMap("followTransfer", parameters, false)

	return call.RecordSession(call.ParseText(name), typeFile, minSec, stereo, bridged, followTransfer)

}
