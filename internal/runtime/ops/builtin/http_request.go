package builtin

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"gopkg.in/xmlpath.v2"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// CookieCache provides cookie persistence for the httpRequest op.
// Pass nil (or a nil-implementing value) to disable cookie caching.
type CookieCache interface {
	GetCookieCache(ctx context.Context, domainID int64, key string) (string, error)
	SetCookieCache(ctx context.Context, domainID int64, key string, value string, ttlSecs int64) error
}

type httpRequestOp struct {
	cache CookieCache
}

// HTTPRequestOp returns the native httpRequest op.
// cache may be nil to disable cookie caching.
func HTTPRequestOp(cache CookieCache) ops.Op { return &httpRequestOp{cache: cache} }

func (o *httpRequestOp) Kind() ops.OpKind { return ops.OpKindSync }

var reHTTPPathVars = regexp.MustCompile(`\$\{([^}]+)\}`)

func (o *httpRequestOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	props := in.Node.Args
	expandFn := func(s string) string { return ops.ExpandStr(s, in.Variables, in.GlobalVar) }
	expandVal := func(v any) string { return expandFn(fmt.Sprintf("%v", v)) }

	rawURL := httpStrFromProps(props, "url", "")
	if rawURL == "" {
		return ops.OpOutput{}, fmt.Errorf("httpRequest: url is required")
	}

	cookieVarName := httpStrFromProps(props, "exportCookie", "")
	cacheEnabled, _ := strconv.ParseBool(httpStrFromProps(props, "cacheCookie", ""))
	uriHash := md5.Sum([]byte(rawURL))
	cacheKey := fmt.Sprintf("%s.%s", uriHash, cookieVarName)

	outVars := make(map[string]string)

	if cookieVarName != "" && cacheEnabled && o.cache != nil {
		if cached, err := o.cache.GetCookieCache(ctx, in.DomainID, cacheKey); err == nil && cached != "" {
			outVars[cookieVarName] = cached
			return ops.OpOutput{SetVars: outVars}, nil
		}
	}

	req, err := buildNativeRequest(props, expandFn, expandVal)
	if err != nil {
		return ops.OpOutput{}, err
	}
	req = req.WithContext(ctx)

	client := buildHTTPClient(props)
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return ops.OpOutput{}, fmt.Errorf("httpRequest: %w", httpErr)
	}
	defer resp.Body.Close()

	if rc := httpStrFromProps(props, "responseCode", ""); rc != "" {
		outVars[rc] = strconv.Itoa(resp.StatusCode)
	}

	if cookieVarName != "" {
		if _, ok := resp.Header["Set-Cookie"]; ok {
			cookie := strings.Join(resp.Header["Set-Cookie"], ";")
			outVars[cookieVarName] = cookie
			if cacheEnabled && o.cache != nil {
				var minTTL int64
				for _, c := range resp.Cookies() {
					ttl := c.Expires.Unix() - time.Now().Unix() - int64(time.Hour.Seconds())
					if ttl > 0 && (minTTL == 0 || ttl < minTTL) {
						minTTL = ttl
					}
				}
				_ = o.cache.SetCookieCache(ctx, in.DomainID, cacheKey, cookie, minTTL)
			}
		}
	}

	if resp.ContentLength == 0 {
		return ops.OpOutput{SetVars: outVars}, nil
	}

	contentType := httpStrFromProps(props, "parser", "")
	if contentType == "" {
		contentType = resp.Header.Get("content-type")
	}

	exportVars, hasExport := props["exportVariables"].(map[string]any)
	if !hasExport {
		return ops.OpOutput{SetVars: outVars}, nil
	}

	parsed, parseErr := parseNativeHTTPResponse(contentType, resp.Body, exportVars)
	if parseErr != nil {
		return ops.OpOutput{}, parseErr
	}
	maps.Copy(outVars, parsed)

	return ops.OpOutput{SetVars: outVars}, nil
}

