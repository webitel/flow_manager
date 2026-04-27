package ops

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// DecodeArgs decodes in.Node.Args into out using mapstructure, expanding
// ${var} and $${globalVar} in every string field — the same contract as
// flow.Flow.Decode / scope.Decode in the legacy runtime.
//
// out must be a non-nil pointer to the target struct (e.g. &MyArgs{}).
// Struct fields are matched by their json tag (WeaklyTypedInput is on).
func DecodeArgs(in OpInput, out interface{}) error {
	expand := func(s string) string {
		return ExpandStr(s, in.Variables, in.GlobalVar)
	}

	hook := buildDecodeHook(expand)
	cfg := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		TagName:          "json",
		DecodeHook:       hook,
		Result:           out,
	}
	dec, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return err
	}
	return dec.Decode(in.Node.Args)
}

// buildDecodeHook returns a mapstructure hook that expands ${var} / $${var}
// in every string encountered during decode, plus handles the common
// string-to-numeric/slice coercions that the legacy runtime supported.
func buildDecodeHook(expand func(string) string) mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		switch from.Kind() {
		case reflect.String:
			s := expand(data.(string))

			switch to.Kind() {
			case reflect.String:
				return s, nil

			case reflect.Interface:
				// Try to unmarshal as JSON first (variable may have expanded to
				// an object/array), fall back to plain string.
				if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
					var v interface{}
					if json.Unmarshal([]byte(s), &v) == nil {
						return v, nil
					}
				}
				return s, nil

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if s == "" {
					return 0, nil
				}
				if strings.ContainsRune(s, '.') {
					f, err := strconv.ParseFloat(s, 64)
					if err != nil {
						return 0, err
					}
					return int64(f), nil
				}
				return s, nil // mapstructure will do the final Atoi with WeaklyTypedInput

			case reflect.Bool:
				return s, nil // mapstructure converts "true"/"1"/etc. with WeaklyTypedInput

			case reflect.Slice:
				// String that should become a slice: try to JSON-unmarshal the
				// expanded value (covers the case where a var holds a JSON array).
				if s == "" {
					return []interface{}{}, nil
				}
				var res interface{}
				if err := json.Unmarshal([]byte(s), &res); err == nil {
					return res, nil
				}
				return []interface{}{}, nil

			case reflect.Ptr:
				// *string
				if to.Elem().Kind() == reflect.String {
					return &s, nil
				}
				// *int / *int64 / etc.
				if k := to.Elem().Kind(); k >= reflect.Int && k <= reflect.Uint64 {
					if s == "" {
						return nil, nil
					}
					i, err := strconv.ParseInt(s, 10, 64)
					if err != nil {
						return nil, err
					}
					v := reflect.New(to.Elem())
					v.Elem().SetInt(i)
					return v.Interface(), nil
				}
			}

		case reflect.Map, reflect.Slice:
			// Nested object/array: JSON-marshal → expand strings inside → unmarshal.
			// Covers deep structures where leaf strings contain ${var}.
			body, err := json.Marshal(data)
			if err != nil {
				return data, nil
			}
			expanded := expandJSONStrings(string(body), expand)
			if to.Kind() == reflect.String {
				return expanded, nil
			}
			if to == reflect.TypeOf((*interface{})(nil)).Elem() {
				var v interface{}
				if json.Unmarshal([]byte(expanded), &v) == nil {
					return v, nil
				}
			}
		}

		return data, nil
	}
}

// expandJSONStrings walks raw JSON and applies expand() to every JSON string
// value (not keys). This lets ${var} inside nested objects be resolved without
// requiring a full custom JSON parser.
func expandJSONStrings(raw string, expand func(string) string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	walked := walkExpand(v, expand)
	out, err := json.Marshal(walked)
	if err != nil {
		return raw
	}
	return string(out)
}

func walkExpand(v interface{}, expand func(string) string) interface{} {
	switch val := v.(type) {
	case string:
		expanded := expand(val)
		// If the expanded value is a JSON object/array, parse it.
		if len(expanded) > 0 && (expanded[0] == '{' || expanded[0] == '[') {
			var inner interface{}
			if json.Unmarshal([]byte(expanded), &inner) == nil {
				return inner
			}
		}
		return expanded
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, elem := range val {
			out[k] = walkExpand(elem, expand)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, elem := range val {
			out[i] = walkExpand(elem, expand)
		}
		return out
	}
	return v
}
