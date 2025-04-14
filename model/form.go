package model

import (
	"context"
	"encoding/json"
)

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
		Collapsible  bool   `json:"collapsible"`
		Component    string `json:"component"`
	} `json:"view"`
	Value interface{} `json:"value"`
}
type FormTableActionFn func(context.Context, bool, Variables) *AppError

type FormTable struct {
	Id        string                       `json:"id"`
	OutputsFn map[string]FormTableActionFn `json:"-"`

	Component string `json:"component"`

	View *struct {
		Table struct {
			Source         string `json:"source"`
			IsSystemSource bool   `json:"isSystemSource"`
			SystemSource   struct {
				Path string `json:"path"`
				Name string `json:"name"`
			} `json:"systemSource"`
			DisplayColumns []struct {
				Name  string `json:"name"`
				Field string `json:"field"`
				Width string `json:"width"`
			} `json:"displayColumns"`
		} `json:"table"`

		Filters []string `json:"filters"`
		Actions []struct {
			Field      string `json:"field"`
			Action     string `json:"action"`
			ButtonName string `json:"buttonName"`
			Color      string `json:"color"`
		} `json:"actions"`
	} `json:"view"`
}

type FormView struct {
	Component string `json:"component"`

	Label        string `json:"label,omitempty"`
	Hint         string `json:"hint,omitempty"`
	InitialValue string `json:"initialValue,omitempty"`

	CurrentTime bool `json:"currentTime,omitempty"` // wt-datetimepicker
	Options     []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"options,omitempty"` //wt-select
	Multiple      bool   `json:"multiple,omitempty"`      //wt-select
	Color         string `json:"color,omitempty"`         //form-text
	Collapsible   bool   `json:"collapsible,omitempty"`   //form-text
	EnableCopying bool   `json:"enableCopying,omitempty"` //form-text
	Output        string `json:"output,omitempty"`        // "rich-text-editor"
	Height        int    `json:"height,omitempty"`        //form-i-frame
	Variable      string `json:"variable,omitempty"`      //form-select-from-object
	Object        *struct {
		Source struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"source"`
		DisplayColumn string   `json:"displayColumn"`
		Filters       []string `json:"filters"`
	} `json:"object,omitempty"` //form-select-from-object
}

type FormComponent struct {
	Id    string      `json:"id"`
	View  *FormView   `json:"view"`
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
