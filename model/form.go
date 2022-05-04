package model

import "encoding/json"

type FormAction struct {
	Name   string
	Fields Variables
}

type FormComponent struct {
	Id   string    `json:"id"`
	View *JsonView `json:"view"`
}

type FormActionElem struct {
	Id   string    `json:"id"`
	View *JsonView `json:"view"`
}

type FormElem struct {
	Id      string            `json:"id"`
	Title   string            `json:"title"`
	Actions []*FormActionElem `json:"actions"`
	Body    []FormComponent   `json:"body"`
}

func (f *FormElem) ToJson() []byte {
	data, _ := json.Marshal(f)
	return data
}
