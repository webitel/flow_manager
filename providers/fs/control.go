package fs

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"strconv"
	"strings"
)

const (
	HANGUP_NORMAL_TEMPORARY_FAILURE = "NORMAL_TEMPORARY_FAILURE"
	HANGUP_NO_ROUTE_DESTINATION     = "NO_ROUTE_DESTINATION"
)

func (c *Connection) Answer(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "answer", "")
}

func (c *Connection) PreAnswer(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "pre_answer", "")
}

func (c *Connection) RingReady(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "ring_ready", "")
}

func (c *Connection) Hangup(ctx context.Context, cause string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "hangup", cause)
}

func (c *Connection) HangupNoRoute(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "hangup", HANGUP_NO_ROUTE_DESTINATION)
}

func (c *Connection) HangupAppErr(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "hangup", HANGUP_NORMAL_TEMPORARY_FAILURE)
}

func (c *Connection) Sleep(ctx context.Context, timeout int) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "sleep", fmt.Sprintf("%d", timeout))
}

// FIXME GLOBAL VARS
func (c *Connection) Bridge(ctx context.Context, call model.Call, strategy string, vars map[string]string, endpoints []*model.Endpoint, codecs []string) (model.Response, *model.AppError) {
	var dialString, separator string

	if strategy == "failover" {
		separator = "|"
	} else if strategy != "" && strategy != "multiple" {
		separator = ":_:"
	} else {
		separator = ","
	}

	var from string
	// FIXME
	//origination_callee_id_name

	from = fmt.Sprintf("sip_copy_custom_headers=false,sip_h_X-Webitel-Domain-Id=%d,sip_h_X-Webitel-Origin=flow,wbt_parent_id=%s,wbt_from_type=%s,wbt_from_id=%s,wbt_destination='%s'",
		call.DomainId(), call.Id(), call.From().Type, call.From().Id, call.Destination())

	from += fmt.Sprintf(",effective_caller_id_name='%s',effective_caller_id_number='%s'", call.From().Name, call.From().Number)

	dialString += "<sip_route_uri=sip:$${outbound_sip_proxy}," + from
	for key, val := range vars {
		dialString += fmt.Sprintf(",'%s'='%s'", key, val)
	}

	if codecs != nil {
		dialString += ",absolute_codec_string='" + strings.Join(codecs, ",") + "'"
	}

	if len(c.exportVariables) > 0 {
		for _, v := range c.exportVariables {
			if val, ok := c.Get(v); ok {
				dialString += fmt.Sprintf(",'usr_%s'='%s'", v, val)
			}
		}
	}

	dialString += ">"

	end := make([]string, 0, len(endpoints))

	for _, e := range endpoints {
		switch e.TypeName {
		case "gateway":
			if e == nil || e.Destination == nil {
				end = append(end, "error/UNALLOCATED_NUMBER")
			} else if e.Dnd != nil && *e.Dnd {
				end = append(end, "error/GATEWAY_DOWN")
			} else {
				e.Id = nil
				e.Name = e.Number
				e.TypeName = "dest"
				end = append(end, fmt.Sprintf("[%s]sofia/sip/%s", e.ToStringVariables(), *e.Destination))
			}
		case "user":
			if e == nil || e.Destination == nil {
				end = append(end, "error/UNALLOCATED_NUMBER")
			} else if e.Dnd != nil && *e.Dnd {
				end = append(end, "error/USER_BUSY")
			} else {
				end = append(end, fmt.Sprintf("[%s]sofia/sip/%s@%s", e.ToStringVariables(), *e.Destination, call.DomainName()))
			}
		}
	}

	dialString += strings.Join(end, separator)

	return c.executeWithContext(ctx, "bridge", dialString)
}

func (c *Connection) Echo(ctx context.Context, delay int) (model.Response, *model.AppError) {
	if delay == 0 {
		return c.executeWithContext(ctx, "echo", "")
	} else {
		return c.executeWithContext(ctx, "delay_echo", delay)
	}
}

func (c *Connection) Export(ctx context.Context, vars []string) (model.Response, *model.AppError) {
	exp := make(map[string]interface{})
	for _, v := range vars {
		exp[fmt.Sprintf("usr_%s", v)], _ = c.Get(v)

		c.exportVariables = append(c.exportVariables, v)
	}

	if len(exp) > 0 {
		return c.Set(ctx, exp)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) Conference(ctx context.Context, name, profile, pin string, tags []string) (model.Response, *model.AppError) {
	data := fmt.Sprintf("%s_%d@%s", name, c.DomainId(), profile)
	if pin != "" {
		data += "+" + pin
	}

	if len(tags) > 0 {
		data += fmt.Sprintf("+flags{%s}", strings.Join(tags, "|"))
	}
	return c.executeWithContext(ctx, "conference", data)
}

func (c *Connection) RecordFile(ctx context.Context, name, format string, maxSec, silenceThresh, silenceHits int) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "record",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s&.%s %d %d %d", c.domainId, c.Id(), name, format,
			maxSec, silenceThresh, silenceHits))
}

