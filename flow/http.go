package flow

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/webitel/flow_manager/model"
	"gopkg.in/xmlpath.v2"
)

func (r *router) httpRequest(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool
	var res *http.Response
	var str string
	var httpErr error

	if props, ok = args.(map[string]interface{}); !ok {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
	}

	req, err := buildRequest(conn, props)
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

	if str = model.StringValueFromMap("exportCookie", props, ""); str != "" {
		if _, ok = res.Header["Set-Cookie"]; ok {
			conn.Set(context.Background(), model.Variables{
				str: strings.Join(res.Header["Set-Cookie"], ";"), // TODO internal variables ?
			})
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

func buildRequest(c model.Connection, props map[string]interface{}) (*http.Request, *model.AppError) {
	var ok bool
	var uri string
	var err error
	var urlParam *url.URL
	var str, k, method string
	var v interface{}
	var body []byte
	var req *http.Request
	headers := make(map[string]string)

	if uri = model.StringValueFromMap("url", props, ""); uri == "" {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, "url is required", http.StatusBadRequest)
	}

	if _, ok = props["path"]; ok {
		if _, ok = props["path"].(map[string]interface{}); ok {
			for k, v = range props["path"].(map[string]interface{}) {
				str = parseMapValue(c, v)
				uri = strings.Replace(uri, "${"+k+"}", encodeURIComponent(str), -1)
			}
		}
	}

	urlParam, err = url.Parse(strings.Trim(uri, " "))
	if err != nil {
		return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.err.args", nil, err.Error(), http.StatusBadRequest)
	}

	if _, ok = props["headers"]; ok {
		if _, ok = props["headers"].(map[string]interface{}); ok {
			for k, v = range props["headers"].(map[string]interface{}) {
				headers[strings.ToLower(k)] = parseMapValue(c, v)
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
					urlEncodeData.Set(k, parseMapValue(c, v))
				}
				str = urlEncodeData.Encode()
			case string:
				str = props["data"].(string)
			}
			body = []byte(str)
		} else {
			//JSON default
			body, err = json.Marshal(props["data"])
			if err != nil {
				return nil, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.valid.args", nil, err.Error(), http.StatusBadRequest)
			} else {
				body = []byte(c.ParseText(string(body)))
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
				vars[k] = gjson.GetBytes(body, model.StringValueFromMap(k, exportVariables, "")).String()
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

		fmt.Println(string(body))
		return model.CallResponseError, model.NewAppError("Flow.HttpRequest", "flow.app.http_request.parse.err",
			nil, "no support parse content-type "+contentType, http.StatusBadRequest)
	}

	return model.CallResponseOK, nil
}

func parseMapValue(c model.Connection, v interface{}) (str string) {
	str = model.InterfaceToString(v)
	if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
		str, _ = c.Get(str[2 : len(str)-1])
	}
	return str
}

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}
