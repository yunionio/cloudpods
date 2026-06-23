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

package http

import (
	"context"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/onecloud/pkg/util/probe"
)

func New() Prober {
	return httpProber{}
}

func NewWithDialContext(dialContext DialContextFunc) Prober {
	return httpProber{dialContext: dialContext}
}

type DialContextFunc func(context.Context, string, string) (net.Conn, error)

type Prober interface {
	Probe(scheme string, host string, port int, reqPath string, headers nethttp.Header, timeout time.Duration) (probe.Result, string, error)
}

type httpProber struct {
	dialContext DialContextFunc
}

func (pr httpProber) Probe(scheme string, host string, port int, reqPath string, headers nethttp.Header, timeout time.Duration) (probe.Result, string, error) {
	reqURL, err := buildProbeURL(scheme, host, port, reqPath)
	if err != nil {
		return probe.Failure, err.Error(), nil
	}
	return DoHTTPProbeWithDialContext(reqURL, headers, timeout, pr.dialContext)
}

func buildProbeURL(scheme string, host string, port int, reqPath string) (string, error) {
	normalizedScheme := strings.ToLower(strings.TrimSpace(scheme))
	if normalizedScheme == "" {
		normalizedScheme = "http"
	}
	switch normalizedScheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("unsupported HTTP probe scheme %q", scheme)
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("HTTP probe host is empty")
	}

	if reqPath == "" {
		reqPath = "/"
	} else if !strings.HasPrefix(reqPath, "/") {
		reqPath = "/" + reqPath
	}

	u := url.URL{
		Scheme: normalizedScheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   reqPath,
	}
	return u.String(), nil
}

func DoHTTPProbe(reqURL string, headers nethttp.Header, timeout time.Duration) (probe.Result, string, error) {
	return DoHTTPProbeWithDialContext(reqURL, headers, timeout, nil)
}

func DoHTTPProbeWithDialContext(reqURL string, headers nethttp.Header, timeout time.Duration, dialContext DialContextFunc) (probe.Result, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, reqURL, nil)
	if err != nil {
		return probe.Failure, err.Error(), nil
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	transport := nethttp.DefaultTransport.(*nethttp.Transport).Clone()
	if dialContext != nil {
		transport.DialContext = dialContext
	}
	defer transport.CloseIdleConnections()

	client := &nethttp.Client{Timeout: timeout, Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return probe.Failure, err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= nethttp.StatusOK && resp.StatusCode < nethttp.StatusBadRequest {
		return probe.Success, resp.Status, nil
	}
	return probe.Failure, resp.Status, nil
}
