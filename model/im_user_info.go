package model

import (
	"encoding/json"
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
