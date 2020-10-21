package flow

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"reflect"
	"strconv"
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

func (f *Flow) Decode(in interface{}, out interface{}) *model.AppError {
	var hook mapstructure.DecodeHookFuncType = func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.String {
			switch to.Kind() {
			case reflect.String:
				return f.Connection.ParseText(data.(string)), nil
			case reflect.Interface:
				return f.Connection.ParseText(data.(string)), nil
			case reflect.Int:
				v := f.Connection.ParseText(data.(string))
				if v == "" {
					return 0, nil
				}
				return v, nil
			case reflect.Bool:
				return f.Connection.ParseText(data.(string)), nil
			}
		}
		return data, nil
	}

	return f.decode(in, out, hook)
}

func (f *Flow) DecodeSrc(in interface{}, out interface{}) *model.AppError {
	var hook mapstructure.DecodeHookFuncType = func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.String {
			switch to.Kind() {
			case reflect.String:
				return data.(string), nil
			case reflect.Interface:
				return data.(string), nil
			case reflect.Int:
				v := data.(string)
				if v == "" {
					return 0, nil
				}
				return v, nil
			case reflect.Bool:
				return data.(string), nil
			}
		}
		return data, nil
	}

	return f.decode(in, out, hook)
}

func (f *Flow) decode(in interface{}, out interface{}, hook mapstructure.DecodeHookFuncType) *model.AppError {
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		TagName:          "json",
		DecodeHook:       hook,
		Result:           &out,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return model.NewAppError("Parser", "parser.decode.create.err", nil, err.Error(), http.StatusBadRequest)
	}
	err = decoder.Decode(in)
	if err != nil {
		return model.NewAppError("Parser", "parser.decode.parse.err", nil, err.Error(), http.StatusBadRequest)
	}

	return nil
}

func GetTopStringArg(args []interface{}) string {
	if args != nil && len(args) > 0 {
		return fmt.Sprintf("%v", args[0])
	}

	return ""
}

func GetTopIntArg(args []interface{}) int {
	var v = 0
	if str := GetTopStringArg(args); str != "" {
		v, _ = strconv.Atoi(str)
	}

	return v
}
