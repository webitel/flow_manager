package fs

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/h2non/filetype"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

const (
	HANGUP_NORMAL_TEMPORARY_FAILURE = "NORMAL_TEMPORARY_FAILURE"
	HANGUP_NO_ROUTE_DESTINATION     = "NO_ROUTE_DESTINATION"
)

const (
	recordSessionFSApp     string = "wbt_record_session"
	stopRecordSessionFSApp string = "stop_record_session"
)

var fixNamePattern = regexp.MustCompile(`'|"|,`)

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

func (c *Connection) BackgroundPlayback(ctx context.Context, file *model.PlaybackFile, name string, volumeReduction int) (model.Response, *model.AppError) {
	s, ok := c.buildFileLink(file)
	if !ok {
		return model.CallResponseError, model.NewAppError("FS", "fs.control.backgroundPlayback", nil, "bad file", http.StatusInternalServerError)
	}

	if len(name) > 10 {
		name = name[0:10]
	}

	res, err := c.executeWithContext(ctx, "wbt_background", fmt.Sprintf("start %s %d %s", s, volumeReduction, name))
	if err == nil {
		c.Lock()
		c.playBackground++
		c.Unlock()
	}
	return res, err
}

func (c *Connection) BackgroundPlaybackStop(ctx context.Context, name string) (model.Response, *model.AppError) {
	if len(name) > 10 {
		name = name[0:10]
	}

	res, err := c.executeWithContext(ctx, "wbt_background", fmt.Sprintf("stop %s", name))
	if err == nil {
		c.Lock()
		c.playBackground--
		c.Unlock()
	}
	return res, err
}

