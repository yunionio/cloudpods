package httputils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/moul/http2curl"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/appctx"
	"github.com/yunionio/pkg/trace"
)

const (
	USER_AGENT = "yunioncloud-go/201708"
)

var (
	red    = color.New(color.FgRed, color.Bold).PrintlnFunc()
	green  = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	yellow = color.New(color.FgYellow, color.Bold).PrintlnFunc()
	cyan   = color.New(color.FgHiCyan, color.Bold).PrintlnFunc()
)

type JSONClientError struct {
	Code    int
	Class   string
	Details string
}

func (e *JSONClientError) Error() string {
	return fmt.Sprintf("JSONClientError: %s %d %s", e.Details, e.Code, e.Class)
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

func GetClient(insecure bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Transport: tr}
}

var defaultHttpClient *http.Client

func init() {
	defaultHttpClient = GetClient(true)
}

func GetDefaultClient() *http.Client {
	return defaultHttpClient
}

func Request(client *http.Client, ctx context.Context, method string, urlStr string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
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
	method = strings.ToUpper(method)
	req, err := http.NewRequest(method, urlStr, body)
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

func JSONRequest(client *http.Client, ctx context.Context, method string, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	bodystr := ""
	if body != nil {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	header.Add("Content-Type", "application/json")
	resp, err := Request(client, ctx, method, urlStr, header, jbody, debug)
	return ParseJSONResponse(resp, err, debug)
}

func ParseJSONResponse(resp *http.Response, err error, debug bool) (http.Header, jsonutils.JSONObject, error) {
	if err != nil {
		ce := JSONClientError{}
		ce.Code = 499
		ce.Details = err.Error()
		return nil, nil, &ce
	}
	defer resp.Body.Close()
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
		fmt.Println(string(rbody))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("Fail to read body: %s", err)
	}
	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 {
		jrbody, err = jsonutils.Parse(rbody)

		if err != nil && debug {
			log.Errorf("parse JSON body %s fail: %s", rbody, err)
		}
		///// XXX: ignore error case
		// if err != nil && resp.StatusCode < 300 {
		//     return nil, nil, fmt.Errorf("Fail to decode body: %s", err)
		// }
		if jrbody != nil && debug {
			fmt.Println(jrbody)
		}
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return nil, nil, &ce
	} else if resp.StatusCode >= 400 {
		ce := JSONClientError{}
		if jrbody == nil {
			ce.Code = resp.StatusCode
			ce.Details = resp.Status
			return nil, nil, &ce
		} else {
			jrbody2, e := jrbody.Get("error")
			if e == nil {
				ecode, e := jrbody2.Int("code")
				if e == nil {
					ce.Code = int(ecode)
					ce.Details, _ = jrbody2.GetString("message")
					ce.Class, _ = jrbody2.GetString("title")
					return nil, nil, &ce
				} else {
					ce.Code = resp.StatusCode
					ce.Details = jrbody2.String()
					return nil, nil, &ce
				}
			} else {
				err = jrbody.Unmarshal(&ce)
				if err != nil {
					return nil, nil, err
				} else {
					return nil, nil, &ce
				}
			}
		}
	} else {
		return resp.Header, jrbody, nil
	}
}
