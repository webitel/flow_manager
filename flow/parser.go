package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/webitel/flow_manager/model"
)

var jsonValueT = reflect.TypeOf(&model.JsonValue{})
var JsonViewT = reflect.TypeOf(&model.JsonView{})

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

func (f *Flow) parseValidJson(in string) string {
	l := len(in)
	if l <= 3 {
		return in
	}

	res := bytes.NewBufferString("")

	var token string

	for i := 0; i < l; i++ {
		/*
			36 = $
			123 = {
			125 = }
		*/
		if in[i] == 36 && l > i+2 && in[i+1] == 123 && in[i+2] != 125 && token == "" {
			i = i + 2
			token += string(in[i])
		} else if token != "" {
			if in[i] == 125 {
				token, _ = f.Connection.Get(token)
				if token != "" {
					ll := len(token)
					if l > 2 && (token[0] == 91 || token[0] == 123) && (token[ll-1] == 93 || token[ll-1] == 125) {
						i++
						res.Truncate(res.Len() - 1)
					}
					res.WriteString(token)

					token = ""
				}
			} else {
				token += string(in[i])
			}
		} else {
			res.WriteString(string(in[i]))
		}
	}
	return res.String()
}

func (f *Flow) parseGlobalVariables(text string) string {
	text = compileVarsGlobal.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVarsGlobal.FindStringSubmatch(varName)
		if len(r) > 0 {
			out = f.router.GlobalVariable(f.Connection.DomainId(), r[1])
		}

		return
	})

	return text
}

func (f *Flow) parseString(text string) string {
	text = f.parseGlobalVariables(text)
	return f.Connection.ParseText(text)
}

func (f *Flow) Decode(in interface{}, out interface{}) *model.AppError {
	var hook mapstructure.DecodeHookFuncType = func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		kind := from.Kind()
		if kind == reflect.String {
			switch to.Kind() {
			case reflect.Ptr:
				if to.AssignableTo(jsonValueT) {
					d := f.parseString(data.(string))
					o := model.JsonValue(d)
					return &o, nil
				} else if to.AssignableTo(JsonViewT) { // TODO
					d := f.parseString(data.(string))
					o := model.JsonView{}
					json.Unmarshal([]byte(d), &o)
					return &o, nil
				}
			case reflect.Slice:
				var res interface{}
				body, err := json.Marshal(data)
				if err != nil {
					return nil, err
				}

				if len(body) < 2 {
					return data, nil
				}

				txt := f.parseString(string(body[1 : len(body)-1]))

				err = json.Unmarshal([]byte(txt), &res)
				if err != nil {
					if txt, _ = data.(string); txt != "" {
						_ = json.Unmarshal([]byte(txt), &res)
					}

					if res == nil {
						res = []interface{}{}
					}
				}
				return res, nil
			case reflect.String:
				return f.parseString(data.(string)), nil
			case reflect.Interface:
				return f.parseString(data.(string)), nil
			//fixme added more types
			case reflect.Int, reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Int64, reflect.Int32:
				v := f.parseString(data.(string))
				if v == "" {
					return 0, nil
				}

				if strings.Index(v, ".") > -1 {
					res, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return 0, err
					}
					return res, nil
				}
				return v, nil
			case reflect.Bool:
				return f.parseString(data.(string)), nil
			}
		} else if (kind == reflect.Map || kind == reflect.Slice) && to.AssignableTo(jsonValueT) {
			body, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}

			if len(body) < 2 {
				return data, nil
			}

			txt := []byte(f.parseValidJson(string(body)))
			return txt, nil
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
