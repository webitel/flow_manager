package model

import (
	"fmt"
	"strings"
)

type TTSSettings struct {
	Profile struct {
		Id int `json:"id"`
	} `json:"profile"`
	Language      string                 `json:"language"`
	Voice         string                 `json:"voice"`
	Text          string                 `json:"text"`
	TextType      string                 `json:"textType"`
	VoiceSettings map[string]interface{} `json:"voice_settings"`
	Format        string                 `json:"format"` // mp3 or wav
}

func (tts *TTSSettings) QueryParams(domainId int64) string {
	var q = fmt.Sprintf("domain_id=%d&profile_id=%d&text=%s", domainId, tts.Profile.Id,
		UrlEncoded(strings.ReplaceAll(tts.Text, "!", ".")))

	for k, v := range tts.VoiceSettings {
		q += fmt.Sprintf("&%s=%v", k, v)
	}

	if tts.TextType != "" {
		q += "&text_type=" + tts.TextType
	}

	if tts.Language != "" {
		q += "&language=" + tts.Language
	}

	if tts.Voice != "" {
		q += "&voice=" + tts.Voice
	}

	return q
}
