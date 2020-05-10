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

func Decode(conn model.Connection, in interface{}, out interface{}) *model.AppError {
	var f mapstructure.DecodeHookFuncType = func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if from.Kind() == reflect.String {
			switch to.Kind() {
			case reflect.String:
				return conn.ParseText(data.(string)), nil
			case reflect.Interface:
				return conn.ParseText(data.(string)), nil
			case reflect.Int:
				return conn.ParseText(data.(string)), nil
			case reflect.Bool:
				return conn.ParseText(data.(string)), nil
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
