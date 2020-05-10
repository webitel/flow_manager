package fs

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"regexp"
	"strings"
)

const (
	HANGUP_NORMAL_TEMPORARY_FAILURE = "NORMAL_TEMPORARY_FAILURE"
	HANGUP_NO_ROUTE_DESTINATION     = "NO_ROUTE_DESTINATION"
)

var httpToShot = regexp.MustCompile(`https?`)

func (c *Connection) Answer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "answer", "")
}

func (c *Connection) PreAnswer() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "pre_answer", "")
}

func (c *Connection) RingReady() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "ring_ready", "")
}

func (c *Connection) Hangup(cause string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", cause)
}

func (c *Connection) HangupNoRoute() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NO_ROUTE_DESTINATION)
}

func (c *Connection) HangupAppErr() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "hangup", HANGUP_NORMAL_TEMPORARY_FAILURE)
}

func (c *Connection) Sleep(timeout int) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "sleep", fmt.Sprintf("%d", timeout))
}

// todo add codecs
func (c *Connection) Bridge(call model.Call, strategy string, vars map[string]string, endpoints []*model.Endpoint) (model.Response, *model.AppError) {
	var dialString, separator string

	if strategy == "failover" {
		separator = "|"
	} else if strategy != "" && strategy != "multiple" {
		separator = ":_:"
	} else {
		separator = ","
	}

	var from string

	from = fmt.Sprintf("sip_h_X-Webitel-Origin=flow,wbt_parent_id=%s,wbt_from_type=%s,wbt_from_id=%s,wbt_destination='%s'",
		call.Id(), call.From().Type, call.From().Id, call.Destination())

	from += fmt.Sprintf(",effective_caller_id_name='%s',effective_caller_id_number='%s'", call.From().Name, call.From().Number)

	dialString += "<sip_route_uri=sip:$${outbound_sip_proxy}," + from
	for key, val := range vars {
		dialString += fmt.Sprintf(",'%s'='%s'", key, val)
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

	return c.Execute(context.Background(), "bridge", dialString)
}

func (c *Connection) Echo(delay int) (model.Response, *model.AppError) {
	if delay == 0 {
		return c.Execute(context.Background(), "echo", "")
	} else {
		return c.Execute(context.Background(), "delay_echo", delay)
	}
}

func (c *Connection) Export(vars []string) (model.Response, *model.AppError) {
	c.exportVariables = vars
	return model.CallResponseOK, nil
}

func (c *Connection) Conference(name, profile, pin string, tags []string) (model.Response, *model.AppError) {
	data := fmt.Sprintf("%s_%d@%s", name, c.DomainId(), profile)
	if pin != "" {
		data += "+" + pin
	}

	if len(tags) > 0 {
		data += fmt.Sprintf("+flags{%s}", strings.Join(tags, "|"))
	}
	return c.Execute(context.Background(), "conference", data)
}

func (c *Connection) RecordFile(name, format string, maxSec, silenceThresh, silenceHits int) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "record",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s&.%s %d %d %d", c.domainId, c.Id(), name, format,
			maxSec, silenceThresh, silenceHits))
}

func (c *Connection) RecordSession(name, format string, minSec int, stereo, bridged, followTransfer bool) (model.Response, *model.AppError) {
	_, err := c.Set(map[string]interface{}{
		"RECORD_MIN_SEC":            minSec,
		"RECORD_STEREO":             stereo,
		"RECORD_BRIDGE_REQ":         bridged,
		"recording_follow_transfer": followTransfer,
	})

	if err != nil {
		return nil, err
	}

	return c.Execute(context.Background(), "record_session",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s_%s&.%s", c.domainId, c.Id(), c.Id(), name, format))
}

func (c *Connection) RecordSessionStop(name, format string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "stop_record_session",
		fmt.Sprintf("http_cache://http://$${cdr_url}/sys/recordings?domain=%d&id=%s&name=%s_%s&.%s", c.domainId, c.Id(), c.Id(), name, format))
}

func (c *Connection) FlushDTMF() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "flush_dtmf", "")
}

func (c *Connection) StartDTMF() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "start_dtmf", "")
}

func (c *Connection) StopDTMF() (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "stop_dtmf", "")
}

func (c *Connection) Park(name string, in bool, lotFrom, lotTo string) (model.Response, *model.AppError) {
	var req = fmt.Sprintf("%s@%s ", c.DomainName(), name)
	if in {
		req += "in"
	} else {
		req += "out"
	}
	req += fmt.Sprintf(" %s %s", lotFrom, lotTo)
	return c.Execute(context.Background(), "valet_park", req)
}

func (c *Connection) Redirect(uri []string) (model.Response, *model.AppError) {
	tmp := c.GetVariable("Caller-Channel-Answered-Time")

	if tmp == "0" || tmp == "" {
		tmp = "redirect"
	} else {
		tmp = "deflect"
	}
	return c.Execute(context.Background(), tmp, strings.Join(uri, ","))
}

func (c *Connection) Playback(files []*model.PlaybackFile) (model.Response, *model.AppError) {
	fileString, ok := getFileString(c.DomainId(), files)
	if !ok {
		return nil, model.NewAppError("FS", "fs.control.playback.err", nil, "not found file", http.StatusBadRequest)
	}

	return c.Execute(context.Background(), "playback", fileString)
}

func (c *Connection) PlaybackAndGetDigits(files []*model.PlaybackFile, params *model.PlaybackDigits) (model.Response, *model.AppError) {
	fileString, ok := getFileString(c.DomainId(), files)
	if !ok {
		return nil, model.NewAppError("FS", "fs.control.playback.err", nil, "not found file", http.StatusBadRequest)
	}

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

	return c.Execute(context.Background(), "play_and_get_digits", fmt.Sprintf("%d %d %d %d %s %s silence_stream://250 %s %s", *params.Min, *params.Max,
		*params.Tries, *params.Timeout, "#", fileString, *params.SetVar, *params.Regexp))
}

func (c *Connection) SetSounds(lang, voice string) (model.Response, *model.AppError) {
	lang = strings.ToLower(lang)
	s := strings.Split(lang, "_")

	if len(s) < 1 {
		return nil, model.NewAppError("FS", "fs.control.setSounds.err", nil, "bad lang parameter", http.StatusBadRequest)
	}

	return c.setInternal(model.Variables{
		"sound_prefix":     `/$${sounds_dir}/` + strings.Join(s, `/`) + `/` + voice,
		"default_language": s[0],
	})
}

func (c *Connection) UnSet(name string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "unset", name)
}

func (c *Connection) ScheduleHangup(sec int, cause string) (model.Response, *model.AppError) {
	return c.Execute(context.Background(), "sched_hangup", fmt.Sprintf("+%d %s", sec, cause))
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
	case "audio/mp3":
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
			return "", false
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
