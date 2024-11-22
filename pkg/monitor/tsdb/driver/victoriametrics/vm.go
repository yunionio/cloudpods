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
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/influxql"
	"github.com/influxdata/promql/v2/pkg/labels"
	"github.com/zexi/influxql-to-metricsql/converter"
	"github.com/zexi/influxql-to-metricsql/converter/translator"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	mod "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/tsdb/driver/influxdb"
)

func init() {
	tsdb.RegisterTsdbQueryEndpoint(monitor.DataSourceTypeVictoriaMetrics, NewVMAdapter)
}

type vmAdapter struct {
	datasource       *tsdb.DataSource
	influxdbExecutor *influxdb.InfluxdbExecutor
}

func NewVMAdapter(datasource *tsdb.DataSource) (tsdb.TsdbQueryEndpoint, error) {
	ie, _ := influxdb.NewInfluxdbExecutor(nil)
	return &vmAdapter{
		datasource:       datasource,
		influxdbExecutor: ie.(*influxdb.InfluxdbExecutor),
	}, nil
}

// Query implements tsdb.TsdbQueryEndpoint.
func (vm *vmAdapter) Query(ctx context.Context, ds *tsdb.DataSource, query *tsdb.TsdbQuery) (*tsdb.Response, error) {
	rawQuery, influxQs, err := vm.influxdbExecutor.GetRawQuery(ds, query)
	if err != nil {
		return nil, errors.Wrapf(err, "get influxdb raw query: %#v", influxQs)
	}

	// TODO: use interval inside query
	return queryByRaw(ctx, ds, rawQuery, query, influxQs[0].Interval)
}

func queryByRaw(ctx context.Context, ds *tsdb.DataSource, rawQuery string, query *tsdb.TsdbQuery, interval time.Duration) (*tsdb.Response, error) {
	promQL, tr, err := convertInfluxQL(rawQuery)
	if err != nil {
		return nil, errors.Wrap(err, "convert influxQL to promQL")
	}

	start := time.Now()
	defer func() {
		log.Infof("influxQL: %s, promQL: %s, elapsed: %s", rawQuery, promQL, time.Now().Sub(start))
	}()

	resp, err := queryRange(ctx, ds, tr, promQL, interval)
	if err != nil {
		return nil, errors.Wrapf(err, "query VM range by: %s", promQL)
	}

	//log.Infof("get vm resp: %s", jsonutils.Marshal(resp).PrettyString())

	tsdbRet, err := convertVMResponse(rawQuery, query, resp)
	if err != nil {
		return nil, errors.Wrapf(err, "convert to tsdb.Response")
	}

	return tsdbRet, nil
}

func queryRange(ctx context.Context, ds *tsdb.DataSource, tr *influxql.TimeRange, promQL string, interval time.Duration) (*Response, error) {
	cli, err := NewClient(ds.Url)
	if err != nil {
		return nil, errors.Wrap(err, "New VM client")
	}
	httpCli, err := ds.GetHttpClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetHttpClient of data source")
	}
	vmTr := NewTimeRangeByInfluxTimeRange(tr)
	if interval <= 0 || interval < 1*time.Minute {
		interval = time.Minute * 5
	}
	return cli.QueryRange(ctx, httpCli, promQL, interval, vmTr, false)
}

func convertInfluxQL(influxQL string) (string, *influxql.TimeRange, error) {
	promQL, timeRange, err := converter.TranslateWithTimeRange(influxQL)
	if err != nil {
		return "", nil, errors.Wrapf(err, "TranslateWithTimeRange: %s", influxQL)
	}
	return promQL, timeRange, nil
}

func convertVMResponse(rawQuery string, tsdbQuery *tsdb.TsdbQuery, resp *Response) (*tsdb.Response, error) {
	result := &tsdb.Response{
		Results: make(map[string]*tsdb.QueryResult),
	}
	for _, query := range tsdbQuery.Queries {
		ret, err := translateResponse(resp, query)
		if err != nil {
			return nil, errors.Wrap(err, "translate response")
		}
		ret.Meta = monitor.QueryResultMeta{
			RawQuery: rawQuery,
		}
		result.Results[query.RefId] = ret
	}

	return result, nil
}

func translateResponse(resp *Response, query *tsdb.Query) (*tsdb.QueryResult, error) {
	queryRes := tsdb.NewQueryResult()

	isUnionResult := false
	results := resp.Data.Result
	if len(results) > 0 {
		_, isUnionResult = results[0].Metric[translator.UNION_RESULT_NAME]
	}

	// 添加值不同的 tag key
	diffTagKeys := sets.NewString()
	if len(results) > 1 {
		result0 := results[0]
		restResults := results[1:]
		for tagKey, tagVal := range result0.Metric {
			for _, result := range restResults {
				resultTagVal := result.Metric[tagKey]
				if tagVal != resultTagVal {
					diffTagKeys.Insert(tagKey)
					break
				}
			}
		}
	}

	if !isUnionResult {
		for _, result := range results {
			ss := transformSeries(result, query, diffTagKeys)
			queryRes.Series = append(queryRes.Series, ss...)
		}
	} else {
		// process union multiple fields response
		points, err := newPointsByResults(results)
		if err != nil {
			return nil, errors.Wrap(err, "process multi fields")
		}
		ss := transPointsToSeries(points, query)
		queryRes.Series = ss
	}

	return queryRes, nil
}

