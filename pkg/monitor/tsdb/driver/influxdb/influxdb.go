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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/context/ctxhttp"
	"moul.io/http2curl/v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	mod "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

const (
	ErrInfluxdbInvalidResponse = errors.Error("Influxdb invalid status")
)

func init() {
	tsdb.RegisterTsdbQueryEndpoint(monitor.DataSourceTypeInfluxdb, NewInfluxdbExecutor)
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

func (e *InfluxdbExecutor) GetRawQuery(dsInfo *tsdb.DataSource, tsdbQuery *tsdb.TsdbQuery) (string, []*Query, error) {
	querys := make([]*tsdb.Query, len(tsdbQuery.Queries)+1)
	influxQ := make([]*Query, 0)
	copy(querys, tsdbQuery.Queries)
	var buffer bytes.Buffer
	var rawQuery string
	for i := 0; i < len(querys); i++ {
		query, err := e.getQuery(dsInfo, querys, tsdbQuery)
		if err != nil {
			return "", nil, errors.Wrap(err, "getQuery")
		}
		influxQ = append(influxQ, query)

		rawQuery, err := query.Build(tsdbQuery)
		if err != nil {
			return "", nil, errors.Wrap(err, "query.Build")
		}
		buffer.WriteString(rawQuery + ";")
		if len(querys) > 0 {
			querys = querys[1:]
		}
	}
	rawQuery = buffer.String()
	spitCount := strings.Count(rawQuery, ";")
	rawQuery = strings.Replace(rawQuery, ";", "", spitCount)
	//query, err := e.getQuery(dsInfo, tsdbQuery.Queries, tsdbQuery)
	//if err != nil {
	//	return nil, err
	//}
	//
	//rawQuery, err := query.Build(tsdbQuery)
	//if err != nil {
	//	return nil, err
	//}
	return rawQuery, influxQ, nil
}

func (e *InfluxdbExecutor) Query(ctx context.Context, dsInfo *tsdb.DataSource, tsdbQuery *tsdb.TsdbQuery) (*tsdb.Response, error) {
	rawQuery, influxQ, err := e.GetRawQuery(dsInfo, tsdbQuery)
	if err != nil {
		return nil, errors.Wrap(err, "GetRawQuery")
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

	result := &tsdb.Response{
		Results: make(map[string]*tsdb.QueryResult),
	}
	for i, query := range tsdbQuery.Queries {
		ret := e.ResponseParser.Parse(&response, influxQ[i])
		ret.Meta = monitor.QueryResultMeta{
			RawQuery: rawQuery,
		}
		result.Results[query.RefId] = ret
	}
	//ret := e.ResponseParser.Parse(&response, query)
	//ret.Meta = tsdb.QueryResultMeta{
	//	RawQuery: rawQuery,
	//}
	//result.Results["A"] = ret

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
	params.Set("epoch", "ms")

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

func (e *InfluxdbExecutor) FilterMeasurement(
	ctx context.Context,
	ds *tsdb.DataSource,
	from, to string,
	ms *monitor.InfluxMeasurement,
	tagFilter *monitor.MetricQueryTag,
) (*monitor.InfluxMeasurement, error) {
	retMs := new(monitor.InfluxMeasurement)
	q := mod.NewAlertQuery(ms.Database, ms.Measurement).From(from).To(to)
	q.Selects().Select("*").LAST()
	if tagFilter != nil {
		q.Where().AddTag(tagFilter)
	}
	tq := q.ToTsdbQuery()
	resp, err := e.Query(ctx, ds, tq)
	if err != nil {
		return nil, errors.Wrap(err, "influxdb.Query")
	}
	ss := resp.Results[""].Series
	//log.Infof("=====get ss: %s", jsonutils.Marshal(ss).PrettyString())

	// parse fields
	retFields := make([]string, 0)
	for _, s := range ss {
		cols := s.Columns
		for _, col := range cols {
			if !strings.Contains(col, "last") {
				continue
			}
			retFields = append(retFields, strings.Replace(col, "last_", "", 1))
		}
	}
	retMs.FieldKey = retFields
	if len(retMs.FieldKey) != 0 {
		retMs.Measurement = ms.Measurement
		retMs.Database = ms.Database
		retMs.ResType = ms.ResType
	}

	return retMs, nil
}

func FillSelectWithMean(query *monitor.AlertQuery) *monitor.AlertQuery {
	for i, sel := range query.Model.Selects {
		if len(sel) > 1 {
			continue
		}
		sel = append(sel, monitor.MetricQueryPart{
			Type:   "mean",
			Params: []string{},
		})
		query.Model.Selects[i] = sel
	}
	return query
}

func (e *InfluxdbExecutor) FillSelect(query *monitor.AlertQuery, isAlert bool) *monitor.AlertQuery {
	return FillSelectWithMean(query)
}

func FillGroupByWithWildChar(query *monitor.AlertQuery, inputQuery *monitor.MetricQueryInput, tagId string) *monitor.AlertQuery {
	if len(tagId) == 0 || (len(inputQuery.Slimit) != 0 && len(inputQuery.Soffset) != 0) {
		tagId = "*"
	}
	if tagId != "" {
		query.Model.GroupBy = append(query.Model.GroupBy,
			monitor.MetricQueryPart{
				Type:   "field",
				Params: []string{tagId},
			})
	}
	return query
}

func (e *InfluxdbExecutor) FillGroupBy(query *monitor.AlertQuery, inputQuery *monitor.MetricQueryInput, tagId string, isAlert bool) *monitor.AlertQuery {
	return FillGroupByWithWildChar(query, inputQuery, tagId)
}
