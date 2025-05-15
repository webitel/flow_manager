package model

type FormFile struct {
	Id   string `json:"id"`
	View struct {
		InitialValue []File `json:"initialValue"`
		Label        string `json:"label"`
		Hint         string `json:"hint"`
		Readonly     bool   `json:"readonly"`
		Collapsible  bool   `json:"collapsible"`
		Component    string `json:"component"`
	} `json:"view"`
	Value interface{} `json:"value"`
}
