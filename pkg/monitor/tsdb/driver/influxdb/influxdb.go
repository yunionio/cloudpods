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

package influxdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/moul/http2curl"
	"golang.org/x/net/context/ctxhttp"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

const (
	ErrInfluxdbInvalidResponse = errors.Error("Influxdb invalid status")
)

func init() {
	tsdb.RegisterTsdbQueryEndpoint("influxdb", NewInfluxdbExecutor)
}

type InfluxdbExecutor struct {
	QueryParser    *InfluxdbQueryParser
	ResponseParser *ResponseParser
}

func NewInfluxdbExecutor(datasource *tsdb.DataSource) (tsdb.TsdbQueryEndpoint, error) {
	return &InfluxdbExecutor{
		QueryParser:    &InfluxdbQueryParser{},
		ResponseParser: &ResponseParser{},
	}, nil
}

func (e *InfluxdbExecutor) Query(ctx context.Context, dsInfo *tsdb.DataSource, tsdbQuery *tsdb.TsdbQuery) (*tsdb.Response, error) {
	result := &tsdb.Response{}

	query, err := e.getQuery(dsInfo, tsdbQuery.Queries, tsdbQuery)
	if err != nil {
		return nil, err
	}

	rawQuery, err := query.Build(tsdbQuery)
	if err != nil {
		return nil, err
	}

	db := dsInfo.Database
	if db == "" {
		db = tsdbQuery.Queries[0].Database
	}
	dsInfo.Database = db

	req, err := e.createRequest(dsInfo, rawQuery)
	if err != nil {
		return nil, err
	}

	httpClient, err := dsInfo.GetHttpClient()
	if err != nil {
		return nil, err
	}

	resp, err := ctxhttp.Do(ctx, httpClient, req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		// TODO: convert status code err
		return nil, errors.Wrapf(ErrInfluxdbInvalidResponse, "status code: %v", resp.Status)
	}

	var response Response
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&response); err != nil {
		return nil, err
	}

	if response.Err != nil {
		return nil, response.Err
	}

	result.Results = make(map[string]*tsdb.QueryResult)
	ret := e.ResponseParser.Parse(&response, query)
	ret.Meta = tsdb.QueryResultMeta{
		RawQuery: rawQuery,
	}
	result.Results["A"] = ret

	return result, nil
}

func (e *InfluxdbExecutor) getQuery(dsInfo *tsdb.DataSource, queries []*tsdb.Query, context *tsdb.TsdbQuery) (*Query, error) {
	// The model supports multiple queries, but right now this is only used from
	// alerting so we only need to support batch executing 1 query at a time.
	if len(queries) > 0 {
		query, err := e.QueryParser.Parse(queries[0], dsInfo)
		if err != nil {
			return nil, err
		}
		return query, nil
	}
	return nil, errors.Error("query request contains no queries")
}

func (e *InfluxdbExecutor) createRequest(dsInfo *tsdb.DataSource, query string) (*http.Request, error) {
	u, _ := url.Parse(dsInfo.Url)
	u.Path = path.Join(u.Path, "query")
	req, err := func() (*http.Request, error) {
		// use POST mode
		bodyValues := url.Values{}
		bodyValues.Add("q", query)
		body := bodyValues.Encode()
		return http.NewRequest(http.MethodPost, u.String(), strings.NewReader(body))
	}()

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "OneCloud Monitor")

	params := req.URL.Query()
	params.Set("db", dsInfo.Database)
	params.Set("epoch", "s")

	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	req.URL.RawQuery = params.Encode()

	/*if dsInfo.BasicAuth {
		req.SetBasicAuth(dsinfo.BasicAuthUser, dsInfo.DecryptedBasicAuthPassword())
	}

	if !dsInfo.BasicAuth && dsInfo.User != "" {
		req.SetBasicAuth(dsInfo.User, dsInfo.DecryptedPassword())
	}*/
	curlCmd, _ := http2curl.GetCurlCommand(req)
	log.Debugf("Influxdb raw query: %q from db %s, curl: %s", query, dsInfo.Database, curlCmd)
	return req, nil
}
