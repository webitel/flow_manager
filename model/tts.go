package model

import (
	"fmt"
	"sort"
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

	//google
	SpeakingRate     string `json:"speakingRate"`
	Pitch            string `json:"pitch"`
	VolumeGainDb     string `json:"volumeGainDb"`
	EffectsProfileId string `json:"effectsProfileId"`
	KeyLocation      string `json:"keyLocation"` // todo deprecated
	Background       *struct {
		FileUri string  `json:"url"`
		Volume  float64 `json:"volume"`
		FadeIn  int64   `json:"fadeIn"`
		FadeOut int64   `json:"fadeOut"`
	} `json:"background"`
}

func (tts *TTSSettings) QueryParams(domainId int64) string {
	var q = fmt.Sprintf("domain_id=%d&profile_id=%d&text=%s", domainId, tts.Profile.Id,
		UrlEncoded(strings.ReplaceAll(tts.Text, "!", ".")))

	concatenedMaps, keys := getSortedKeys(tts.VoiceSettings)
	for _, k := range keys {
		q += fmt.Sprintf("&%s=%v", k, concatenedMaps[k])
	}

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

	if tts.Background != nil && tts.Background.FileUri != "" {
		q += "&bg_url=" + tts.Background.FileUri

		if tts.Background.Volume > 0 {
			q += fmt.Sprintf("&bg_vol=%f", tts.Background.Volume)
		}
		if tts.Background.FadeIn > 0 {
			q += fmt.Sprintf("&bg_fin=%d", tts.Background.FadeIn)
		}
		if tts.Background.FadeOut > 0 {
			q += fmt.Sprintf("&bg_fout=%d", tts.Background.FadeOut)
		}
	}

	if tts.SpeakingRate != "" {
		q += "&speakingRate=" + tts.SpeakingRate
	}
	if tts.SpeakingRate != "" {
		q += "&pitch=" + tts.Pitch
	}
	if tts.SpeakingRate != "" {
		q += "&volumeGainDb=" + tts.VolumeGainDb
	}
	if tts.SpeakingRate != "" {
		q += "&effectsProfileId=" + tts.EffectsProfileId
	}
	if tts.KeyLocation != "" {
		q += "&keyLocation=" + UrlEncoded(tts.KeyLocation)
	}

	return q
}

func getSortedKeys(maps ...map[string]interface{}) (map[string]interface{}, []string) {
	var keys []string
	resultMap := map[string]interface{}{}

	for _, currMap := range maps {
		for k, v := range currMap {
			resultMap[k] = v
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return resultMap, keys

}
