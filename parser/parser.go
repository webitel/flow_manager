package parser

import (
	"github.com/mitchellh/mapstructure"
	"reflect"
)

const (
	tagName = "json"
	tagDef  = "def"
)

/*
\d := map[string]interface{}{
		"terminator": "ddsada",
		"files": []interface{}{
			map[string]interface{}{
				"name": 123,
				"id":   "${123}",
			},
		},
		"getDigits": map[string]interface{}{
			"setVar":    "getIvrDigit",
			"min":       "3",
			"max":       4,
			"tries":     1,
			"timeout":   2000,
			"flushDTMF": true,
		},
	}
*/

type Sleep int

func Decode(in interface{}, out interface{}) error {
	var f mapstructure.DecodeHookFuncType = func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.String {
			switch to.Kind() {
			case reflect.String:
				return "OK", nil
			case reflect.Int:
				return 1050, nil
			case reflect.Bool:
				return true, nil
			}
		}
		return data, nil
	}

	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		TagName:          "json",
		DecodeHook:       f,
		Result:           &out,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	err = decoder.Decode(in)
	if err != nil {
		return err
	}

	return nil
}
