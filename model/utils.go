package model

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"strconv"
)

type AppError struct {
	Id            string `json:"id"`
	Message       string `json:"status"`               // Message to be display to the end user without debugging information
	DetailedError string `json:"detail"`               // Internal error string to help the developer
	RequestId     string `json:"request_id,omitempty"` // The RequestId that's also set in the header
	StatusCode    int    `json:"code,omitempty"`       // The http status code
	Where         string `json:"-"`                    // The function where it happened in the form of Struct.Func
	IsOAuth       bool   `json:"is_oauth,omitempty"`   // Whether the error is OAuth specific
	params        map[string]interface{}
}

func (er *AppError) Error() string {
	return er.Where + ": " + er.Message + ", " + er.DetailedError
}

func (er *AppError) ToJson() string {
	b, _ := json.Marshal(er)
	return string(b)
}

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")

func NewId() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26) // removes the '==' padding
	return b.String()
}

func NewAppError(where string, id string, params map[string]interface{}, details string, status int) *AppError {
	ap := &AppError{}
	ap.Id = id
	ap.params = params
	ap.Message = id
	ap.Where = where
	ap.DetailedError = details
	ap.StatusCode = status
	ap.IsOAuth = false
	//ap.Translate(translateFunc)
	return ap
}

func InterfaceToString(_args interface{}) string {
	return fmt.Sprintf("%v", _args)
}

func StringValueFromMap(name string, params map[string]interface{}, def string) (res string) {
	var ok bool
	var v interface{}

	if v, ok = params[name]; ok {

		switch v.(type) {
		case map[string]interface{}:
		case []interface{}:
			return def

		default:
			return fmt.Sprint(v)
		}
	}

	return def
}

func IntValueFromMap(name string, params map[string]interface{}, def int) int {
	var ok bool
	var v interface{}
	var res int

	if v, ok = params[name]; ok {
		switch v.(type) {
		case int:
			return v.(int)
		case float64:
			return int(v.(float64))
		case float32:
			return int(v.(float32))
		case string:
			var err error
			if res, err = strconv.Atoi(v.(string)); err == nil {
				return res
			}
		}
	}

	return def
}
