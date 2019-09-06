// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httputils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/moul/http2curl"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/trace"

	"yunion.io/x/onecloud/pkg/appctx"
)

type THttpMethod string

const (
	USER_AGENT = "yunioncloud-go/201708"

	GET    = THttpMethod("GET")
	HEAD   = THttpMethod("HEAD")
	POST   = THttpMethod("POST")
	PUT    = THttpMethod("PUT")
	PATCH  = THttpMethod("PATCH")
	DELETE = THttpMethod("DELETE")
	OPTION = THttpMethod("OPTION")
)

var (
	red    = color.New(color.FgRed, color.Bold).PrintlnFunc()
	green  = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	yellow = color.New(color.FgYellow, color.Bold).PrintlnFunc()
	cyan   = color.New(color.FgHiCyan, color.Bold).PrintlnFunc()
)

type Error struct {
	Id     string
	Fields []string
}

type JSONClientError struct {
	Code    int
	Class   string
	Details string
	Data    Error
}

type JSONClientErrorMsg struct {
	Error *JSONClientError
}

func (e *JSONClientError) Error() string {
	errMsg := JSONClientErrorMsg{Error: e}
	return jsonutils.Marshal(errMsg).String()
}

func ErrorCode(err error) int {
	if err == nil {
		return 0
	}
	switch je := err.(type) {
	case *JSONClientError:
		return je.Code
	}
	return -1
}

func headerExists(header *http.Header, key string) bool {
	keyu := strings.ToUpper(key)
	for k := range *header {
		if strings.ToUpper(k) == keyu {
			return true
		}
	}
	return false
}

func GetAddrPort(urlStr string) (string, int, error) {
	parts, err := url.Parse(urlStr)
	if err != nil {
		return "", 0, err
	}
	host := parts.Host
	commaPos := strings.IndexByte(host, ':')
	if commaPos > 0 {
		port, err := strconv.ParseInt(host[commaPos+1:], 10, 32)
		if err != nil {
			return "", 0, err
		} else {
			return host[:commaPos], int(port), nil
		}
	} else {
		switch parts.Scheme {
		case "http":
			return parts.Host, 80, nil
		case "https":
			return parts.Host, 443, nil
		default:
			return "", 0, fmt.Errorf("Unknown schema %s", parts.Scheme)
		}
	}
}

func GetTransport(insecure bool, timeout time.Duration) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
		}).DialContext,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
		DisableCompression:    true,
	}
}

