package model

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
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
	Format        string                 `json:"format"` // mp3 or wav or ulaw
	Static        bool                   `json:"static"`

	//google
	SpeakingRate     string `json:"speakingRate"`
	Pitch            string `json:"pitch"`
	VolumeGainDb     string `json:"volumeGainDb"`
	EffectsProfileId string `json:"effectsProfileId"`
	Background       *struct {
		FileUri string  `json:"url"`
		Volume  float64 `json:"volume"`
		FadeIn  int64   `json:"fadeIn"`
		FadeOut int64   `json:"fadeOut"`
	} `json:"background"`
}

func (tts *TTSSettings) QueryValues(domainId int64, q url.Values) {
	AddFirstValue(q, "domain_id", strconv.FormatInt(domainId, 10))
	AddFirstValue(q, "profile_id", strconv.Itoa(tts.Profile.Id))
	AddFirstValue(q, "text", strings.ReplaceAll(tts.Text, "!", "."))

	concatenedMaps, keys := getSortedKeys(tts.VoiceSettings)
	for _, k := range keys {
		AddFirstValue(q, k, fmt.Sprintf("%v", concatenedMaps[k]))
	}

	if tts.TextType != "" {
		AddFirstValue(q, "text_type", tts.TextType)
	}

	if tts.Language != "" {
		AddFirstValue(q, "language", tts.Language)
	}

	if tts.Voice != "" {
		AddFirstValue(q, "voice", tts.Voice)
	}

	if tts.Background != nil && tts.Background.FileUri != "" {
		AddFirstValue(q, "bg_url", tts.Background.FileUri)

		if tts.Background.Volume > 0 {
			AddFirstValue(q, "bg_vol", fmt.Sprintf("%f", tts.Background.Volume))
		}
		if tts.Background.FadeIn > 0 {
			AddFirstValue(q, "bg_fin", strconv.FormatInt(tts.Background.FadeIn, 10))
		}
		if tts.Background.FadeOut > 0 {
			AddFirstValue(q, "bg_fout", strconv.FormatInt(tts.Background.FadeOut, 10))
		}
	}

	if tts.SpeakingRate != "" {
		AddFirstValue(q, "speakingRate", tts.SpeakingRate)
	}
	if tts.Pitch != "" {
		AddFirstValue(q, "pitch", tts.Pitch)
	}
	if tts.VolumeGainDb != "" {
		AddFirstValue(q, "volumeGainDb", tts.VolumeGainDb)
	}
	if tts.EffectsProfileId != "" {
		AddFirstValue(q, "effectsProfileId", tts.EffectsProfileId)
	}
}

func AddFirstValue(q url.Values, key, value string) {
	if _, ok := q[key]; !ok {
		q.Set(key, value)
	}
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
