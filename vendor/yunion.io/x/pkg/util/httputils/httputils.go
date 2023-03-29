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
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"moul.io/http2curl/v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/trace"
	"yunion.io/x/pkg/utils"
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

	IdleConnTimeout       = 60
	TLSHandshakeTimeout   = 10
	ResponseHeaderTimeout = 30
)

var (
	red    = color.New(color.FgRed, color.Bold).PrintlnFunc()
	green  = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	yellow = color.New(color.FgYellow, color.Bold).PrintlnFunc()
	cyan   = color.New(color.FgHiCyan, color.Bold).PrintlnFunc()
)

type Error struct {
	Id     string        `json:"id,omitempty"`
	Fields []interface{} `json:"fields,omitempty"`
}

type JSONClientError struct {
	Request struct {
		Method  string               `json:"method,omitempty"`
		Url     string               `json:"url,omitempty"`
		Body    jsonutils.JSONObject `json:"body,omitempty"`
		Headers map[string]string    `json:"headers,omitempty"`
	} `json:"request,omitempty"`

	Code    int    `json:"code,omitzero"`
	Class   string `json:"class,omitempty"`
	Details string `json:"details,omitempty"`
	Data    Error  `json:"data,omitempty"`
}

type sClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// body might have been consumed, so body is provided separately
func newJsonClientErrorFromRequest(req *http.Request, body string) *JSONClientError {
	return newJsonClientErrorFromRequest2(req.Method, req.URL.String(), req.Header, body)
}

func newJsonClientErrorFromRequest2(method string, urlStr string, hdrs http.Header, body string) *JSONClientError {
	jce := &JSONClientError{}

	jce.Request.Method = strings.ToUpper(method)
	jce.Request.Url = urlStr
	jce.Request.Headers = make(map[string]string)
	excludeHdrs := []string{
		"Accept",
		"Accept-Encoding",
	}
	authHdrs := []string{
		http.CanonicalHeaderKey("authorization"),
		http.CanonicalHeaderKey("x-auth-token"),
		http.CanonicalHeaderKey("x-subject-token"),
	}
	const (
		MAX_BODY   = 128
		FIRST_PART = 100
	)
	switch jce.Request.Method {
	case "PUT", "POST", "PATCH":
		contType := hdrs.Get(http.CanonicalHeaderKey("content-type"))
		if len(body) > MAX_BODY {
			jce.Request.Body = jsonutils.NewString(body[:FIRST_PART] + "..." + body[len(body)-MAX_BODY+FIRST_PART+3:])
		} else if strings.Contains(contType, "json") {
			jce.Request.Body, _ = jsonutils.ParseString(body)
		} else if strings.Contains(contType, "xml") ||
			strings.Contains(contType, "x-www-form-urlencoded") {
			jce.Request.Body = jsonutils.NewString(body)
		}
	default:
		excludeHdrs = append(excludeHdrs, http.CanonicalHeaderKey("content-type"), http.CanonicalHeaderKey("content-length"))
	}
	for h := range hdrs {
		ch := http.CanonicalHeaderKey(h)
		if utils.IsInStringArray(ch, excludeHdrs) {
			continue
		}
		if utils.IsInStringArray(ch, authHdrs) {
			jce.Request.Headers[ch] = "*"
		} else {
			jce.Request.Headers[ch] = hdrs.Get(ch)
		}
	}

	return jce
}

type JSONClientErrorMsg struct {
	Error *JSONClientError
}

type JsonClient struct {
	client sClient
}

type JsonRequest interface {
	GetHttpMethod() THttpMethod
	GetRequestBody() jsonutils.JSONObject
	GetUrl() string
	SetHttpMethod(method THttpMethod)
	GetHeader() http.Header
	SetHeader(header http.Header)
}

type JsonBaseRequest struct {
	httpMethod THttpMethod
	url        string
	params     interface{}
	header     http.Header
}

func (req *JsonBaseRequest) GetHttpMethod() THttpMethod {
	return req.httpMethod
}

func (req *JsonBaseRequest) GetRequestBody() jsonutils.JSONObject {
	if req.params != nil {
		return jsonutils.Marshal(req.params)
	}
	return nil
}

func (req *JsonBaseRequest) GetUrl() string {
	return req.url
}

func (req *JsonBaseRequest) SetHttpMethod(method THttpMethod) {
	req.httpMethod = method
}

func (req *JsonBaseRequest) GetHeader() http.Header {
	return req.header
}