// buildNativeRequest is the native equivalent of flow/http.go buildRequest.
func buildNativeRequest(props map[string]any, expandFn func(string) string, expandVal func(any) string) (*http.Request, error) {
	rawURL := strings.TrimSpace(httpStrFromProps(props, "url", ""))
	if rawURL == "" {
		return nil, fmt.Errorf("httpRequest: url is required")
	}

	// Path substitution: replace ${key} in URL using the path map.
	if pathMap, ok := props["path"].(map[string]any); ok {
		pathVars := make(map[string]string, len(pathMap))
		for k, v := range pathMap {
			pathVars[k] = expandVal(v)
		}
		rawURL = httpRawSubstitute(rawURL, pathVars)
	}
	rawURL = expandFn(rawURL)

	parsedURL, urlErr := url.Parse(rawURL)
	if urlErr != nil {
		return nil, fmt.Errorf("httpRequest: parse url: %w", urlErr)
	}
	if len(parsedURL.RawQuery) > 0 {
		if q, _ := url.ParseQuery(parsedURL.RawQuery); q != nil {
			parsedURL.RawQuery = httpEncode(q)
		}
	}

	headers := map[string]string{"content-type": "application/json"}
	if rawHeaders, ok := props["headers"].(map[string]any); ok {
		for k, v := range rawHeaders {
			headers[strings.ToLower(k)] = expandVal(v)
		}
	}

	var body []byte
	if data, hasData := props["data"]; hasData {
		ct := headers["content-type"]
		switch {
		case strings.Contains(ct, "text/xml") || strings.Contains(ct, "application/soap+xml"):
			if s, ok := data.(string); ok {
				body = []byte(expandFn(s))
			}
		case strings.HasPrefix(ct, "application/x-www-form-urlencoded"):
			vals := url.Values{}
			switch d := data.(type) {
			case map[string]any:
				for k, v := range d {
					vals.Set(k, expandVal(v))
				}
				body = []byte(vals.Encode())
			case string:
				body = []byte(d)
			}
		default:
			switch d := data.(type) {
			case string:
				body = []byte(expandFn(d))
			default:
				b, err := json.Marshal(d)
				if err != nil {
					return nil, fmt.Errorf("httpRequest: marshal data: %w", err)
				}
				body = []byte(httpExpandJSONBody(string(b), func(name string) string {
					return expandFn("${" + name + "}")
				}))
			}
		}
	}

	method := strings.ToUpper(httpStrFromProps(props, "method", "POST"))
	req, err := http.NewRequest(method, parsedURL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("httpRequest: new request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// buildHTTPClient builds an http.Client from props.
// Identical logic to flow/http.go buildHttpClient.
func buildHTTPClient(props map[string]any) *http.Client {
	client := &http.Client{
		Timeout: time.Duration(httpIntFromProps(props, "timeout", 1000)) * time.Millisecond,
	}
	skipVerify := httpStrFromProps(props, "insecureSkipVerify", "") == "true"
	renegotiation := httpStrFromProps(props, "renegotiation", "")
	if skipVerify || renegotiation != "" {
		tlsCfg := &tls.Config{InsecureSkipVerify: skipVerify} //nolint:gosec
		switch renegotiation {
		case "renegotiateNever":
			tlsCfg.Renegotiation = tls.RenegotiateNever
		case "renegotiateOnceAsClient":
			tlsCfg.Renegotiation = tls.RenegotiateOnceAsClient
		case "renegotiateFreelyAsClient":
			tlsCfg.Renegotiation = tls.RenegotiateFreelyAsClient
		}
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	}
	return client
}

// parseNativeHTTPResponse parses the HTTP response body into a variable map.
// Mirrors flow/http.go parseHttpResponse but returns map[string]string.
func parseNativeHTTPResponse(contentType string, body io.ReadCloser, exportVars map[string]any) (map[string]string, error) {
	out := make(map[string]string, len(exportVars))

	switch {
	case strings.Contains(contentType, "application/json"):
		raw, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("httpRequest: read body: %w", err)
		}
		for varName := range exportVars {
			path, _ := exportVars[varName].(string)
			r := gjson.GetBytes(raw, path)
			if r.Type == gjson.JSON {
				dst := &bytes.Buffer{}
				if err := json.Compact(dst, []byte(r.Raw)); err != nil {
					return nil, fmt.Errorf("httpRequest: compact json: %w", err)
				}
				out[varName] = dst.String()
			} else {
				out[varName] = r.String()
			}
		}

	case strings.Contains(contentType, "text/xml"):
		xmlNode, err := xmlpath.Parse(body)
		if err != nil {
			return nil, fmt.Errorf("httpRequest: parse xml: %w", err)
		}
		for varName := range exportVars {
			pathStr, _ := exportVars[varName].(string)
			path, err := xmlpath.Compile(pathStr)
			if err != nil {
				continue
			}
			if v, ok := path.String(xmlNode); ok {
				out[varName] = v
			}
		}

	default:
		raw, _ := io.ReadAll(body)
		return nil, fmt.Errorf("httpRequest: unsupported content-type %q: %s", contentType, raw)
	}

	return out, nil
}

// httpExpandJSONBody expands ${var} placeholders inside a JSON-encoded string,
// JSON-escaping each substituted value so the result remains valid JSON.
// The resolver function receives the variable name and returns its value.
func httpExpandJSONBody(jsonStr string, resolve func(name string) string) string {
	return reHTTPPathVars.ReplaceAllStringFunc(jsonStr, func(m string) string {
		sub := reHTTPPathVars.FindStringSubmatch(m)
		val := resolve(sub[1])
		// json.Marshal a string produces `"..."` — strip the outer quotes to get the escaped content.
		if b, err := json.Marshal(val); err == nil && len(b) >= 2 {
			return string(b[1 : len(b)-1])
		}
		return val
	})
}

// httpRawSubstitute replaces ${key} patterns in template using vars.
// Unknown placeholders are left as-is (for the subsequent ops.ExpandStr call).
func httpRawSubstitute(template string, vars map[string]string) string {
	return reHTTPPathVars.ReplaceAllStringFunc(template, func(match string) string {
		name := match[2 : len(match)-1]
		if val, ok := vars[name]; ok {
			return val
		}
		return match
	})
}

func httpStrFromProps(props map[string]any, key, def string) string {
	v, ok := props[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]any, []any:
		return def
	default:
		return fmt.Sprint(val)
	}
}

func httpIntFromProps(props map[string]any, key string, def int) int {
	v, ok := props[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

func httpEncode(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, k := range slices.Sorted(maps.Keys(v)) {
		for i, val := range v[k] {
			if buf.Len() > 0 || i > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(httpEncodeURIComponent(k))
			buf.WriteByte('=')
			buf.WriteString(httpEncodeURIComponent(val))
		}
	}
	return buf.String()
}

func httpEncodeURIComponent(s string) string {
	r := url.QueryEscape(s)
	return strings.ReplaceAll(r, "+", "%20")
}
