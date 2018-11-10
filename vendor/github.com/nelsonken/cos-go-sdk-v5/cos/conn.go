package cos

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Conn http 请求类
type Conn struct {
	c    *http.Client
	conf *Conf
}

func (conn *Conn) Do(ctx context.Context, method, bucket, object string, params map[string]interface{}, headers map[string]string, body io.Reader) (*http.Response, error) {
	queryStr := getQueryStr(params)
	url := conn.buildURL(bucket, object, queryStr)

	switch body.(type) {
	case *bytes.Buffer, *bytes.Reader, *strings.Reader:
	default:
		if body != nil {
			b, err := ioutil.ReadAll(body)
			if err != nil {
				return nil, err
			}
			body = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	conn.signHeader(req, params, headers)
	req.Header.Set("User-Agent", conn.conf.UA)
	setHeader(req, headers)

	res, err := conn.c.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer res.Body.Close()
		return checkHTTPErr(res)
	}

	return res, nil
}

func getQueryStr(params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return ""
	}

	buf := new(bytes.Buffer)
	buf.WriteString("?")
	for k, v := range params {
		buf.WriteString(k)
		vs := interfaceToString(v)
		if vs == "" {
			buf.WriteString("&")
			continue
		}
		buf.WriteString("=")
		buf.WriteString(vs)
		buf.WriteString("&")
	}

	return strings.Trim(buf.String(), "&")
}

func (conn *Conn) buildURL(bucket, object, queryStr string) string {
	domain := fmt.Sprintf("%s-%s.cos.%s.%s", bucket, conn.conf.AppID, conn.conf.Region, conn.conf.Domain)
	url := fmt.Sprintf("http://%s/%s%s", domain, escape(object), queryStr)

	return url
}

func escape(str string) string {
	//go语言中将空格编码为+，需要改为%20
	return strings.Replace(url.QueryEscape(str), "+", "%20", -1)
}

func setHeader(req *http.Request, headers map[string]string) {
	if headers == nil {
		return
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func checkHTTPErr(res *http.Response) (*http.Response, error) {
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	err := HTTPError{}
	err.Code = res.StatusCode
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		err.Message = "资源被重定向"
	}

	if res.StatusCode >= 400 && res.StatusCode < 500 {
		err.Message = "请求被拒绝"
	}

	if res.StatusCode >= 500 {
		err.Message = "cos服务器错误"
	}

	if res.ContentLength > 0 {
		resErr := &Error{}
		e := XMLDecode(res.Body, resErr)
		if e != nil {
			return nil, err
		}
		err.Message += resErr.Message
	}

	return res, err
}

// XMLDecode xml解析方法
func XMLDecode(r io.Reader, i interface{}) error {
	jd := xml.NewDecoder(r)
	err := jd.Decode(i)

	return err
}