func GetClient(insecure bool, timeout time.Duration) *http.Client {
	tr := GetTransport(insecure, timeout)
	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

func GetTimeoutClient(timeout time.Duration) *http.Client {
	return GetClient(true, timeout)
}

var defaultHttpClient *http.Client

func init() {
	defaultHttpClient = GetClient(true, time.Second*15)
}

func GetDefaultClient() *http.Client {
	return defaultHttpClient
}

func Request(client *http.Client, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	if client == nil {
		client = defaultHttpClient
	}
	if header == nil {
		header = http.Header{}
	}
	ctxData := appctx.FetchAppContextData(ctx)
	var clientTrace *trace.STrace
	if !ctxData.Trace.IsZero() {
		addr, port, err := GetAddrPort(urlStr)
		if err != nil {
			return nil, err
		}
		clientTrace = trace.StartClientTrace(&ctxData.Trace, addr, port, ctxData.ServiceName)
		clientTrace.AddClientRequestHeader(header)
	}
	if len(ctxData.RequestId) > 0 {
		header.Set("X-Request-Id", ctxData.RequestId)
	}
	req, err := http.NewRequest(string(method), urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", USER_AGENT)
	if body == nil {
		req.Header.Set("Content-Length", "0")
	} else {
		if headerExists(&header, "Content-Length") {
			clen := header.Get("Content-Length")
			req.ContentLength, _ = strconv.ParseInt(clen, 10, 64)
		}
	}
	if header != nil {
		for k, v := range header {
			req.Header[k] = v
		}
	}
	if debug {
		yellow("Request", method, urlStr, req.Header, body)
		// 忽略掉上传文件的请求,避免大量日志输出
		if header.Get("Content-Type") != "application/octet-stream" {
			curlCmd, _ := http2curl.GetCurlCommand(req)
			cyan("CURL:", curlCmd)
		}
	}
	resp, err := client.Do(req)
	if err == nil && clientTrace != nil {
		clientTrace.EndClientTraceHeader(resp.Header)
	}
	return resp, err
}

func JSONRequest(client *http.Client, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	bodystr := ""
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	resp, err := Request(client, ctx, method, urlStr, header, jbody, debug)
	return ParseJSONResponse(resp, err, debug)
}

// closeResponse close non nil response with any response Body.
// convenient wrapper to drain any remaining data on response body.
//
// Subsequently this allows golang http RoundTripper
// to re-use the same connection for future requests.
func CloseResponse(resp *http.Response) {
	// Callers should close resp.Body when done reading from it.
	// If resp.Body is not closed, the Client's underlying RoundTripper
	// (typically Transport) may not be able to re-use a persistent TCP
	// connection to the server for a subsequent "keep-alive" request.
	if resp != nil && resp.Body != nil {
		// Drain any remaining Body and then close the connection.
		// Without this closing connection would disallow re-using
		// the same connection for future uses.
		//  - http://stackoverflow.com/a/17961593/4465767
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func ParseResponse(resp *http.Response, err error, debug bool) (http.Header, []byte, error) {
	if err != nil {
		ce := JSONClientError{}
		ce.Code = 499
		ce.Details = err.Error()
		return nil, nil, &ce
	}
	defer CloseResponse(resp)
	if debug {
		if resp.StatusCode < 300 {
			green("Status:", resp.StatusCode)
			green(resp.Header)
		} else if resp.StatusCode < 400 {
			yellow("Status:", resp.StatusCode)
			yellow(resp.Header)
		} else {
			red("Status:", resp.StatusCode)
			red(resp.Header)
		}
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("Fail to read body: %s", err)
	}
	if resp.StatusCode < 300 {
		return resp.Header, rbody, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return nil, nil, &ce
	} else {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Status
		if len(rbody) > 0 {
			ce.Details = string(rbody)
		}
		return nil, nil, &ce
	}
}

func ParseJSONResponse(resp *http.Response, err error, debug bool) (http.Header, jsonutils.JSONObject, error) {
	if err != nil {
		ce := JSONClientError{}
		ce.Code = 499
		ce.Details = err.Error()
		return nil, nil, &ce
	}
	defer CloseResponse(resp)
	if debug {
		if resp.StatusCode < 300 {
			green("Status:", resp.StatusCode)
			green(resp.Header)
		} else if resp.StatusCode < 400 {
			yellow("Status:", resp.StatusCode)
			yellow(resp.Header)
		} else {
			red("Status:", resp.StatusCode)
			red(resp.Header)
		}
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("Fail to read body: %s", err)
	}

	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 && string(rbody[0]) == "{" {
		var err error
		jrbody, err = jsonutils.Parse(rbody)
		if err != nil && debug {
			fmt.Fprintf(os.Stderr, "parsing json failed: %s", err)
		}
	}

	if resp.StatusCode < 300 {
		return resp.Header, jrbody, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return nil, nil, &ce
	} else {
		ce := JSONClientError{}

		if jrbody == nil {
			ce.Code = resp.StatusCode
			ce.Details = resp.Status
			if len(rbody) > 0 {
				ce.Details = string(rbody)
			}
			return nil, nil, &ce
		}

		err = jrbody.Unmarshal(&ce)
		if len(ce.Class) > 0 && ce.Code >= 400 && len(ce.Details) > 0 {
			return nil, nil, &ce
		}

		jrbody1, err := jrbody.GetMap()
		if err != nil {
			err = jrbody.Unmarshal(&ce)
			if err != nil {
				ce.Details = err.Error()
			}
			return nil, nil, &ce
		}
		var jrbody2 jsonutils.JSONObject
		if len(jrbody1) > 1 {
			jrbody2 = jsonutils.Marshal(jrbody1)
		} else {
			for _, v := range jrbody1 {
				jrbody2 = v
			}
		}
		if ecode, _ := jrbody2.GetString("code"); len(ecode) > 0 {
			code, err := strconv.Atoi(ecode)
			if err != nil {
				ce.Class = ecode
			} else {
				ce.Code = code
			}
		}
		if ce.Code == 0 {
			ce.Code = resp.StatusCode
		}
		if edetail := jsonutils.GetAnyString(jrbody2, []string{"message", "detail", "details", "error_msg"}); len(edetail) > 0 {
			ce.Details = edetail
		}
		if eclass := jsonutils.GetAnyString(jrbody2, []string{"title", "type", "error_code"}); len(eclass) > 0 {
			ce.Class = eclass
		}
		return nil, nil, &ce
	}
}

func JoinPath(ep string, path string) string {
	return strings.TrimRight(ep, "/") + "/" + strings.TrimLeft(path, "/")
}