func (c *Connection) RecordSession(ctx context.Context, name, format string, minSec int, stereo, bridged, followTransfer bool) (model.Response, *model.AppError) {
	// FIXME SET
	_, err := c.Set(ctx, map[string]interface{}{
		"RECORD_MIN_SEC":            minSec,
		"RECORD_STEREO":             stereo,
		"RECORD_BRIDGE_REQ":         bridged,
		"recording_follow_transfer": followTransfer,
	})

	if err != nil {
		return nil, err
	}

	return c.executeWithContext(ctx, "record_session",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s_%s&.%s", c.domainId, c.Id(), c.Id(), name, format))
}

func (c *Connection) RecordSessionStop(ctx context.Context, name, format string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "stop_record_session",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s_%s&.%s", c.domainId, c.Id(), c.Id(), name, format))
}

func (c *Connection) FlushDTMF(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "flush_dtmf", "")
}

func (c *Connection) StartDTMF(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "start_dtmf", "")
}

func (c *Connection) StopDTMF(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "stop_dtmf", "")
}

func (c *Connection) Queue(ctx context.Context, ringFile string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "wbt_queue", ringFile)
}

func (c *Connection) Intercept(ctx context.Context, id string) (model.Response, *model.AppError) {
	c.Api(fmt.Sprintf("uuid_transfer %s intercept:%s inline", c.Id(), id))
	return model.CallResponseOK, nil
}

func (c *Connection) Park(ctx context.Context, name string, in bool, lotFrom, lotTo string) (model.Response, *model.AppError) {
	var req = fmt.Sprintf("%s@%s ", c.DomainName(), name)
	if in {
		req += "in"
	} else {
		req += "out"
	}
	req += fmt.Sprintf(" %s %s", lotFrom, lotTo)
	return c.executeWithContext(ctx, "valet_park", req)
}

func (c *Connection) Redirect(ctx context.Context, uri []string) (model.Response, *model.AppError) {
	tmp := c.GetVariable("Caller-Channel-Answered-Time")

	if tmp == "0" || tmp == "" {
		tmp = "redirect"
	} else {
		tmp = "deflect"
	}
	return c.executeWithContext(ctx, tmp, strings.Join(uri, ","))
}

func (c *Connection) Playback(ctx context.Context, files []*model.PlaybackFile) (model.Response, *model.AppError) {
	fileString, ok := getFileString(c.DomainId(), files)
	if !ok {
		return nil, model.NewAppError("FS", "fs.control.playback.err", nil, "not found file", http.StatusBadRequest)
	} else {
		return c.executeWithContext(ctx, "playback", fileString)
	}
}

func (c *Connection) SetTransferAfterBridge(ctx context.Context, schemaId int) (model.Response, *model.AppError) {
	return c.Set(ctx, model.Variables{
		"transfer_to_schema_id": fmt.Sprintf("%d", schemaId),
		"transfer_after_bridge": fmt.Sprintf("%s:XML:default", c.Destination()),
	})
}

func ttsGetCodecSettings(writeRateVar string) (rate string, format string) {
	rate = "8000"
	format = "mp3"

	if writeRateVar != "" {
		if i, err := strconv.Atoi(writeRateVar); err == nil {
			if i == 8000 || i == 16000 {
				format = "wav"
				return
			} else if i >= 22050 {
				rate = "22050"
			}
		}
	}
	return
}

func (c *Connection) TTS(ctx context.Context, path string, digits *model.PlaybackDigits) (model.Response, *model.AppError) {
	var tmp string
	rate, format := ttsGetCodecSettings(c.GetVariable("write_rate"))
	if format == "mp3" {
		tmp = "shout://$${cdr_url}/sys/tts"
	} else {
		tmp = "http_cache://http://$${cdr_url}/sys/tts"
	}
	path += "&format=" + format
	if rate != "" {
		path += "&rate=" + rate
	}

	if digits != nil {
		return c.PlaybackUrlAndGetDigits(ctx, tmp+path+"."+format, digits)
	} else {
		return c.PlaybackUrl(ctx, tmp+path+"."+format)
	}
}

