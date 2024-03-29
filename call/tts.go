package call

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TTSArgs struct {
	Profile struct {
		Id int `json:"id"`
	} `json:"profile"`
	Key      string `json:"key"`
	Token    string `json:"token"`
	Provider string `json:"provider"`

	Language string `json:"language"`
	Voice    string `json:"voice"`
	Text     string `json:"text"`
	Region   string `json:"region"`
	//google
	SpeakingRate     string `json:"speakingRate"`
	Pitch            string `json:"pitch"`
	VolumeGainDb     string `json:"volumeGainDb"`
	EffectsProfileId string `json:"effectsProfileId"`
	KeyLocation      string `json:"keyLocation"`
	Background       *struct {
		FileUri string  `json:"url"`
		Volume  float64 `json:"volume"`
		FadeIn  int64   `json:"fadeIn"`
		FadeOut int64   `json:"fadeOut"`
	} `json:"background"`

	TextType   string                `json:"textType"`
	Terminator string                `json:"terminator"`
	GetDigits  *model.PlaybackDigits `json:"getDigits"`
	GetSpeech  *model.GetSpeech      `json:"getSpeech"`
}

func (r *Router) TTS(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv TTSArgs
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Text == "" {
		return model.CallResponseError, ErrorRequiredParameter("tts", "text")
	}

	q := ""

	switch argv.Provider {
	case "polly":
		q = "/polly?"
	case "microsoft":
		q = "/microsoft?"
	case "yandex":
		q = "/yandex?"
	case "webitel":
		q = "/webitel?"
	case "google":
		q = "/google?"

		if argv.SpeakingRate != "" {
			q += "&speakingRate=" + argv.SpeakingRate
		}
		if argv.SpeakingRate != "" {
			q += "&pitch=" + argv.Pitch
		}
		if argv.SpeakingRate != "" {
			q += "&volumeGainDb=" + argv.VolumeGainDb
		}
		if argv.SpeakingRate != "" {
			q += "&effectsProfileId=" + argv.EffectsProfileId
		}
		if argv.KeyLocation != "" {
			q += "&keyLocation=" + UrlEncoded(argv.KeyLocation)
		}

	default:
		q = "/?"
	}

	q += fmt.Sprintf("&domain_id=%d", call.DomainId())
	if argv.Profile.Id > 0 {
		q += "&profile_id=" + strconv.Itoa(argv.Profile.Id)
	}

	if argv.Key != "" {
		q += "&key=" + UrlEncoded(argv.Key)
	}

	if argv.Token != "" {
		q += "&token=" + UrlEncoded(argv.Token)
	}

	if argv.TextType != "" {
		q += "&text_type=" + argv.TextType
	}

	if argv.Region != "" {
		q += "&region=" + argv.Region
	}

	if argv.Language != "" {
		q += "&language=" + argv.Language
	}

	if argv.Voice != "" {
		q += "&voice=" + argv.Voice
	}

	if argv.Background != nil && argv.Background.FileUri != "" {
		q += "&bg_url=" + argv.Background.FileUri

		if argv.Background.Volume > 0 {
			q += fmt.Sprintf("&bg_vol=%f", argv.Background.Volume)
		}
		if argv.Background.FadeIn > 0 {
			q += fmt.Sprintf("&bg_fin=%d", argv.Background.FadeIn)
		}
		if argv.Background.FadeOut > 0 {
			q += fmt.Sprintf("&bg_fout=%d", argv.Background.FadeOut)
		}
	}

	q += "&text=" + UrlEncoded(argv.Text)

	if argv.GetSpeech != nil {
		if _, err := call.GoogleTranscribe(ctx); err != nil {
			return nil, err
		}

		if _, err := call.TTS(ctx, q, argv.GetDigits, argv.GetSpeech.Timeout); err != nil {
			return nil, err
		}

		if err := r.fm.Store.Call().SaveTranscribe(call.Id(), call.GetVariable("variable_google_transcript")); err != nil {
			return nil, err
		}
		return model.CallResponseOK, nil
	}
	if argv.Provider == "yandex" {
		return call.TTSOpus(ctx, q, argv.GetDigits, 0)
	}
	return call.TTS(ctx, q, argv.GetDigits, 0)
}

func UrlEncoded(str string) string {
	var res = url.Values{"": {str}}.Encode()

	if len(res) < 2 {
		return ""
	}

	return compatibleJSEncodeURIComponent(res[1:])
	//u, err := url.ParseRequestURI(str)
	//if err != nil {
	//	return compatibleJSEncodeURIComponent(url.QueryEscape(str))
	//}
	//return compatibleJSEncodeURIComponent(u.String())
}

func compatibleJSEncodeURIComponent(str string) string {
	resultStr := str
	resultStr = strings.Replace(resultStr, "+", "%20", -1)
	resultStr = strings.Replace(resultStr, "%21", "!", -1)
	//resultStr = strings.Replace(resultStr, "%27", "'", -1)
	resultStr = strings.Replace(resultStr, "%28", "(", -1)
	resultStr = strings.Replace(resultStr, "%29", ")", -1)
	resultStr = strings.Replace(resultStr, "%2A", "*", -1)
	return resultStr
}