func (req *JsonBaseRequest) SetHeader(header http.Header) {
	for k, values := range header {
		req.header.Del(k)
		for _, v := range values {
			req.header.Add(k, v)
		}
	}
}

func NewJsonRequest(method THttpMethod, url string, params interface{}) *JsonBaseRequest {
	return &JsonBaseRequest{
		httpMethod: method,
		url:        url,
		params:     params,
		header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

type JsonResponse interface {
	ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error
}

func (ce *JSONClientError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	body.Unmarshal(ce)
	if ce.Code == 0 {
		ce.Code = statusCode
	}
	if len(ce.Class) == 0 {
		ce.Class = http.StatusText(statusCode)
	}
	if len(ce.Details) == 0 {
		ce.Details = body.String()
	}
	return ce
}

func NewJsonClient(client sClient) *JsonClient {
	return &JsonClient{client: client}
}

func (e *JSONClientError) Error() string {
	errMsg := JSONClientErrorMsg{Error: e}
	return jsonutils.Marshal(errMsg).String()
}

func (err *JSONClientError) Cause() error {
	if len(err.Class) > 0 {
		return errors.Error(err.Class)
	} else if err.Code >= 500 {
		return errors.ErrServer
	} else if err.Code >= 400 {
		return errors.ErrClient
	} else {
		return errors.ErrUnclassified
	}
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

func ErrorMsg(err error) string {
	if err == nil {
		return ""
	}
	switch je := err.(type) {
	case *JSONClientError:
		return je.Details
	}
	return err.Error()
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

func GetTransport(insecure bool) *http.Transport {
	return getTransport(insecure, false, 0)
}

func GetAdaptiveTransport(insecure bool) *http.Transport {
	return getTransport(insecure, true, 0)
}

func adptiveDial(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return getConnDelegate(conn, 10*time.Second, 20*time.Second), nil
}

func getTransport(insecure bool, adaptive bool, timeout time.Duration) *http.Transport {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// 一个空闲连接保持连接的时间
		// IdleConnTimeout is the maximum amount of time an idle
		// (keep-alive) connection will remain idle before closing
		// itself.
		// Zero means no limit.
		IdleConnTimeout: IdleConnTimeout * time.Second,
		// 建立TCP连接后，等待TLS握手的超时时间
		// TLSHandshakeTimeout specifies the maximum amount of time waiting to
		// wait for a TLS handshake. Zero means no timeout.
		TLSHandshakeTimeout: TLSHandshakeTimeout * time.Second,
		// 发送请求后，等待服务端http响应的超时时间
		// ResponseHeaderTimeout, if non-zero, specifies the amount of
		// time to wait for a server's response headers after fully
		// writing the request (including its body, if any). This
		// time does not include the time to read the response body.
		ResponseHeaderTimeout: ResponseHeaderTimeout * time.Second,
		// 当请求携带Expect: 100-continue时，等待服务端100响应的超时时间
		// ExpectContinueTimeout, if non-zero, specifies the amount of
		// time to wait for a server's first response headers after fully
		// writing the request headers if the request has an
		// "Expect: 100-continue" header. Zero means no timeout and
		// causes the body to be sent immediately, without
		// waiting for the server to approve.
		// This time does not include the time to send the request header.
		ExpectContinueTimeout: 5 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
	}
	if adaptive {
		tr.DialContext = adptiveDial
	} else {
		tr.IdleConnTimeout = timeout
		tr.TLSHandshakeTimeout = timeout
		tr.ResponseHeaderTimeout = timeout
		tr.DialContext = (&net.Dialer{
			// 建立TCP连接超时时间
			// Timeout is the maximum amount of time a dial will wait for
			// a connect to complete. If Deadline is also set, it may fail
			// earlier.
			//
			// The default is no timeout.
			//
			// When using TCP and dialing a host name with multiple IP
			// addresses, the timeout may be divided between them.
			//
			// With or without a timeout, the operating system may impose
			// its own earlier timeout. For instance, TCP timeouts are
			// often around 3 minutes.
			Timeout: 10 * time.Second,
			//
			// KeepAlive specifies the interval between keep-alive
			// probes for an active network connection.
			// If zero, keep-alive probes are sent with a default value
			// (currently 15 seconds), if supported by the protocol and operating
			// system. Network protocols or operating systems that do
			// not support keep-alives ignore this field.
			// If negative, keep-alive probes are disabled.
			KeepAlive: 5 * time.Second, // send keep-alive probe every 5 seconds
		}).DialContext
	}
	return tr
}

func GetClient(insecure bool, timeout time.Duration) *http.Client {
	adaptive := false
	if timeout == 0 {
		adaptive = true
	}
	tr := getTransport(insecure, adaptive, timeout)
	return &http.Client{
		Transport: tr,
		// 一个完整http request的超时时间
		// Timeout specifies a time limit for requests made by this
		// Client. The timeout includes connection time, any
		// redirects, and reading the response body. The timer remains
		// running after Get, Head, Post, or Do return and will
		// interrupt reading of the Response.Body.
		//
		// A Timeout of zero means no timeout.
		//
		// The Client cancels requests to the underlying Transport
		// as if the Request's Context ended.
		//
		// For compatibility, the Client will also use the deprecated
		// CancelRequest method on Transport if found. New
		// RoundTripper implementations should use the Request's Context
		// for cancellation instead of implementing CancelRequest.
		Timeout: timeout,
	}
}

type TransportProxyFunc func(*http.Request) (*url.URL, error)

func SetClientProxyFunc(
	client *http.Client,
	proxyFunc TransportProxyFunc,
) bool {
	set := false
	if transport, ok := client.Transport.(*http.Transport); ok {
		transport.Proxy = proxyFunc
		set = true
	}
	return set
}

func GetTimeoutClient(timeout time.Duration) *http.Client {
	return GetClient(true, timeout)
}

func GetAdaptiveTimeoutClient() *http.Client {
	return GetClient(true, 0)
}

var defaultHttpClient *http.Client

func init() {
	defaultHttpClient = GetDefaultClient()
}

func GetDefaultClient() *http.Client {
	return GetClient(true, time.Second*15)
}

func getClientErrorClass(err error) error {
	cause := errors.Cause(err)
	if urlErr, ok := cause.(*url.Error); ok {
		if netErr, ok := urlErr.Err.(*net.OpError); ok {
			switch t := netErr.Err.(type) {
			case *net.DNSError:
				return errors.ErrDNS
			case *os.SyscallError:
				if errno, ok := t.Err.(syscall.Errno); ok {
					switch errno {
					case syscall.ECONNREFUSED:
						return errors.ErrConnectRefused
					case syscall.ETIMEDOUT:
						return errors.ErrTimeout
					}
				}
			}
		}
	}
	return errors.ErrClient
}

func isHTTPReqErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *url.Error:
		switch e.Err.(type) {
		case *net.DNSError, *net.OpError, net.UnknownNetworkError:
			return true
		}
		if strings.Contains(err.Error(), "Connection closed by foreign host") {
			return true
		} else if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
			// If error is - tlsHandshakeTimeoutError, retry.
			return true
		} else if strings.Contains(err.Error(), "i/o timeout") {
			// If error is - tcp timeoutError, retry.
			return true
		} else if strings.Contains(err.Error(), "connection timed out") {
			// If err is a net.Dial timeout, retry.
			return true
		} else if strings.Contains(err.Error(), "net/http: HTTP/1.x transport connection broken") {
			// If error is transport connection broken, retry.
			return true
		} else if strings.Contains(err.Error(), "net/http: timeout awaiting response headers") {
			// Retry errors due to server not sending the response before timeout
			return true
		} else if strings.Contains(err.Error(), "dial tcp: lookup") {
			return true
		}
	}
	return false
}

