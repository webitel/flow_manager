package call

// moved from model/tts.go — see model/tts.go for re-export alias

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// TTSSettings holds text-to-speech synthesis parameters.
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

	// google
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

// QueryParams builds a URL query string for TTS synthesis requests.
func (tts *TTSSettings) QueryParams(domainId int64) string {
	var q = fmt.Sprintf("domain_id=%d&profile_id=%d&text=%s", domainId, tts.Profile.Id,
		urlEncoded(strings.ReplaceAll(tts.Text, "!", ".")))

	concatenedMaps, keys := getSortedKeys(tts.VoiceSettings)
	for _, k := range keys {
		q += fmt.Sprintf("&%s=%v", k, concatenedMaps[k])
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
		q += "&keyLocation=" + urlEncoded(tts.KeyLocation)
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

func urlEncoded(str string) string {
	res := url.Values{"": {str}}.Encode()
	if len(res) < 2 {
		return ""
	}
	return compatibleJSEncodeURIComponent(res[1:])
}

func compatibleJSEncodeURIComponent(str string) string {
	resultStr := str
	resultStr = strings.Replace(resultStr, "+", "%20", -1)
	resultStr = strings.Replace(resultStr, "%21", "!", -1)
	resultStr = strings.Replace(resultStr, "%28", "(", -1)
	resultStr = strings.Replace(resultStr, "%29", ")", -1)
	resultStr = strings.Replace(resultStr, "%2A", "*", -1)
	return resultStr
}
