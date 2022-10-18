package model

import "encoding/json"

type FormAction struct {
	Name   string
	Fields Variables
}

type FormFile struct {
	Id   string `json:"id"`
	View struct {
		InitialValue []File `json:"initialValue"`
		Label        string `json:"label"`
		Hint         string `json:"hint"`
		Readonly     bool   `json:"readonly"`
	} `json:"view"`
	Value interface{} `json:"value"`
}

type FormComponent struct {
	Id    string      `json:"id"`
	View  *JsonView   `json:"view"`
	Value interface{} `json:"value"`
}

type FormActionElem struct {
	Id   string    `json:"id"`
	View *JsonView `json:"view"`
}

type FormElem struct {
	Id      string            `json:"id"`
	Title   string            `json:"title"`
	Actions []*FormActionElem `json:"actions"`
	Body    []interface{}     `json:"body"`
}

func (f *FormElem) ToJson() []byte {
	data, _ := json.Marshal(f)
	return data
}
