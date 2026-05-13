package utils

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

// GetMillis returns current time in milliseconds since epoch.
func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")

func NewId() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26)
	return b.String()
}

func InterfaceToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func StringValueFromMap(name string, params map[string]interface{}, def string) string {
	v, ok := params[name]
	if !ok {
		return def
	}
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return def
	}
	return fmt.Sprint(v)
}

func IntValueFromMap(name string, params map[string]interface{}, def int) int {
	v, ok := params[name]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case float32:
		return int(t)
	case string:
		if res, err := strconv.Atoi(t); err == nil {
			return res
		}
	}
	return def
}

func InterfaceToJson(i interface{}) []byte {
	v, _ := json.Marshal(i)
	return v
}

func UrlEncoded(str string) string {
	res := url.Values{"": {str}}.Encode()
	if len(res) < 2 {
		return ""
	}
	return compatibleJSEncodeURIComponent(res[1:])
}

func compatibleJSEncodeURIComponent(str string) string {
	r := str
	r = strings.Replace(r, "+", "%20", -1)
	r = strings.Replace(r, "%21", "!", -1)
	r = strings.Replace(r, "%28", "(", -1)
	r = strings.Replace(r, "%29", ")", -1)
	r = strings.Replace(r, "%2A", "*", -1)
	return r
}
