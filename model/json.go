package model

import "encoding/json"

type JsonView map[string]interface{}

func ToJson(src interface{}) string {
	data, _ := json.Marshal(src)
	return string(data)
}