// FIXME GLOBAL VARS
func (c *Connection) Bridge(ctx context.Context, call model.Call, strategy string, vars map[string]string,
	endpoints []*model.Endpoint, codecs []string, hook chan struct{}, pickup string,
) (model.Response, *model.AppError) {
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
	// origination_callee_id_name

	from = fmt.Sprintf("sip_copy_custom_headers=false,sip_h_X-Webitel-Domain-Id=%d,sip_h_X-Webitel-Origin=flow,wbt_parent_id=%s,wbt_from_type=%s,wbt_from_id=%s,wbt_destination='%s'"+
		",wbt_from_number='%s',wbt_from_name='%s'",
		call.DomainId(), call.Id(), call.From().Type, call.From().Id, call.Destination(), call.From().Number, fixName(call.From().Name))

	from += fmt.Sprintf(",effective_caller_id_name='%s',effective_caller_id_number='%s'", fixName(call.From().Name), call.From().Number)

	dialString += "<sip_route_uri=sip:$${outbound_sip_proxy}," + from
	for key, val := range vars {
		dialString += fmt.Sprintf(",'%s'='%s'", key, val)
	}

	if c.transferFromId != "" {
		dialString += ",wbt_transfer_from=" + c.transferFromId
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

	if c.meetingId != "" {
		dialString += fmt.Sprintf(",sip_h_X-Webitel-Meeting='%s'", c.meetingId)
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

				tmp := c.GetVariable("variable_sip_to_display")
				if tmp != "" && c.direction == model.CallDirectionOutbound {
					e.Name = model.NewString(tmp)
				} else if e.Name == nil {
					e.Name = e.Number
				}

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

	if pickup != "" {
		dialString += fmt.Sprintf("pickup/%s:_:", c.PickupHash(pickup))
	}
	dialString += strings.Join(end, separator)

	if hook != nil {
		// todo
		c.Lock()
		c.setHookBridged(hook)
		c.Unlock()
	}

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
	exp := make(map[string]any)
	for _, v := range vars {
		if v == "" {
			continue
		}
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
	if c.resample != 0 && !c.IsSetResample() {
		c.Set(ctx, model.Variables{
			"record_sample_rate": c.resample,
		})
	}

	link, err := c.sysurl.RecordingUpload(c.domainId, c.Id(), name+"."+format, format, nil)
	if err != nil {
		return model.CallResponseError, sysURLError(err)
	}

	return c.executeWithContext(ctx, "record",
		fmt.Sprintf("http_cache://http://$${cdr_url}%s %d %d %d", link, maxSec, silenceThresh, silenceHits))
}

type SpeechAiMessage struct {
	Message string `json:"message"`
	Sender  string `json:"sender"`
}

func (c *Connection) SendFileToAi(ctx context.Context, aiUrl string, m map[string]string, format string, maxSec, silenceThresh, silenceHits int) (model.Response, *model.AppError) {
	if c.resample != 0 && !c.IsSetResample() {
		c.Set(ctx, model.Variables{
			"record_sample_rate": c.resample,
		})
	}
	msg := c.SpeechMessages(20)

	history := make([]SpeechAiMessage, 0, len(msg))
	for _, ms := range msg {
		if ms.Question != "" {
			history = append(history, SpeechAiMessage{
				Message: ms.Question,
				Sender:  "human",
			})
		}
		if ms.Answer != "" {
			history = append(history, SpeechAiMessage{
				Message: ms.Answer,
				Sender:  "ai",
			})
		}
	}

	historyJson, _ := json.Marshal(history)

	id := model.NewId()
	params := url.Values{}
	params.Set("domain", strconv.FormatInt(c.domainId, 10))
	params.Set("url", aiUrl)
	params.Set("chat_history", b64.URLEncoding.EncodeToString(historyJson))
	params.Set("id", id)
	for k, v := range m {
		model.AddFirstValue(params, k, b64.URLEncoding.EncodeToString([]byte(v)))
	}

	link, err := c.sysurl.AiUpload(id, params, format)
	if err != nil {
		return model.CallResponseError, sysURLError(err)
	}

	recUrl := fmt.Sprintf("http_cache://http://$${cdr_url}%s %d %d %d", link, maxSec, silenceThresh, silenceHits)

	r, e := c.executeWithContext(ctx, "record", recUrl)

	if e != nil {
		return r, e
	}

	cdrUrl, _ := c.Get("Application-Data")
	if len(cdrUrl) < 15 {
		return model.CallResponseError, model.NewAppError("FS", "fs.control.ai.cdr_url", nil, "not found Application-Data url", http.StatusInternalServerError)
	}
	i := strings.Index(cdrUrl[13:], "/sys/recordings/ai")
	if i > 1 {
		cdrUrl = cdrUrl[13 : i+13]
	}

	metadataLink, err := c.sysurl.AiMetadata(id)
	if err != nil {
		return model.CallResponseError, sysURLError(err)
	}

	res, err := http.DefaultClient.Get(cdrUrl + metadataLink)
	if err != nil {
		return model.CallResponseError, model.NewAppError("FS", "fs.control.ai.err", nil, err.Error(), http.StatusInternalServerError)
	}

	data, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return model.CallResponseError, model.NewAppError("FS", "fs.control.ai.err", nil, string(data), http.StatusInternalServerError)
	}

	var vars model.Variables
	err = json.Unmarshal(data, &vars)
	if err != nil {
		return model.CallResponseError, model.NewAppError("FS", "fs.control.ai.parse", nil, err.Error(), http.StatusInternalServerError)
	}
	r, e = c.Set(ctx, vars)
	if e != nil {
		return r, e
	}
	sp := model.SpeechMessage{}
	var tmp any
	tmp, _ = vars["ai_human"]
	sp.Question = fmt.Sprintf("%v", tmp)
	tmp, _ = vars["ai_answer"]
	sp.Answer = fmt.Sprintf("%v", tmp)
	c.PushSpeechMessage(sp)

	speechLink, err := c.sysurl.AiSpeech(id, "wav")
	if err != nil {
		return model.CallResponseError, sysURLError(err)
	}

	return c.executeWithContext(ctx, "playback", "http_cache://http://$${cdr_url}"+speechLink)
}

func (c *Connection) buildRecordSessionPath(args *model.RecordSessionArgs) (string, *model.AppError) {
	link, err := c.sysurl.RecordingUpload(c.DomainId(), c.Id(), args.Name+"."+args.Format, args.Format, nil)
	if err != nil {
		return "", sysURLError(err)
	}

	return "http_cache://http://$${cdr_url}" + link, nil
}

func (c *Connection) RecordSession(ctx context.Context, args *model.RecordSessionArgs) (model.Response, *model.AppError) {
	recordPath, appErr := c.buildRecordSessionPath(args)
	if appErr != nil {
		return nil, appErr
	}

	vars := map[string]any{
		"RECORD_MIN_SEC":            args.MinSec,
		"RECORD_STEREO":             args.Stereo,
		"RECORD_BRIDGE_REQ":         args.Bridged,
		"media_bug_answer_req":      args.Bridged,
		"recording_follow_transfer": args.FollowTransfer,
		args.PathVar:                recordPath,
	}

	if c.resample != 0 && !c.IsSetResample() {
		vars["record_sample_rate"] = c.resample
	}

	if _, err := c.setInternal(ctx, vars); err != nil {
		return nil, err
	}

	path, exists := c.Get(args.PathVar)
	if !exists || strings.TrimSpace(path) == "" {
		return nil, model.NewAppError("RecordSession", "fs.control.record_session.get_path", nil, "finding recording path in FS variables", http.StatusNotFound)
	}

	if args.Force {
		path += " {force=true}"
	}

	return c.executeWithContext(ctx, recordSessionFSApp, path)
}

func (c *Connection) RecordSessionStop(ctx context.Context, pathVar string) (model.Response, *model.AppError) {
	path, exists := c.Get(pathVar)
	if !exists || strings.TrimSpace(path) == "" {
		return nil, model.NewAppError("RecordSessionStop", "fs.control.record_session.get_path", nil, "finding recording path in FS variables", http.StatusNotFound)
	}

	return c.executeWithContext(ctx, stopRecordSessionFSApp, path)
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
	req := fmt.Sprintf("%s@%s ", c.DomainName(), name)
	if in {
		req += "in"
	} else {
		req += "out"
	}
	req += fmt.Sprintf(" %s %s", lotFrom, lotTo)
	return c.executeWithContext(ctx, "valet_park", req)
}

func (c *Connection) Push(ctx context.Context, name, tag string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "push", fmt.Sprintf("%s=%s", name, tag))
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
	fileString, ok := c.getFileString(files)
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

func ttsGetCodecSettings(writeRateVar string) (rate, format string) {
	rate = "8000"
	format = "mp3"

	if writeRateVar != "" {
		if i, err := strconv.Atoi(writeRateVar); err == nil {
			if i == 8000 || i == 16000 {
				format = "wav"
				return rate, format
			} else if i >= 22050 {
				rate = "22050"
			}
		}
	}
	return rate, format
}

func (c *Connection) PushSpeechMessage(msg model.SpeechMessage) {
	c.Lock()
	c.speechMessages = append(c.speechMessages, msg)
	c.Unlock()
}

func (c *Connection) SpeechMessages(limit int) []model.SpeechMessage {
	c.Lock()
	cnt := len(c.speechMessages)
	c.Unlock()
	if cnt == 0 {
		return nil
	}

	if cnt < limit {
		limit = cnt
	}
	res := make([]model.SpeechMessage, 0, limit)
	c.Lock()

	for _, v := range c.speechMessages[(cnt - limit):] {
		res = append(res, v)
	}
	c.Unlock()
	return res
}

func (c *Connection) TTS(ctx context.Context, provider string, params url.Values, tts model.TTSSettings, digits *model.PlaybackDigits, timeout int) (model.Response, *model.AppError) {
	link, format, err := c.ttsLink(provider, params, &tts)
	if err != nil {
		return model.CallResponseError, err
	}

	fs := []string{
		c.ttsProtocol(&tts, format, true) + link,
		c.ttsProtocol(&tts, format, false) + link,
	}
	if timeout > 0 {
		fs = append(fs, fmt.Sprintf("silence_stream://%d", timeout))
	}

	url := "file_string://" + strings.Join(fs, "!")

	if digits != nil {
		return c.PlaybackUrlAndGetDigits(ctx, url, digits)
	} else {
		return c.PlaybackUrl(ctx, url)
	}
}

func (c *Connection) TTSOpus(ctx context.Context, provider string, params url.Values, digits *model.PlaybackDigits, timeout int) (model.Response, *model.AppError) {
	q := url.Values{}
	for k, v := range params {
		q[k] = v
	}
	model.AddFirstValue(q, "format", "opus")

	link, err := c.sysurl.TTS(provider, q, "opus")
	if err != nil {
		return model.CallResponseError, sysURLError(err)
	}

	fileUrl := "http_cache://http://$${cdr_url}" + link

	if timeout > 0 {
		fileUrl = fmt.Sprintf("file_string://%s!silence_stream://%d", fileUrl, timeout)
	}

	if digits != nil {
		return c.PlaybackUrlAndGetDigits(ctx, fileUrl, digits)
	} else {
		return c.PlaybackUrl(ctx, fileUrl)
	}
}

func (c *Connection) PlaybackUrl(ctx context.Context, url string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "playback", url)
}

func (c *Connection) RefreshVars(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "dump", "")
}

