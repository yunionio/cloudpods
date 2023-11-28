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

package victoriametrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/influxdata/influxql"
	"golang.org/x/net/context/ctxhttp"
	"moul.io/http2curl/v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	ErrVMInvalidResponse = errors.Error("VictoriaMetrics invalid response")
)

type TimeRange struct {
	Start int64
	End   int64
}

func NewTimeRange(start, end int64) *TimeRange {
	return &TimeRange{
		Start: start,
		End:   end,
	}
}

func NewTimeRangeByInfluxTimeRange(tr *influxql.TimeRange) *TimeRange {
	// format should be: https://docs.victoriametrics.com/#timestamp-formats
	nTr := &TimeRange{}
	if !tr.MinTime().IsZero() {
		nTr.Start = tr.MinTime().Unix()
	}
	if !tr.MaxTime().IsZero() {
		nTr.End = tr.MaxTime().Unix()
	}
	return nTr
}

type Client interface {
	QueryRange(ctx context.Context, httpCli *http.Client, query string, step time.Duration, timeRange *TimeRange, disableCache bool) (*Response, error)
}

type client struct {
	endpoint    string
	endpointURL url.URL
}

func (c *client) getAPIURL(reqPath string) string {
	apiPrefix := "/api/v1"
	reqPath = fmt.Sprintf("%s/%s", apiPrefix, reqPath)
	reqURL := c.endpointURL
	reqURL.Path = path.Join(reqURL.Path, reqPath)
	return reqURL.String()
}

// ResponseDataResultValue likes: [ 1652169600, "1" ]
type ResponseDataResultValue []interface{}

type ResponseDataResult struct {
	Metric map[string]string         `json:"metric"`
	Values []ResponseDataResultValue `json:"values"`
}

type ResponseData struct {
	ResultType string               `json:"resultType"`
	Result     []ResponseDataResult `json:"result"`
}

type ResponseStats struct {
	// SeriesFetched is like integer type: {seriesFetched: "2"}
	SeriesFetched string `json:"seriesFetched"`
}

type Response struct {
	Status string `json:"status"`
	Data   ResponseData
	Stats  ResponseStats
}

// QueryRange implements Client.
func (c *client) QueryRange(ctx context.Context, httpCli *http.Client, query string, step time.Duration, tr *TimeRange, disableCache bool) (*Response, error) {
	req, err := c.createQueryRangeReq(query, step, tr, disableCache)
	if err != nil {
		return nil, errors.Wrap(err, "get request")
	}

	resp, err := ctxhttp.Do(ctx, httpCli, req)
	if err != nil {
		return nil, errors.Wrap(err, "Do request")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, errors.Wrapf(ErrVMInvalidResponse, "status code: %d", resp.StatusCode)
	}

	var response Response
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&response); err != nil {
		return nil, errors.Wrap(err, "decode json response")
	}
	return &response, nil
}

func (c *client) createQueryRangeReq(query string, step time.Duration, tr *TimeRange, disableCache bool) (*http.Request, error) {
	reqURL := c.getAPIURL("/query_range")
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "new HTTP request of: %s", reqURL)
	}
	req.Header.Set("User-Agent", "Cloudpods Monitor Service")
	params := req.URL.Query()
	params.Set("query", query)
	if step != 0 {
		params.Set("step", step.String())
	}
	if tr != nil {
		if tr.Start != 0 {
			params.Set("start", fmt.Sprintf("%d", tr.Start))
		}
		if tr.End != 0 {
			params.Set("end", fmt.Sprintf("%d", tr.End))
		}
	}
	if disableCache {
		params.Set("nocache", "1")
	}
	req.URL.RawQuery = params.Encode()
	curlCmd, _ := http2curl.GetCurlCommand(req)
	log.Infof("VictoriaMetrics curl cmd: %s", curlCmd)
	return req, nil
}

func NewClient(endpoint string) (Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid url: %q", endpoint)
	}
	cli := &client{
		endpoint:    endpoint,
		endpointURL: *u,
	}
	return cli, nil
}