func Request(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	return request(client, ctx, method, urlStr, header, body, false, debug)
}

func RequestWithRetry(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	return request(client, ctx, method, urlStr, header, body, true, debug)
}

func request(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body io.Reader, retry, debug bool) (*http.Response, error) {
	req, resp, err := requestInternal(client, ctx, method, urlStr, header, body, retry, debug)
	if err != nil {
		var reqBody string
		if bodySeeker, ok := body.(io.ReadSeeker); ok {
			bodySeeker.Seek(0, io.SeekStart)
			reqBodyBytes, _ := ioutil.ReadAll(bodySeeker)
			if reqBodyBytes != nil {
				reqBody = string(reqBodyBytes)
			}
		}
		if req == nil {
			ce := newJsonClientErrorFromRequest2(string(method), urlStr, header, reqBody)
			ce.Class = getClientErrorClass(err).Error()
			ce.Details = err.Error()
			ce.Code = 499
			return nil, ce
		}
		ce := newJsonClientErrorFromRequest(req, reqBody)
		ce.Class = getClientErrorClass(err).Error()
		ce.Details = err.Error()
		ce.Code = 499
		return nil, ce
	}
	return resp, nil
}

func requestInternal(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body io.Reader, retry, debug bool) (*http.Request, *http.Response, error) {
	if client == nil {
		client = defaultHttpClient
	}
	if header == nil {
		header = http.Header{}
	}
	ctxData := appctx.FetchAppContextData(ctx)
	var clientTrace *trace.STrace
	if len(ctxData.ServiceName) > 0 {
		if !ctxData.Trace.IsZero() {
			clientTrace = &ctxData.Trace
		}
		addr, port, err := GetAddrPort(urlStr)
		if err != nil {
			return nil, nil, err
		}
		clientTrace = trace.StartClientTrace(clientTrace, addr, port, ctxData.ServiceName)
		clientTrace.AddClientRequestHeader(header)
	}

	if len(ctxData.RequestId) > 0 {
		header.Set("X-Request-Id", ctxData.RequestId)
	}
	req, err := http.NewRequest(string(method), urlStr, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "*")
	if body == nil {
		if method != GET && method != HEAD {
			req.ContentLength = 0
			req.Header.Set("Content-Length", "0")
		}
	} else {
		clen := header.Get("Content-Length")
		if len(clen) > 0 {
			req.ContentLength, _ = strconv.ParseInt(clen, 10, 64)
		}
	}
	if header != nil {
		for k, vs := range header {
			for i, v := range vs {
				if i == 0 {
					req.Header.Set(k, v)
				} else {
					req.Header.Add(k, v)
				}
			}
		}
	}
	if debug {
		dump, _ := httputil.DumpRequestOut(req, false)
		yellow(string(dump))
		// 忽略掉上传文件的请求,避免大量日志输出
		if header.Get("Content-Type") != "application/octet-stream" {
			curlCmd, _ := http2curl.GetCurlCommand(req)
			cyan("CURL:", curlCmd, "\n")
		}
	}
	resp, err := func() (*http.Response, error) {
		var resp *http.Response
		for i := 0; i < 3; i++ {
			resp, err = client.Do(req)
			if err == nil || !retry || !isHTTPReqErrorRetryable(err) {
				return resp, err
			}
			time.Sleep(time.Second * 5)
		}
		return resp, err
	}()
	if err != nil {
		red(err.Error())
		return req, nil, err
	}
	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "", "identity":
		// do nothing
	case "gzip":
		gzipBody, err := gzip.NewReader(resp.Body)
		if err != nil {
			return req, nil, errors.Wrap(err, "gzip.NewReader")
		}
		resp.Body = gzipBody
	case "deflate":
		resp.Body = flate.NewReader(resp.Body)
	default:
		return req, nil, errors.Wrapf(errors.ErrNotSupported, "unsupported content-encoding %s", encoding)
	}
	if clientTrace != nil {
		clientTrace.EndClientTraceHeader(resp.Header)
	}

	return req, resp, nil
}