func (c *Connection) PlaybackAndGetDigits(ctx context.Context, files []*model.PlaybackFile, params *model.PlaybackDigits) (model.Response, *model.AppError) {
	fileString, ok := c.getFileString(files)
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

	if params.Terminators == "" {
		params.Terminators = "#"
	}

	dgTimeout := ""
	if params.DigitTimeout != nil {
		dgTimeout = " " + strconv.Itoa(*params.DigitTimeout)
	}

	return c.executeWithContext(ctx, "play_and_get_digits", fmt.Sprintf("%d %d %d %d %s %s silence_stream://250 %s %s%s", *params.Min, *params.Max,
		*params.Tries, *params.Timeout, params.Terminators, fileString, *params.SetVar, *params.Regexp, dgTimeout))
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
		if l, ok := c.buildFileLink(call); ok {
			vars["ringback"] = l
		}
	}

	if hold != nil {
		if l, ok := c.buildFileLink(hold); ok {
			vars["hold_music"] = l
		}
	}

	if transfer != nil {
		if l, ok := c.buildFileLink(transfer); ok {
			vars["transfer_ringback"] = l
		}
	}

	if export {
		vars["bridge_export_vars"] = "hold_music,ringback,transfer_ringback"
	}

	return c.Set(ctx, vars)
}

