package flow

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (r *Router) stringApp(c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool
	var vm *otto.Otto
	var varName, fnName, data, value string
	var argsElem []interface{}
	var _args interface{}

	if props, ok = args.(map[string]interface{}); !ok {
		return nil, model.NewAppError("Flow.String", "flow.app.string.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
	}

	if varName = model.StringValueFromMap("setVar", props, ""); varName == "" {
		return nil, model.NewAppError("Flow.String", "flow.app.string.valid.args", nil, fmt.Sprintf("setVar is required %v", args), http.StatusBadRequest)
	}

	if fnName = model.StringValueFromMap("fn", props, ""); fnName == "" {
		return nil, model.NewAppError("Flow.String", "flow.app.string.valid.args", nil, fmt.Sprintf("fn is required %v", args), http.StatusBadRequest)
	}

	data = c.ParseText(model.StringValueFromMap("data", props, ""))

	switch fnName {
	case "reverse":
		value = reverse(data)
		break
	case "charAt":
		if pos := model.IntValueFromMap("args", props, -1); pos > -1 {
			value = charAt(data, pos)
		}
		break
	case "base64":
		mode := ""
		if _args, ok = props["args"]; ok {
			mode = model.InterfaceToString(_args)
		}
		value = base64Fn(mode, data)
		break
	case "MD5":
		value = md5Fn(data)
		break
	case "SHA-256":
		value = sha256Fn(data)
		break
	case "SHA-512":
		value = sha512Fn(data)
		break
	default:
		if _args, ok = props["args"]; ok {
			//FIXME NOW
			//argsElem = parseArgsToArrayInterface(c, _args)
		} else {
			argsElem = []interface{}{}
		}

		vm = otto.New()
		vm.Set("fnName", fnName)
		vm.Set("args", argsElem)
		vm.Set("data", data)
		v, err := vm.Run(`
				var value, match;

				if (args instanceof Array) {
					args = args.map(function(v) {
						if (typeof v === "string") {
							match = v.match(new RegExp('^/(.*?)/([gimy]*)$'));
							if (match) {
								return new RegExp(match[1], match[2])
							}
						}
						return v;
					})
				} else {
					args = [args]
				}

				if (typeof data[fnName] === "function") {
					value = data[fnName].apply(data, args)
				} else {
					throw "Bad string function " + fnName
				}
			`)

		if err != nil {
			return nil, model.NewAppError("Flow.String", "flow.app.string.error.args", nil, err.Error(), http.StatusBadRequest)
		}

		value = v.String()
	}

	return c.Set(map[string]interface{}{
		varName: value,
	})
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func charAt(s string, pos int) string {
	if len(s) > pos {
		return string(s[pos])
	}
	return ""
}

func base64Fn(mode, data string) string {
	if mode == "encoder" {
		return base64.StdEncoding.EncodeToString([]byte(data))
	} else if mode == "decoder" {
		body, _ := base64.StdEncoding.DecodeString(data)
		return string(body)
	}
	return ""
}

func md5Fn(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

func sha256Fn(data string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}

func sha512Fn(data string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(data)))
}
