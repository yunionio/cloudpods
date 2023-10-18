package tos

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type urlMode int

const (
	// urlModePath url pattern is http(s)://{bucket}.domain/{object}
	urlModeDefault = 0
	// urlModePath url pattern is http(s)://domain/{bucket}/{object}
	urlModePath = 1
)

type Request struct {
	Scheme        string
	Method        string
	Host          string
	Path          string
	ContentLength *int64
	Content       io.Reader
	Query         url.Values
	Header        http.Header
}

func (req *Request) URL() string {
	u := url.URL{
		Scheme:   req.Scheme,
		Host:     req.Host,
		Path:     req.Path,
		RawQuery: req.Query.Encode(),
	}
	return u.String()
}

func OnRetryFromStart(req *Request) error {
	if seek, ok := req.Content.(io.Seeker); ok {
		_, err := seek.Seek(0, io.SeekStart)
		return err
	}
	return nil
}

// Range represents a range of an object
type Range struct {
	Start int64
	End   int64
}

// HTTP Range header
func (hr *Range) String() string {
	return fmt.Sprintf("bytes=%d-%d", hr.Start, hr.End)
}

type CopySource struct {
	srcBucket    string
	srcObjectKey string
}

type requestBuilder struct {
	Signer         Signer
	Scheme         string
	Host           string
	Bucket         string
	Object         string
	URLMode        urlMode
	ContentLength  *int64
	Range          *Range
	Query          url.Values
	Header         http.Header
	Retry          *retryer
	OnRetry        func(req *Request) error
	Classifier     classifier
	CopySource     *CopySource
	IsCustomDomain bool
	// CheckETag  bool
	// CheckCRC32 bool
}

func (rb *requestBuilder) WithRetry(onRetry func(req *Request) error, classifier classifier) *requestBuilder {
	if onRetry == nil {
		rb.OnRetry = func(req *Request) error { return nil }
	} else {
		rb.OnRetry = onRetry
	}
	if classifier == nil {
		rb.Classifier = NoRetryClassifier{}
	} else {
		rb.Classifier = classifier
	}
	return rb
}

func (rb *requestBuilder) WithCopySource(srcBucket, srcObjectKey string) *requestBuilder {
	rb.CopySource = &CopySource{
		srcBucket:    srcBucket,
		srcObjectKey: srcObjectKey,
	}
	return rb
}

func (rb *requestBuilder) WithQuery(key, value string) *requestBuilder {
	rb.Query.Add(key, value)
	return rb
}

func (rb *requestBuilder) WithHeader(key, value string) *requestBuilder {
	if len(value) > 0 {
		rb.Header.Set(key, value)
	}
	return rb
}

func convertToString(iface interface{}, tag *reflect.StructTag) string {
	// return empty string if value is zero except filed with "default" tag
	var result string
	switch v := iface.(type) {
	case string:
		result = v
	case int:
		if v != 0 {
			result = strconv.Itoa(v)
		} else {
			result = tag.Get("default")
		}
	case int64:
		if v != 0 {
			result = strconv.Itoa(int(v))
		} else {
			result = tag.Get("default")
		}
	case time.Time:
		if !v.IsZero() {
			result = v.Format(http.TimeFormat)
		}
	case bool:
		result = strconv.FormatBool(v)
	default:
		if reflect.TypeOf(iface).Kind() == reflect.String {
			result = reflect.ValueOf(iface).String()
		}
	}
	return result
}

// WithParams will set filed with tag "header" in input to rb.Header.
func (rb *requestBuilder) WithParams(input interface{}) *requestBuilder {

	t := reflect.TypeOf(input)
	v := reflect.ValueOf(input)
	for i := 0; i < v.NumField(); i++ {
		filed := t.Field(i)
		if filed.Type.Kind() == reflect.Struct {
			rb.WithParams(v.Field(i).Interface())
		}
		location := filed.Tag.Get("location")
		switch location {
		case "header":
			value := convertToString(v.Field(i).Interface(), &filed.Tag)
			if filed.Tag.Get("encodeChinese") == "true" {
				value = headerEncode(value)
			}
			rb.WithHeader(filed.Tag.Get("locationName"), value)
		case "headers":
			if headers, ok := v.Field(i).Interface().(map[string]string); ok {
				for k, v := range headers {
					rb.Header.Set(HeaderMetaPrefix+headerEncode(k), headerEncode(v))
				}
				return rb
			}
		case "query":
			v := convertToString(v.Field(i).Interface(), &filed.Tag)
			if len(v) > 0 {
				rb.WithQuery(filed.Tag.Get("locationName"), v)
			}
		}
	}
	return rb
}

func (rb *requestBuilder) WithContentLength(length int64) *requestBuilder {
	rb.ContentLength = &length
	return rb
}