func (c *Connection) Amd(ctx context.Context, params model.AmdParameters) (model.Response, *model.AppError) {
	return model.CallResponseOK, nil
}

func (c *Connection) Cv(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "cv_bug", "start zidx=0 debug=0 neighbors=1 skip=1 abs=9 scaleto=wh allclear png=igor.png ticker=#cccccc:#54d41e:/usr/share/fonts/truetype/dejavu/DejaVuSerifCondensed.ttf:4%:1:igor:'Hello test test' allflat")
	// return c.executeWithContext(ctx, "cv_bug", "start zidx=1 debug=1 neighbors=1 skip=1 abs=4 scaleto=wh allclear")
}

func (c *Connection) GoogleTranscribe(ctx context.Context, config *model.GetSpeech) (model.Response, *model.AppError) {
	if config.Lang == "" {
		if config.Lang, _ = c.get("GOOGLE_SPEECH_LANG"); config.Lang == "" {
			config.Lang = "en-US"
		}
	}

	if config.SampleRate == 0 {
		config.SampleRate = 8000
	}

	switch config.Version {
	case "v2":
		boolString := func(b bool) string {
			if b {
				return " true"
			} else {
				return " false"
			}
		}
		vars := map[string]any{
			"GOOGLE_SPEECH_CLOUD_SERVICES_VERSION": "v2",
			"GOOGLE_SPEECH_RECOGNIZER_PARENT":      config.Recognizer,
			"GOOGLE_SPEECH_TO_TEXT_URI":            config.Uri,
			"RECOGNIZING_VAD_TIMEOUT":              fmt.Sprintf("%d", config.VadTimeout),
		}
		if config.DisableBreakFinal {
			vars["GOOGLE_DISABLE_BREAK"] = "true"
		} else {
			vars["GOOGLE_DISABLE_BREAK"] = "false"
		}

		vars["GOOGLE_MIN_RESPONSE_WORDS"] = fmt.Sprintf("%d", config.MinWords)
		vars["GOOGLE_MAX_RESPONSE_WORDS"] = fmt.Sprintf("%d", config.MaxWords)

		if config.BreakFinalOnTimeout {
			vars["GOOGLE_BREAK_ON_TIMEOUT"] = "true"
		} else {
			vars["GOOGLE_BREAK_ON_TIMEOUT"] = "false"
		}

		vars["GOOGLE_BREAK_STABILITY"] = fmt.Sprintf("%v", config.BreakStability)

		if len(config.AlternativeLang) != 0 {
			vars["GOOGLE_SPEECH_ALTERNATIVE_LANGUAGE_CODES"] = strings.Join(config.AlternativeLang, ",")
		}
		c.Set(ctx, vars)
		str := "uuid_google_transcribe2 " + c.id + " start " + config.Lang + boolString(config.Interim) + boolString(config.SingleUtterance) +
			boolString(config.SeparateRecognition) + " " + strconv.Itoa(config.MaxAlternatives) + boolString(config.ProfanityFilter) +
			boolString(config.WordTime) + boolString(config.Punctuation) + " " + strconv.Itoa(config.SampleRate) + " " + config.Model + " " +
			boolString(config.Enhanced) + " " + config.Hints

		if _, err := c.Api(str); err != nil {
			return nil, model.NewAppError("FS", "fs.control.GoogleTranscribe.err", nil, fmt.Sprintf("%s", err.Error()), http.StatusBadRequest)
		}
	default:
		if _, err := c.Api(fmt.Sprintf("uuid_google_transcribe %s start %s interim", c.id, config.Lang)); err != nil {
			return nil, model.NewAppError("FS", "fs.control.GoogleTranscribe.err", nil, fmt.Sprintf("%s", err.Error()), http.StatusBadRequest)
		}
	}

	return model.CallResponseOK, nil
}

