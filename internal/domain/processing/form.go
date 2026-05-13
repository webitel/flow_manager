package processing

// moved from model/form.go — see model/form.go for re-export alias

import "github.com/webitel/flow_manager/internal/domain/files"

// FormFile represents a file-type field in a processing form.
type FormFile struct {
	Id   string `json:"id"`
	View struct {
		InitialValue []files.File `json:"initialValue"`
		Label        string       `json:"label"`
		Hint         string       `json:"hint"`
		Readonly     bool         `json:"readonly"`
		Collapsible  bool         `json:"collapsible"`
		Component    string       `json:"component"`
		Uuid         string       `json:"uuid"`
		EntityId     string       `json:"entityId"`
		Channel      string       `json:"channel"`
	} `json:"view"`
	Value interface{} `json:"value"`
}