// Check VictoriaMetrics response at: https://docs.victoriametrics.com/keyConcepts.html#range-query
func transformSeries(vmResult ResponseDataResult, query *tsdb.Query, diffTagKeys sets.String) monitor.TimeSeriesSlice {
	var result monitor.TimeSeriesSlice
	metric := vmResult.Metric

	points := transValuesToTSDBPoints(vmResult.Values)

	tags := reviseTags(metric)

	aliasName := ""
	if len(query.Selects) > 0 {
		lastSel := query.Selects[len(query.Selects)-1]
		lastSelPart := lastSel[len(lastSel)-1]
		if lastSelPart.Type == "alias" && len(lastSelPart.Params) > 0 {
			aliasName = lastSelPart.Params[0]
		}
	}
	metricName := metric[labels.MetricName]
	if metricName == "" {
		metricName = "value"
	}
	if aliasName != "" {
		metricName = aliasName
	}
	ts := tsdb.NewTimeSeries(metricName, formatRawName(0, metricName, query, tags, diffTagKeys), []string{metricName, "time"}, points, tags)
	result = append(result, ts)
	return result
}

func formatRawName(idx int, name string, query *tsdb.Query, tags map[string]string, diffTagKeys sets.String) string {
	groupByTags := []string{}
	if query != nil {
		for _, group := range query.GroupBy {
			if group.Type == "tag" {
				groupByTags = append(groupByTags, group.Params[0])
			}
		}
	}
	return tsdb.FormatRawName(idx, name, groupByTags, tags, diffTagKeys)
}

func parseTimepoint(val ResponseDataResultValue) (monitor.TimePoint, error) {
	timepoint := make(monitor.TimePoint, 0)
	// parse timestamp
	timestampNumber, _ := val[0].(json.Number)
	timestamp, err := timestampNumber.Float64()
	if err != nil {
		return monitor.TimePoint{}, errors.Wrapf(err, "parse timestampNumber")
	}
	// to influxdb timestamp format, millisecond ?
	timestamp *= 1000

	// parse value
	for i := 1; i < len(val); i++ {
		valStr := val[i]
		pVal := parsePointValue(valStr)
		timepoint = append(timepoint, pVal)
	}
	timepoint = append(timepoint, timestamp)
	return timepoint, nil
}

func parsePointValue(value interface{}) interface{} {
	number, ok := value.(json.Number)
	if !ok {
		// try parse string
		valStr, ok := value.(string)
		if ok {
			valF, err := strconv.ParseFloat(valStr, 64)
			if err == nil {
				return &valF
			}
			return value
		}
		return value
	}

	fvalue, err := number.Float64()
	if err == nil {
		return &fvalue
	}

	ivalue, err := number.Int64()
	if err == nil {
		ret := float64(ivalue)
		return &ret
	}
	return number.String()
}

func (vm *vmAdapter) FilterMeasurement(ctx context.Context, ds *tsdb.DataSource, from, to string, ms *monitor.InfluxMeasurement, tagFilter *monitor.MetricQueryTag) (*monitor.InfluxMeasurement, error) {
	retMs := new(monitor.InfluxMeasurement)
	q := mod.NewAlertQuery(ms.Database, ms.Measurement).From(from).To(to)
	q.Interval("5m")
	q.Selects().Select("*").LAST()
	if tagFilter != nil {
		q.Where().AddTag(tagFilter)
	}
	q.GroupBy().TAG(labels.MetricName)
	tq := q.ToTsdbQuery()
	resp, err := vm.Query(ctx, ds, tq)
	if err != nil {
		return nil, errors.Wrap(err, "VictoriaMetrics.Query")
	}
	ss := resp.Results[""].Series
	//log.Infof("=====get ss: %s", jsonutils.Marshal(ss).PrettyString())

	// parse fields
	retFields := sets.NewString()
	msPrefix := fmt.Sprintf("%s_", ms.Measurement)
	for _, s := range ss {
		cols := s.Columns
		for _, col := range cols {
			if !strings.HasPrefix(col, msPrefix) {
				continue
			}
			field := strings.TrimPrefix(col, msPrefix)
			retFields.Insert(field)
		}
	}
	retMs.FieldKey = retFields.List()
	if len(retMs.FieldKey) != 0 {
		retMs.Measurement = ms.Measurement
		retMs.Database = ms.Database
		retMs.ResType = ms.ResType
	}
	return retMs, nil
}

func (vm *vmAdapter) FillSelect(query *monitor.AlertQuery, isAlert bool) *monitor.AlertQuery {
	if isAlert {
		query = influxdb.FillSelectWithMean(query)
	}
	return query
}

func (vm *vmAdapter) FillGroupBy(query *monitor.AlertQuery, inputQuery *monitor.MetricQueryInput, tagId string, isAlert bool) *monitor.AlertQuery {
	if isAlert {
		query = influxdb.FillGroupByWithWildChar(query, inputQuery, tagId)
	}
	return query
}