func (c *Connection) GoogleTranscribeStop(ctx context.Context) (model.Response, *model.AppError) {
	if _, err := c.Api(fmt.Sprintf("uuid_google_transcribe %s stop", c.id)); err != nil {
		return nil, model.NewAppError("FS", "fs.control.GoogleTranscribeStop.err", nil, fmt.Sprintf("%s", err.Error()), http.StatusBadRequest)
	}

	return model.CallResponseOK, nil
}

/*
StartRecognize(ctx context.Context, connection, dialogId string, rate int) error
StopRecognize(ctx context.Context, dialogId string)
*/
func (c *Connection) StartRecognize(ctx context.Context, connection, dialogId string, rate, vadTimeout int) (model.Response, *model.AppError) {
	args := fmt.Sprintf("uuid_wbt_stt %s start %s %d %s %d", c.id, connection, rate, dialogId, vadTimeout)

	_, err := c.Api(args)
	if err != nil {
		return nil, model.NewAppError("FS", "fs.control.stt.start", nil, fmt.Sprintf("%s", err.Error()), http.StatusBadRequest)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) StopRecognize(ctx context.Context) (model.Response, *model.AppError) {
	args := fmt.Sprintf("uuid_wbt_stt %s stop", c.id)

	_, err := c.Api(args)
	if err != nil {
		return nil, model.NewAppError("FS", "fs.control.stt.stop", nil, fmt.Sprintf("%s", err.Error()), http.StatusBadRequest)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) UpdateCid(ctx context.Context, name, number, destination *string) (res model.Response, err *model.AppError) {
	if name != nil {
		if res, err = c.executeWithContext(ctx, "set_profile_var", fmt.Sprintf("caller_id_name=%s", *name)); err != nil {
			return nil, err
		}

		c.from.Name = *name
	}

	if number != nil {
		if res, err = c.executeWithContext(ctx, "set_profile_var", fmt.Sprintf("caller_id_number=%s", *number)); err != nil {
			return nil, err
		}

		c.from.Number = *number
	}

	if destination != nil {
		if res, err = c.executeWithContext(ctx, "set_profile_var", fmt.Sprintf("destination=%s", *destination)); err != nil {
			return nil, err
		}

		c.destination = *destination
	}

	return res, err
}

func (c *Connection) AmdML(ctx context.Context, params model.AmdMLParameters) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "wbt_amd", strings.Join(params.Tags, ","))
}

func (c *Connection) Pickup(ctx context.Context, name string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "pickup", fmt.Sprintf("%s@%d", name, c.domainId))
}

// TODO
func (c *Connection) Say(ctx context.Context, val string) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "say", val)
}

// PickupHash todo
func (c *Connection) PickupHash(name string) string {
	return fmt.Sprintf("%s@%d", name, c.domainId)
}

func (c *Connection) Bot(ctx context.Context, conn string, rate int, id string, vars map[string]string) (model.Response, *model.AppError) {
	args := fmt.Sprintf("%s %d %s", conn, rate, id)

	if vars != nil {
		// TODO JSON ?
	}

	return c.executeWithContext(ctx, "wbt_voice_bot", args)
}

func (c *Connection) Update(ctx context.Context) (model.Response, *model.AppError) {
	return c.executeWithContext(ctx, "wbt_update_call", "")
}

