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

type StringArgs struct {
	SetVar string        `json:"setVar"`
	Fn     string        `json:"fn"`
	Data   string        `json:"data"`
	Args   []interface{} `json:"args"`
}

func (r *Router) stringApp(c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vm *otto.Otto
	var argv = StringArgs{}

	err := Decode(c, args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return nil, ErrorRequiredParameter("string", "setVar")
	}
	if argv.Fn == "" {
		return nil, ErrorRequiredParameter("string", "fn")
	}

	var value string

	switch argv.Fn {
	case "reverse":
		value = reverse(argv.Data)
		break
	case "charAt":
		value = charAt(argv.Data, GetTopIntArg(argv.Args))
		break
	case "base64":
		value = base64Fn(GetTopStringArg(argv.Args), argv.Data)
		break
	case "MD5":
		value = md5Fn(argv.Data)
		break
	case "SHA-256":
		value = sha256Fn(argv.Data)
		break
	case "SHA-512":
		value = sha512Fn(argv.Data)
		break
	default:
		vm = otto.New()
		vm.Set("fnName", argv.Fn)
		vm.Set("args", argv.Args)
		vm.Set("data", argv.Data)
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
		argv.SetVar: value,
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