func (rb *requestBuilder) hostPath() (string, string) {

	if rb.IsCustomDomain {
		if len(rb.Object) > 0 {
			return rb.Host, "/" + rb.Object
		}
		return rb.Host, "/"
	}

	if rb.URLMode == urlModePath {
		if len(rb.Object) > 0 {
			return rb.Host, "/" + rb.Bucket + "/" + rb.Object
		}
		return rb.Host, "/" + rb.Bucket // rb.Bucket may be empty ""
	}
	// URLModeDefault
	if len(rb.Bucket) == 0 {
		return rb.Host, "/"
	}
	return rb.Bucket + "." + rb.Host, "/" + rb.Object
}

func (rb *requestBuilder) build(method string, content io.Reader) *Request {
	host, path := rb.hostPath()
	req := &Request{
		Scheme:  rb.Scheme,
		Method:  method,
		Host:    host,
		Path:    path,
		Content: content,
		Query:   rb.Query,
		Header:  rb.Header,
	}

	if content != nil {
		if rb.ContentLength != nil {
			req.ContentLength = rb.ContentLength
		} else if length := tryResolveLength(content); length >= 0 {
			req.ContentLength = &length
		}
	}
	return req
}

func (rb *requestBuilder) Build(method string, content io.Reader) *Request {
	req := rb.build(method, content)
	if rb.CopySource != nil {
		versionID := req.Query.Get("versionId")
		req.Query.Del("versionId")
		req.Header.Add(HeaderCopySource, copySource(rb.CopySource.srcBucket, rb.CopySource.srcObjectKey, versionID))
	}
	if rb.Signer != nil {
		signed := rb.Signer.SignHeader(req)
		for key, values := range signed {
			req.Header[key] = values
		}
	}
	return req
}

type roundTripper func(ctx context.Context, req *Request) (*Response, error)

func (rb *requestBuilder) Request(ctx context.Context, method string,
	content io.Reader, roundTripper roundTripper) (*Response, error) {

	var (
		req *Request
		res *Response
		err error
	)

	req = rb.Build(method, content)

	if rb.Retry != nil {
		work := func() (err error) {
			err = rb.OnRetry(req)
			if err != nil {
				return err
			}
			res, err = roundTripper(ctx, req)
			return err
		}
		err = rb.Retry.Run(ctx, work, rb.Classifier)
		if err != nil {
			return nil, err
		}
		return res, err
	}
	res, err = roundTripper(ctx, req)

	return res, err
}

func (rb *requestBuilder) PreSignedURL(method string, ttl time.Duration) (string, error) {
	req := rb.build(method, nil)
	if rb.Signer == nil {
		return "", errors.New("tos: credentials is not set when the tos.Client was created")
	}

	query := rb.Signer.SignQuery(req, ttl)
	for k, v := range query {
		req.Query[k] = v
	}
	return req.URL(), nil
}

type RequestInfo struct {
	RequestID  string
	ID2        string
	StatusCode int
	Header     http.Header
}

type Response struct {
	StatusCode    int
	ContentLength int64
	Header        http.Header
	Body          io.ReadCloser
}

func (r *Response) RequestInfo() RequestInfo {
	return RequestInfo{
		RequestID:  r.Header.Get(HeaderRequestID),
		ID2:        r.Header.Get(HeaderID2),
		StatusCode: r.StatusCode,
		Header:     r.Header,
	}
}

func (r *Response) Close() error {
	if r.Body != nil {
		return r.Body.Close()
	}
	return nil
}

func marshalOutput(requestID string, reader io.Reader, output interface{}) error {
	// Although status code is ok, we need to check if response body is valid.
	// If response body is invalid, TosServerError should be raised. But we can't
	// unmarshal error from response body now.
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return &TosServerError{
			TosError:    TosError{Message: "tos: unmarshal response body failed."},
			RequestInfo: RequestInfo{RequestID: requestID},
		}
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return &TosServerError{
			TosError:    TosError{Message: "server returns empty result"},
			RequestInfo: RequestInfo{RequestID: requestID},
		}
	}
	if err = json.Unmarshal(data, output); err != nil {
		return &TosServerError{
			TosError:    TosError{Message: err.Error()},
			RequestInfo: RequestInfo{RequestID: requestID},
		}
	}
	return nil
}

func marshalInput(name string, input interface{}) ([]byte, string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, "", InvalidMarshal
	}

	sum := md5.Sum(data)
	return data, base64.StdEncoding.EncodeToString(sum[:]), nil
}

func fileUnreadLength(file *os.File) (int64, error) {
	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	size := stat.Size()
	if offset > size || offset < 0 {
		return 0, newTosClientError("tos: unexpected file size and(or) offset", nil)
	}

	return size - offset, nil
}

func tryResolveLength(reader io.Reader) int64 {
	switch v := reader.(type) {
	case *bytes.Buffer:
		return int64(v.Len())
	case *bytes.Reader:
		return int64(v.Len())
	case *strings.Reader:
		return int64(v.Len())
	case *os.File:
		length, err := fileUnreadLength(v)
		if err != nil {
			return -1
		}
		return length
	case *io.LimitedReader:
		return v.N
	case *net.Buffers:
		if v != nil {
			length := int64(0)
			for _, p := range *v {
				length += int64(len(p))
			}
			return length
		}
		return 0
	default:
		return -1
	}
}

func Int64(value int64) *int64 { return &value }
