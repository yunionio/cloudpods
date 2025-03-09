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

package stream

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/http2"
	"k8s.io/apimachinery/pkg/util/net"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
)

type Request struct {
	body    io.Reader
	headers http.Header
	client  *http.Client
}

func NewRequest(client *http.Client, body io.Reader, headers http.Header) *Request {
	return &Request{
		body:    body,
		headers: headers,
		client:  client,
	}
}

func (r *Request) Stream(ctx context.Context, method string, url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	if r.body != nil {
		req.Body = ioutil.NopCloser(r.body)
	}
	req = req.WithContext(ctx)
	req.Header = r.headers
	client := r.client
	if client == nil {
		client = httputils.GetTimeoutClient(1 * time.Hour)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	switch {
	case (resp.StatusCode >= 200) && (resp.StatusCode < 300):
		return resp.Body, nil

	default:
		// ensure we close the body before returning the error
		defer resp.Body.Close()

		result := r.transformResponse(resp, req)
		err := result.Error()
		if err == nil {
			err = fmt.Errorf("%d while accessing %v: %s", result.statusCode, url, string(result.body))
		}
		return nil, err
	}
}

// transformResponse converts an API response into a structured API object
func (r *Request) transformResponse(resp *http.Response, req *http.Request) Result {
	var body []byte
	if resp.Body != nil {
		data, err := ioutil.ReadAll(resp.Body)
		switch err.(type) {
		case nil:
			body = data
		case http2.StreamError:
			// This is trying to catch the scenario that the server may close the connection when sending the
			// response body. This can be caused by server timeout due to a slow network connection.
			// TODO: Add test for this. Steps may be:
			// 1. client-go (or kubectl) sends a GET request.
			// 2. Apiserver sends back the headers and then part of the body
			// 3. Apiserver closes connection.
			// 4. client-go should catch this and return an error.
			log.Infof("Stream error %#v when reading response body, may be caused by closed connection.", err)
			streamErr := fmt.Errorf("stream error when reading response body, may be caused by closed connection. Please retry. Original error: %v", err)
			return Result{
				err: streamErr,
			}
		default:
			log.Errorf("Unexpected error when reading response body: %v", err)
			unexpectedErr := fmt.Errorf("unexpected error when reading response body. Please retry. Original error: %v", err)
			return Result{
				err: unexpectedErr,
			}
		}
	}

	contentType := resp.Header.Get("Content-Type")

	switch {
	case resp.StatusCode == http.StatusSwitchingProtocols:
		// no-op, we've been upgraded
	case resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusPartialContent:
		// calculate an unstructured error from the response which the Result object may use if the caller
		// did not return a structured error.
		// retryAfter, _ := retryAfterSeconds(resp)
		// err := r.newUnstructuredResponseError(body, isTextResponse(resp), resp.StatusCode, req.Method, retryAfter)
		return Result{
			body:        body,
			contentType: contentType,
			statusCode:  resp.StatusCode,
			err:         nil,
		}
	}

	return Result{
		body:        body,
		contentType: contentType,
		statusCode:  resp.StatusCode,
	}
}

// retryAfterSeconds returns the value of the Retry-After header and true, or 0 and false if
// the header was missing or not a valid number.
func retryAfterSeconds(resp *http.Response) (int, bool) {
	if h := resp.Header.Get("Retry-After"); len(h) > 0 {
		if i, err := strconv.Atoi(h); err == nil {
			return i, true
		}
	}
	return 0, false
}

// Result contains the result of calling Request.Do().
type Result struct {
	body        []byte
	warnings    []net.WarningHeader
	contentType string
	err         error
	statusCode  int
}

// Raw returns the raw result.
func (r Result) Raw() ([]byte, error) {
	return r.body, r.err
}

func (r Result) Error() error {
	return r.err
}
