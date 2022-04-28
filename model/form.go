package model

import "encoding/json"

type Form struct {
	Id      string                 `json:"id"`
	Actions []string               `json:"actions"`
	View    map[string]interface{} `json:"view"`
}

type FormAction struct {
	Name   string
	Fields Variables
}

func (f *Form) ToJson() []byte {
	data, _ := json.Marshal(f)
	return data
}
