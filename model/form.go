package model

import "encoding/json"

type FormAction struct {
	Name   string
	Fields Variables
}

type FormComponent struct {
	Name string    `json:"name"`
	View *JsonView `json:"view"`
}

type FormActionElem struct {
	Name string    `json:"name"`
	View *JsonView `json:"view"`
}

type FormElem struct {
	Name    string            `json:"name"`
	Title   string            `json:"title"`
	Actions []*FormActionElem `json:"actions"`
	Body    []FormComponent   `json:"body"`
}

func (f *FormElem) ToJson() []byte {
	data, _ := json.Marshal(f)
	return data
}
