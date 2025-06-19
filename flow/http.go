package flow

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"gopkg.in/xmlpath.v2"
	"io"
	"io/ioutil"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	rePathVars = regexp.MustCompile(`\$\{([^}]+)\}`)
)

func (r *router) httpRequest(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool
	var res *http.Response
	var str string
	var httpErr error
	var uri string

	if props, ok = args.(map[string]interface{}); !ok {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
	}
	if uri = model.StringValueFromMap("url", props, ""); uri == "" {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, "url is required", http.StatusBadRequest)
	}
	uriEncoded := md5.Sum([]byte(uri))
	cookieVariableName := model.StringValueFromMap("exportCookie", props, "")
	cacheEnabled, _ := strconv.ParseBool(model.StringValueFromMap("cacheCookie", props, ""))
	cacheKey := fmt.Sprintf("%s.%s", uriEncoded, cookieVariableName)

	if cookieVariableName != "" && cacheEnabled {
		v, err := r.fm.CacheGetValue(ctx, string(app.Memory), conn.DomainId(), cacheKey)
		if err == nil {
			_, err = conn.Set(context.Background(), model.Variables{
				cookieVariableName: v,
			})
			if err != nil {
				return nil, err
			}
			wlog.Debug("http: found cookie cache entry, setting cache value")
			return model.CallResponseOK, nil
		}
	}
	req, err := r.buildRequest(conn, scope, props)
	if err != nil {
		return nil, err
	}

	client := buildHttpClient(props)

	res, httpErr = client.Do(req)
	if httpErr != nil {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, httpErr.Error(), http.StatusBadRequest)
	}
	defer res.Body.Close()

	if str := model.StringValueFromMap("responseCode", props, ""); str != "" {
		//TODO
		conn.Set(context.Background(), model.Variables{
			str: strconv.Itoa(res.StatusCode),
		})
	}

	if cookieVariableName != "" {
		if _, ok = res.Header["Set-Cookie"]; ok {
			cookie := strings.Join(res.Header["Set-Cookie"], ";")
			_, err = conn.Set(context.Background(), model.Variables{
				cookieVariableName: cookie, // TODO internal variables ?
			})
			if err != nil {
				return nil, err
			}
			if cacheEnabled {
				var cookieExpiresAfter int64
				for _, v := range res.Cookies() {
					expiresAfter := v.Expires.Unix() - time.Now().Unix() - int64(time.Hour.Seconds())
					if expiresAfter > 0 { // get minimal but not lower than 0 value
						if cookieExpiresAfter == 0 || expiresAfter < cookieExpiresAfter {
							cookieExpiresAfter = expiresAfter
						}
					}
				}
				err := r.fm.CacheSetValue(ctx, string(app.Memory), conn.DomainId(), cacheKey, cookie, cookieExpiresAfter)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if res.ContentLength == 0 {
		return model.CallResponseOK, nil
	}

	if str = model.StringValueFromMap("parser", props, ""); str == "" {
		str = res.Header.Get("content-type")
	}

	var exp map[string]interface{}
	if exp, ok = props["exportVariables"].(map[string]interface{}); ok {
		return parseHttpResponse(conn, str, res.Body, exp)
	}

	return model.CallResponseOK, nil
}

func buildHttpClient(props map[string]interface{}) *http.Client {
	client := &http.Client{
		Timeout: time.Duration(model.IntValueFromMap("timeout", props, 1000)) * time.Millisecond,
	}

	skipVerify := model.StringValueFromMap("insecureSkipVerify", props, "") == "true"
	renegotiation := model.StringValueFromMap("renegotiation", props, "")

	if skipVerify || renegotiation != "" {
		t := &tls.Config{
			InsecureSkipVerify: skipVerify,
		}

		switch renegotiation {
		case "renegotiateNever":
			t.Renegotiation = tls.RenegotiateNever
		case "renegotiateOnceAsClient":
			t.Renegotiation = tls.RenegotiateOnceAsClient
		case "renegotiateFreelyAsClient":
			t.Renegotiation = tls.RenegotiateFreelyAsClient
		}

		client.Transport = &http.Transport{
			TLSClientConfig: t,
		}
	}

	return client
}

func (r *router) buildRequest(c model.Connection, scope *Flow, props map[string]interface{}) (*http.Request, *model.AppError) {
	var ok bool
	var rawUrl string
	var err error
	var urlParam *url.URL
	var str, k, method string
	var v interface{}
	var body []byte
	var req *http.Request
	headers := make(map[string]string)

	if rawUrl = model.StringValueFromMap("url", props, ""); rawUrl == "" {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, "url is required", http.StatusBadRequest)
	}

	rawUrl = strings.Trim(rawUrl, " ")

	if m, ok := props["path"].(map[string]any); ok {
		vars := make(map[string]string, len(m))
		for k, v = range m {
			vars[k] = r.parseMapValue(c, v)
		}
		rawUrl = rawSubstitute(rawUrl, vars)
		rawUrl = c.ParseText(rawUrl)
		urlParam, err = url.Parse(rawUrl)

		if err != nil {
			return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse_path.args", nil, err.Error(), http.StatusBadRequest)
		}

		if len(urlParam.RawQuery) != 0 {
			q, _ := url.ParseQuery(urlParam.RawQuery)
			if q != nil {
				urlParam.RawQuery = encode(q)
			}
		}

	} else {
		urlParam, err = url.Parse(c.ParseText(rawUrl))
		if err != nil {
			return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.err.args", nil, err.Error(), http.StatusBadRequest)
		}
	}

	if _, ok = props["headers"]; ok {
		if _, ok = props["headers"].(map[string]interface{}); ok {
			for k, v = range props["headers"].(map[string]interface{}) {
				headers[strings.ToLower(k)] = r.parseMapValue(c, v)
			}
		}
	}

	if _, ok = headers["content-type"]; !ok {
		headers["content-type"] = "application/json"
	}

	if _, ok = props["data"]; ok {

		if strings.Index(headers["content-type"], "text/xml") > -1 || strings.Index(headers["content-type"], "application/soap+xml") > -1 {
			switch props["data"].(type) {
			case string:
				body = []byte(c.ParseText(model.StringValueFromMap("data", props, "")))
			}
		} else if strings.HasPrefix(headers["content-type"], "application/x-www-form-urlencoded") {
			str = ""
			urlEncodeData := url.Values{}
			switch props["data"].(type) {
			case map[string]interface{}:
				for k, v = range props["data"].(map[string]interface{}) {
					urlEncodeData.Set(k, r.parseMapValue(c, v))
				}
				str = urlEncodeData.Encode()
			case string:
				str = props["data"].(string)
			}
			body = []byte(str)
		} else {
			//JSON default
			switch pd := props["data"].(type) {
			case string:
				body = []byte(c.ParseText(pd))
				// TODO WTEL-4905 bug parse nested object
				/*
					case map[string]interface{}:
						var b model.JsonView
						if appErr := scope.Decode(props["data"], &b); appErr == nil {
							body, err = json.Marshal(b)
						} else {
							err = appErr
						}*/
			default:
				body, err = json.Marshal(props["data"])
				if err == nil {
					body = []byte(model.ParseText(c, string(body), model.ParseOptionJson))
				}
			}

			if err != nil {
				return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, err.Error(), http.StatusBadRequest)
			}
		}

	}

	method = strings.ToUpper(model.StringValueFromMap("method", props, "POST"))

	req, err = http.NewRequest(method, urlParam.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, err.Error(), http.StatusBadRequest)
	}

	for k, str = range headers {
		req.Header.Set(k, str)
	}
	return req, nil
}