func (c *Connection) PlaybackUrl(ctx context.Context, url string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "playback", url)
}

func (c *Connection) PlaybackAndGetDigits(ctx context.Context, files []*model.PlaybackFile, params *model.PlaybackDigits) (model.Response, *model.AppError) {
	fileString, ok := getFileString(c.DomainId(), files)
	if !ok {
		return nil, model.NewAppError("FS", "fs.control.playback.err", nil, "not found file", http.StatusBadRequest)
	}

	return c.PlaybackUrlAndGetDigits(ctx, fileString, params)
}

func (c *Connection) PlaybackUrlAndGetDigits(ctx context.Context, fileString string, params *model.PlaybackDigits) (model.Response, *model.AppError) {
	if params.Timeout == nil {
		params.Timeout = model.NewInt(3000)
	}
	if params.Min == nil {
		params.Min = model.NewInt(1)
	}
	if params.Max == nil {
		params.Max = model.NewInt(1)
	}
	if params.Tries == nil {
		params.Tries = model.NewInt(1)
	}
	if params.Regexp == nil {
		params.Regexp = model.NewString(".*")
	}
	if params.SetVar == nil {
		params.SetVar = model.NewString("MyVar")
	}

	return c.executeWithContext(ctx, "play_and_get_digits", fmt.Sprintf("%d %d %d %d %s %s silence_stream://250 %s %s", *params.Min, *params.Max,
		*params.Tries, *params.Timeout, "#", fileString, *params.SetVar, *params.Regexp))
}

func (c *Connection) SetSounds(ctx context.Context, lang, voice string) (model.Response, *model.AppError) {
	lang = strings.ToLower(lang)
	s := strings.Split(lang, "_")

	if len(s) < 1 {
		return nil, model.NewAppError("FS", "fs.control.setSounds.err", nil, "bad lang parameter", http.StatusBadRequest)
	}

	return c.setInternal(ctx, model.Variables{
		"sound_prefix":     `/$${sounds_dir}/` + strings.Join(s, `/`) + `/` + voice,
		"default_language": s[0],
	})
}

func (c *Connection) UnSet(ctx context.Context, name string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "unset", name)
}

func (c *Connection) ScheduleHangup(ctx context.Context, sec int, cause string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "sched_hangup", fmt.Sprintf("+%d %s", sec, cause))
}

func (c *Connection) Ringback(ctx context.Context, export bool, call, hold, transfer *model.PlaybackFile) (model.Response, *model.AppError) {
	vars := model.Variables{}
	if call != nil {
		if l, ok := buildFileLink(c.domainId, call); ok {
			vars["ringback"] = l
		}
	}

	if hold != nil {
		if l, ok := buildFileLink(c.domainId, hold); ok {
			vars["hold_music"] = l
		}
	}

	if transfer != nil {
		if l, ok := buildFileLink(c.domainId, transfer); ok {
			vars["transfer_ringback"] = l
		}
	}

	return c.Set(ctx, vars)
}

func getFileString(domainId int64, files []*model.PlaybackFile) (string, bool) {
	fileString := make([]string, 0, len(files))

	for _, v := range files {
		if str, ok := buildFileLink(domainId, v); ok {
			fileString = append(fileString, str)
		}
	}
	length := len(fileString)
	if length == 1 {
		return fileString[0], true
	} else if length > 1 {
		return "file_string://" + strings.Join(fileString, "!"), true
	} else {
		return "", false
	}
}

func buildFileLink(domainId int64, file *model.PlaybackFile) (string, bool) {
	if file.Type == nil {
		return "", false
	}

	switch *file.Type {
	case "audio/mp3", "audio/mpeg":
		return fmt.Sprintf("shout://$${cdr_url}/sys/media/%d/stream?domain_id=%d&.mp3", *file.Id, domainId), true

	case "audio/wav":
		return fmt.Sprintf("http_cache://http://$${cdr_url}/sys/media/%d/stream?domain_id=%d&.wav", *file.Id, domainId), true

	case "video/mp4":
		return fmt.Sprintf("http_cache://http://$${cdr_url}/sys/media/%d/stream?domain_id=%d&.mp4", *file.Id, domainId), true

	case "tone":
		if file.Name == nil {
			return "", false
		}
		return fmt.Sprintf("tone_stream://%s", *file.Name), true

	case "silence":
		if file.Name == nil {
			return "silence_stream://-1", true
		}
		return fmt.Sprintf("silence_stream://%s", *file.Name), true

	case "local":
		if file.Name == nil {
			return "", false
		}
		return *file.Name, true
	default:
		return "", false
	}
}