func (c *Connection) exportCallVariables(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	var err *model.AppError
	for k, v := range vars {
		if _, err = c.executeWithContext(ctx, "export", fmt.Sprintf("%s=%s", k, v)); err != nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}

func (c *Connection) getFileString(files []*model.PlaybackFile) (string, bool) {
	fileString := make([]string, 0, len(files))

	for _, v := range files {
		if v.Type != nil && *v.Type == "tts" && v.TTS != nil {
			link, format, err := c.ttsLink("", nil, v.TTS)
			if err != nil {
				c.log.Error("build tts link", wlog.Err(err))
				continue
			}
			fileString = append(fileString,
				c.ttsProtocol(v.TTS, format, true)+link,
				c.ttsProtocol(v.TTS, format, false)+link,
			)
			continue
		}
		if str, ok := c.buildFileLink(v); ok {
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

func (c *Connection) IsPlayBackground() bool {
	c.RLock()
	bg := c.playBackground > 0
	c.RUnlock()
	return bg
}

func (c *Connection) buildFileLink(file *model.PlaybackFile) (string, bool) {
	if file == nil || file.Type == nil {
		return "", false
	}

	switch *file.Type {
	case "audio/mp3", "audio/mpeg":
		if file.Id == nil {
			return "", false
		}
		return c.mediaStreamLink("shout://$${cdr_url}", *file.Id, "mp3")

	case "audio/wav", "audio/ogg", "audio/x-wav":
		if file.Id == nil {
			return "", false
		}
		return c.mediaStreamLink("http_cache://http://$${cdr_url}", *file.Id, "wav")

	case "video/mp4":
		if file.Id == nil {
			return "", false
		}
		return c.mediaStreamLink("http_cache://http://$${cdr_url}", *file.Id, "mp4")

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

	case "http_audio":
		var args model.HttpFileArgs
		if file.Args == nil {
			return "", false
		}
		bytes, err := json.Marshal(file.Args)
		if err != nil {
			return "", false
		}
		err = json.Unmarshal(bytes, &args)
		if err != nil {
			return "", false
		}
		var protocol string
		tp := filetype.GetType(args.FileType)
		switch tp.MIME.Value {
		case "audio/wav", "audio/x-wav":
			protocol = "http_cache://https://$${cdr_url}"
		case "audio/mp3", "audio/mpeg":
			protocol = "shout://$${cdr_url}"
		default:
			return "", false
		}
		link, err := c.sysurl.RedirectPlayback(args.Url, args.Method, args.Headers, "")
		if err != nil {
			c.log.Error("build redirect playback link", wlog.Err(err))
			return "", false
		}
		return protocol + link, true

	case "local":
		if file.Name == nil {
			return "", false
		}
		return *file.Name, true

	case "tts":
		if file.TTS == nil {
			return "", false
		}
		link, format, err := c.ttsLink("", nil, file.TTS)
		if err != nil {
			c.log.Error("build tts link", wlog.Err(err))
			return "", false
		}
		return c.ttsProtocol(file.TTS, format, false) + link, true
	default:
		return "", false
	}
}

func (c *Connection) ttsLink(provider string, params url.Values, tts *model.TTSSettings) (string, string, *model.AppError) {
	q := url.Values{}
	for k, v := range params {
		q[k] = v
	}
	tts.QueryValues(c.domainId, q)

	rate, format := ttsGetCodecSettings(c.GetVariable("variable_write_rate"))
	if tts.Format == "ulaw" { // todo 11lab test
		format = tts.Format
	}

	if !tts.Static {
		model.AddFirstValue(q, "r", c.id[:5])
	}

	model.AddFirstValue(q, "format", format)
	if rate != "" {
		model.AddFirstValue(q, "rate", rate)
	}

	link, err := c.sysurl.TTS(provider, q, format)
	if err != nil {
		return "", "", sysURLError(err)
	}

	return link, format, nil
}

func (c *Connection) ttsProtocol(tts *model.TTSSettings, format string, prepare bool) string {
	var protocol string

	if prepare {
		protocol = "{call_id=" + c.id + ",id=" + model.NewId()[:8]
		if c.IsPlayBackground() && !tts.Static {
			protocol += ",skip_cache=true"
		}
		protocol += "}"
		protocol += "wbt_prepare://http://$${cdr_url}"
	} else if format == "mp3" {
		protocol = "shout://$${cdr_url}"
	} else {
		if c.IsPlayBackground() && !tts.Static {
			protocol = "{refresh=true}"
		}
		protocol += "http_cache://http://$${cdr_url}"
	}

	return protocol
}

func (c *Connection) mediaStreamLink(protocol string, fileId int, format string) (string, bool) {
	link, err := c.sysurl.MediaStream(c.domainId, int64(fileId), format)
	if err != nil {
		c.log.Error("build media stream link", wlog.Err(err))
		return "", false
	}

	return protocol + link, true
}

func sysURLError(err error) *model.AppError {
	return model.NewAppError("FS", "fs.control.sys_url", nil, err.Error(), http.StatusInternalServerError)
}

func fixName(n string) string {
	return fixNamePattern.ReplaceAllString(n, "")
}