func JSONRequestWithRetry(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	return jsonRequest(client, ctx, method, urlStr, header, body, true, debug)
}

func JSONRequest(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	return jsonRequest(client, ctx, method, urlStr, header, body, false, debug)
}

func jsonRequest(client sClient, ctx context.Context, method THttpMethod, urlStr string, header http.Header, body jsonutils.JSONObject, retry, debug bool) (http.Header, jsonutils.JSONObject, error) {
	var bodystr string
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Length", strconv.FormatInt(int64(len(bodystr)), 10))
	header.Set("Content-Type", "application/json")
	resp, err := request(client, ctx, method, urlStr, header, jbody, retry, debug)
	return ParseJSONResponse(bodystr, resp, err, debug)
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

func (client *JsonClient) Send(ctx context.Context, req JsonRequest, response JsonResponse, debug bool) (http.Header, jsonutils.JSONObject, error) {
	var bodystr string
	body := req.GetRequestBody()
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	resp, err := Request(client.client, ctx, req.GetHttpMethod(), req.GetUrl(), req.GetHeader(), jbody, debug)
	if err != nil {
		return nil, nil, err
	}
	defer CloseResponse(resp)
	if debug {
		dump, _ := httputil.DumpResponse(resp, false)
		if resp.StatusCode < 300 {
			green(string(dump))
		} else if resp.StatusCode < 400 {
			yellow(string(dump))
		} else {
			red(string(dump))
		}
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ce := newJsonClientErrorFromRequest(resp.Request, bodystr)
		ce.Code = resp.StatusCode
		ce.Class = string(errors.ErrClient)
		ce.Details = fmt.Sprintf("Fail to read body: %v", err)
		return resp.Header, nil, ce
	} else if debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}

	rbody = bytes.TrimSpace(rbody)

	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 && (rbody[0] == '{' || rbody[0] == '[') {
		var err error
		jrbody, err = jsonutils.Parse(rbody)
		if err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "parsing json %s failed: %v", string(rbody), err)
			}
			ce := newJsonClientErrorFromRequest(resp.Request, bodystr)
			ce.Code = resp.StatusCode
			ce.Class = string(errors.ErrServer)
			ce.Details = fmt.Sprintf("jsonutils.Parse(%s) error: %v", string(rbody), err)
			return resp.Header, nil, ce
		}
	}

	if resp.StatusCode < 300 {
		return resp.Header, jrbody, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return resp.Header, jrbody, &ce
	}

	return resp.Header, jrbody, response.ParseErrorFromJsonResponse(resp.StatusCode, jrbody)
}

