package model

import (
	"encoding/json"
	"fmt"
)

var ErrAuthSesionNotFound = NewAppError(
	"model.ErrAuthSessionNotFound",
	"app.api.account.get_auth_session.not_found",
	nil,
	"The requested authorization session does not exist or has expired.",
	404,
)

type IMUserInfo struct {
	Session IMUserSession `json:"session"`
}

func (i *IMUserInfo) Serialize() (string, *AppError) {
	if i == nil {
		return "", nil
	}

	serialized, err := json.Marshal(i)
	if err != nil {
		return "", NewAppError("Serialize", "model.im_user_info.marshaling", nil, err.Error(), 500)
	}

	return string(serialized), nil
}

type IMUserSession struct {
	Date          int64        `json:"date"`
	Name          string       `json:"name"`
	ApplicationID string       `json:"application_id"`
	Current       bool         `json:"current"`
	Device        IMUserDevice `json:"device"`
}

type IMUserDevice struct {
	IP   string      `json:"ip"`
	Push string      `json:"push"`
	App  IMUserAgent `json:"app"`
}

type IMUserAgent struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	Device  string `json:"device"`
	Mobile  bool   `json:"mobile"`
	Tablet  bool   `json:"tablet"`
	Desktop bool   `json:"desktop"`
	Bot     bool   `json:"bot"`

	//ua source string
	String string `json:"string"`
}

type IMGateType string

const (
	IMGateTypeUnspecified IMGateType = "unspecified"
	IMGateTypeFacebook    IMGateType = "facebook"
	IMGateTypeWhatsapp    IMGateType = "whatsapp"
)

func IsIMGateTypeUnspecified(t IMGateType) bool { return t == IMGateTypeUnspecified }

func IMGateTypeFromString(in string) IMGateType {
	switch in {
	case "facebook":
		return IMGateTypeFacebook
	case "whatsapp":
		return IMGateTypeWhatsapp
	default:
		return IMGateTypeUnspecified
	}
}

type IMGate struct {
	Type    IMGateType
	Payload any
}

func (g *IMGate) Facebook() (*GateFacebook, *AppError) {
	f, ok := g.Payload.(*GateFacebook)
	if !ok {
		return nil, NewAppError(
			"Facebook",
			"model.im_user_info.facebook.assert",
			nil,
			fmt.Sprintf("asserting different type from expected gate facebook: %T", g.Payload),
			400,
		)
	}

	return f, nil
}

func (g *IMGate) MarshalJSON() ([]byte, error) {
	switch p := g.Payload.(type) {
	case *GateFacebook:
		return json.Marshal(struct {
			Type     IMGateType    `json:"type"`
			Facebook *GateFacebook `json:"facebook"`
		}{
			Type:     g.Type,
			Facebook: p,
		})

	default:
		type Alias IMGate
		return json.Marshal((*Alias)(g))
	}
}

type GateFacebook struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MetaAppID string `json:"meta_app_id"`
	PageID    string `json:"page"`
	PageName  string `json:"page_name"`
}