func parseHttpResponse(c model.Connection, contentType string, response io.ReadCloser, exportVariables map[string]interface{}) (model.Response, *model.AppError) {
	var err error
	var body []byte

	if strings.Index(contentType, "application/json") > -1 {
		if len(exportVariables) > 0 {
			body, err = ioutil.ReadAll(response)
			if err != nil {
				return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse.err", nil, err.Error(), http.StatusBadRequest)
			}

			vars := model.Variables{}
			for k, _ := range exportVariables {
				// TODO DEV-4908
				r := gjson.GetBytes(body, model.StringValueFromMap(k, exportVariables, ""))
				if r.Type == gjson.JSON {
					dst := &bytes.Buffer{}
					err = json.Compact(dst, []byte(r.Raw))
					if err != nil {
						return model.CallResponseError, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.json.err",
							nil, err.Error(), http.StatusBadRequest)
					}
					vars[k] = dst.String()
				} else {
					vars[k] = r.String()
				}
			}
			if _, err := c.Set(context.Background(), vars); err != nil {
				return model.CallResponseError, err
			}
		}
	} else if strings.Index(contentType, "text/xml") > -1 {
		var xml *xmlpath.Node
		var path *xmlpath.Path

		if len(exportVariables) < 1 {
			return model.CallResponseOK, nil
		}

		xml, err = xmlpath.Parse(response)
		if err != nil {
			return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse.err", nil, err.Error(), http.StatusBadRequest)
		}

		for k, _ := range exportVariables {
			path, err = xmlpath.Compile(model.StringValueFromMap(k, exportVariables, ""))
			if err != nil {
				continue
			}

			if str, ok := path.String(xml); ok {
				if _, err := c.Set(context.Background(), model.Variables{
					k: str,
				}); err != nil {
					return nil, err
				}
			}
		}

	} else {
		body, err = ioutil.ReadAll(response)
		if err != nil {
			return model.CallResponseError, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse.err", nil, err.Error(), http.StatusBadRequest)
		}

		c.Log().Error(string(body))
		return model.CallResponseError, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse.err",
			nil, "no support parse content-type "+contentType, http.StatusBadRequest)
	}

	return model.CallResponseOK, nil
}

func (r *router) parseMapValue(c model.Connection, v interface{}) (str string) {
	str = model.InterfaceToString(v)
	if strings.HasSuffix(str, "}") {
		if strings.HasPrefix(str, "$${") {
			str = r.GlobalVariable(c.DomainId(), str[3:len(str)-1])
		} else if strings.HasPrefix(str, "${") {
			str, _ = c.Get(str[2 : len(str)-1])
		}
	}
	return str
}

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}

func rawSubstitute(template string, vars map[string]string) string {
	return rePathVars.ReplaceAllStringFunc(template, func(match string) string {
		name := match[2 : len(match)-1]
		if val, ok := vars[name]; ok {
			return val
		}
		return match // parse global
	})
}

func encode(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, k := range slices.Sorted(maps.Keys(v)) {
		vs := v[k]
		keyEscaped := encodeURIComponent(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(encodeURIComponent(v))
		}
	}
	return buf.String()
}

type CookieCacheOptions struct {
}