func IsRedirectError(err error) bool {
	ce, ok := err.(*JSONClientError)
	if ok && ce.Class == "redirect" {
		return true
	}
	return false
}

func ParseResponse(reqBody string, resp *http.Response, err error, debug bool) (http.Header, []byte, error) {
	if err != nil {
		return nil, nil, err
	}
	defer CloseResponse(resp)
	if debug {
		dump, _ := httputil.DumpResponse(resp, false)
		if resp.StatusCode < 300 {
			green(string(dump))
		} else if resp.StatusCode < 400 {
			yellow(string(dump))
		} else {
			red(string(dump))
		}
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)
		ce.Code = 499
		ce.Details = fmt.Sprintf("Fail to read body: %s", err)
		ce.Class = string(errors.ErrClient)
		return resp.Header, nil, ce
	} else if debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}

	if resp.StatusCode < 300 {
		return resp.Header, rbody, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return resp.Header, rbody, ce
	} else {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)
		ce.Code = resp.StatusCode
		ce.Details = resp.Status
		if len(rbody) > 0 {
			ce.Details = string(rbody)
		}
		return nil, nil, ce
	}
}

func ParseJSONResponse(reqBody string, resp *http.Response, err error, debug bool) (http.Header, jsonutils.JSONObject, error) {
	if err != nil {
		return nil, nil, err
	}
	defer CloseResponse(resp)
	if debug {
		dump, _ := httputil.DumpResponse(resp, false)
		if resp.StatusCode < 300 {
			green(string(dump))
		} else if resp.StatusCode < 400 {
			yellow(string(dump))
		} else {
			red(string(dump))
		}
	}

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)
		ce.Code = 499
		ce.Class = string(errors.ErrClient)
		ce.Details = fmt.Sprintf("Fail to read body: %s", err)
		return resp.Header, nil, ce
	} else if debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}

	rbody = bytes.TrimSpace(rbody)

	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 && (rbody[0] == '{' || rbody[0] == '[') {
		var err error
		jrbody, err = jsonutils.Parse(rbody)
		if err != nil && debug {
			// ignore the error
			fmt.Fprintf(os.Stderr, "parsing json failed: %s", err)
		}
	}

	if resp.StatusCode < 300 {
		return resp.Header, jrbody, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return resp.Header, jrbody, ce
	} else {
		ce := newJsonClientErrorFromRequest(resp.Request, reqBody)

		if jrbody == nil {
			ce.Code = resp.StatusCode
			ce.Details = resp.Status
			if len(rbody) > 0 {
				ce.Details = string(rbody)
			}
			return nil, nil, ce
		}

		err = jrbody.Unmarshal(ce)
		if len(ce.Class) > 0 && ce.Code >= 400 && len(ce.Details) > 0 {
			return nil, nil, ce
		}

		jrbody1, err := jrbody.GetMap()
		if err != nil {
			err = jrbody.Unmarshal(ce)
			if err != nil {
				ce.Details = err.Error()
			}
			return nil, nil, ce
		}
		var jrbody2 jsonutils.JSONObject
		if len(jrbody1) > 1 {
			jrbody2 = jsonutils.Marshal(jrbody1)
		} else {
			for _, v := range jrbody1 {
				jrbody2 = v
			}
		}
		if jrbody2 != nil {
			if ecode, _ := jrbody2.GetString("code"); len(ecode) > 0 {
				code, err := strconv.Atoi(ecode)
				if err != nil {
					ce.Class = ecode
				} else {
					ce.Code = code
				}
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
		return nil, nil, ce
	}
}

func JoinPath(ep string, paths ...string) string {
	buf := strings.Builder{}
	buf.WriteString(strings.TrimRight(ep, "/"))
	for _, path := range paths {
		buf.WriteByte('/')
		buf.WriteString(strings.Trim(path, "/"))
	}
	return buf.String()
}
