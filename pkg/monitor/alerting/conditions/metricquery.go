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

package conditions

import (
	gocontext "context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/datasource"
	mq "yunion.io/x/onecloud/pkg/monitor/metricquery"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

func init() {
	mq.RegisterMetricQuery(monitor.ConditionTypeMetricQuery, func(model []*monitor.AlertCondition) (mq.MetricQuery, error) {
		return NewMetricQueryCondition(model)
	})
}

type MetricQueryCondition struct {
	QueryCons     []QueryCondition
	HandleRequest tsdb.HandleRequestFunc
}

func NewMetricQueryCondition(models []*monitor.AlertCondition) (*MetricQueryCondition, error) {
	cond := new(MetricQueryCondition)

	cond.HandleRequest = tsdb.HandleRequest
	for index, model := range models {
		qc := new(QueryCondition)
		qc.Index = index
		q := model.Query
		qc.Query.Model = q.Model
		qc.Query.From = q.From
		qc.Query.To = q.To
		if err := validators.ValidateFromValue(qc.Query.From); err != nil {
			return nil, errors.Wrapf(err, "validate from value %q", qc.Query.From)
		}

		if err := validators.ValidateToValue(qc.Query.To); err != nil {
			return nil, errors.Wrapf(err, "validate to value %q", qc.Query.To)
		}
		reducer, err := NewAlertReducer(&model.Reducer)
		if err != nil {
			return nil, errors.Wrapf(err, "NewAlertReducer")
		}
		qc.Reducer = reducer
		qc.ReducerOrder = model.ReducerOrder
		qc.setResType()
		cond.QueryCons = append(cond.QueryCons, *qc)
	}

	return cond, nil
}

type queryReducedResultSorter struct {
	r     *queryResult
	order monitor.ResultReducerOrder
}

func newQueryReducedResultSorter(
	r *queryResult,
	order monitor.ResultReducerOrder) *queryReducedResultSorter {
	return &queryReducedResultSorter{
		r:     r,
		order: order,
	}
}

func (q *queryReducedResultSorter) Len() int {
	return len(q.r.reducedResult.Result)
}

func (q *queryReducedResultSorter) Less(i, j int) bool {
	vi, vj := q.r.reducedResult.Result[i], q.r.reducedResult.Result[j]
	if q.order == monitor.RESULT_REDUCER_ORDER_ASC {
		if vi < vj {
			return true
		}
		return false
	}
	if vi > vj {
		return true
	}
	return false
}

func (q *queryReducedResultSorter) Swap(i, j int) {
	q.r.reducedResult.Result[i], q.r.reducedResult.Result[j] = q.r.reducedResult.Result[j], q.r.reducedResult.Result[i]
	q.r.series[i], q.r.series[j] = q.r.series[j], q.r.series[i]
	if len(q.r.metas) > i && len(q.r.metas) > j {
		q.r.metas[i], q.r.metas[j] = q.r.metas[j], q.r.metas[i]
	}
}

func (query *MetricQueryCondition) ExecuteQuery(userCred mcclient.TokenCredential, scope string, skipCheckSeries bool) (*monitor.MetricsQueryResult, error) {
	firstCond := query.QueryCons[0]
	timeRange := tsdb.NewTimeRange(firstCond.Query.From, firstCond.Query.To)
	ctx := gocontext.Background()
	evalContext := alerting.EvalContext{
		Ctx:       ctx,
		IsDebug:   true,
		IsTestRun: false,
	}

	allStartTime := time.Now()
	var qr *queryResult
	var err error

	ds, err := datasource.GetDefaultSource("")
	if err != nil {
		return nil, errors.Wrapf(err, "Can't find default datasource")
	}

	sortMetrics := func(metrics *queryResult, order monitor.ResultReducerOrder) *queryResult {
		if metrics.reducedResult != nil && len(metrics.reducedResult.Result) > 0 {
			s := newQueryReducedResultSorter(metrics, order)
			sort.Sort(s)
			return s.r
		}
		return metrics
	}

	queryTSDB := func() (*queryResult, error) {
		startTime := time.Now()

		queryResult, err := query.executeQuery(ds, &evalContext, timeRange)
		if err != nil {
			return nil, errors.Wrapf(err, "query.executeQuery from %s", ds.Type)
		}
		log.Debugf("query metrics from TSDB %q elapsed: %s", ds.Type, time.Since(startTime))
		if firstCond.Reducer.GetType() != "" {
			if queryResult.reducedResult == nil {
				queryResult.reducedResult = &monitor.ReducedResult{
					Reducer: monitor.Condition{
						Type:   string(firstCond.Reducer.GetType()),
						Params: firstCond.Reducer.GetParams(),
					},
					Result: make([]float64, len(queryResult.series)),
				}
			}
			for i, ss := range queryResult.series {
				resultReducerValue, _ := firstCond.Reducer.Reduce(ss)
				if resultReducerValue != nil {
					queryResult.reducedResult.Result[i] = *resultReducerValue
				}
			}
		}
		return sortMetrics(queryResult, firstCond.ReducerOrder), nil
	}

	noCheck := query.noCheckSeries(skipCheckSeries)
	if noCheck {
		qr, err = queryTSDB()
		if err != nil {
			return nil, errors.Wrap(err, "queryTSDB")
		}
		metrics := &monitor.MetricsQueryResult{
			Series:        qr.series,
			Metas:         qr.metas,
			ReducedResult: qr.reducedResult,
		}
		return metrics, nil
	}

	qInfluxdbCh := make(chan bool, 0)
	qRegionCh := make(chan bool, 0)

	go func() {
		qr, err = queryTSDB()
		qInfluxdbCh <- true
	}()

	var (
		regionErr error
		ress      []jsonutils.JSONObject
		resMap    = make(map[string]jsonutils.JSONObject)
	)
	go func() {
		startTime := time.Now()
		defer func() {
			qRegionCh <- true
			log.Debugf("get resources from region elapsed: %s", time.Since(startTime))
		}()

		s := auth.GetSession(ctx, userCred, "")
		ress, regionErr = firstCond.getOnecloudResources(s, scope, false)
		if err != nil {
			regionErr = errors.Wrap(regionErr, "get resources from region")
			return
		}
		// convert resources to map
		for _, res := range ress {
			id, _ := res.GetString("id")
			if id == "" {
				continue
			}
			tmpRes := res
			resMap[id] = tmpRes
		}
	}()

	<-qInfluxdbCh
	<-qRegionCh

	if err != nil || regionErr != nil {
		return nil, errors.Errorf("tsdb error: %v, region error: %v", err, regionErr)
	}

	startTime := time.Now()
	//for _, serie := range queryResult.series {
	//	isLatestOfSerie, resource := firstCond.serieIsLatestResource(resMap, serie)
	//	if !isLatestOfSerie {
	//		continue
	//	}
	//	firstCond.FillSerieByResourceField(resource, serie)
	//	metrics.Series = append(metrics.Series, serie)
	//}
	metrics := monitor.MetricsQueryResult{
		Series:        make(monitor.TimeSeriesSlice, 0),
		Metas:         qr.metas,
		ReducedResult: qr.reducedResult,
	}
	mtx := sync.Mutex{}
	workqueue.Parallelize(4, len(qr.series), func(piece int) {
		serie := qr.series[piece]
		isLatestOfSerie, resource := firstCond.serieIsLatestResource(resMap, serie)
		if !isLatestOfSerie {
			return
		}
		firstCond.FillSerieByResourceField(resource, serie)
		mtx.Lock()
		defer mtx.Unlock()
		metrics.Series = append(metrics.Series, serie)
	})
	log.Debugf("fill metrics tag elapsed: %s", time.Since(startTime))
	log.Debugf("all steps elapsed: %s", time.Since(allStartTime))

	return &metrics, nil
}

func (query *MetricQueryCondition) noCheckSeries(skipCheckSeries bool) bool {
	firstCond := query.QueryCons[0]
	// always check series when resource type is "" or external resource
	if len(firstCond.ResType) == 0 || strings.HasPrefix(firstCond.ResType, monitor.EXT_PREFIX) {
		return true
	}
	if len(firstCond.Query.Model.GroupBy) == 0 {
		return true
	}

	if skipCheckSeries {
		return true
	}

	groupBys := make([]string, 0)
	containGlob := false
	for _, groupby := range firstCond.Query.Model.GroupBy {
		if utils.IsInStringArray("*", groupby.Params) {
			containGlob = true
		}
		groupBys = append(groupBys, groupby.Params...)
	}
	for _, supportId := range monitor.MEASUREMENT_TAG_ID {
		if utils.IsInStringArray(supportId, groupBys) {
			return false
		}
	}

	containNoCheckAggregator := false
	sels := firstCond.Query.Model.Selects
	lastSel := sels[len(sels)-1]
	for _, part := range lastSel {
		if utils.IsInStringArray(part.Type, []string{"sum"}) {
			containNoCheckAggregator = true
		}
	}

	if containGlob {
		if containNoCheckAggregator {
			return true
		}
		return false
	}
	return true
}

func (c *MetricQueryCondition) executeQuery(ds *tsdb.DataSource, context *alerting.EvalContext, timeRange *tsdb.TimeRange) (*queryResult, error) {
	req := c.getRequestQuery(ds, timeRange, context.IsDebug)
	result := make(monitor.TimeSeriesSlice, 0)
	metas := make([]monitor.QueryResultMeta, 0)

	if context.IsDebug {
		setContextLog(context, req)
	}

	resp, err := c.HandleRequest(context.Ctx, ds, req)
	if err != nil {
		if err == gocontext.DeadlineExceeded {
			return nil, errors.Error("Alert execution exceeded the timeout")
		}
		return nil, errors.Wrap(err, "metricQuery HandleRequest")
	}
	for _, v := range resp.Results {
		if v.Error != nil {
			return nil, errors.Wrap(err, "metricQuery HandleResult response error")
		}

		result = append(result, v.Series...)
		metas = append(metas, v.Meta)

		queryResultData := map[string]interface{}{}

		if context.IsTestRun {
			queryResultData["series"] = v.Series
		}

		if context.IsDebug {
			queryResultData["meta"] = v.Meta
		}

		if context.IsTestRun || context.IsDebug {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: "Metric Query Result",
				Data:    queryResultData,
			})
		}
	}

	return &queryResult{
		series: result,
		metas:  metas,
	}, nil
}

func setContextLog(context *alerting.EvalContext, req *tsdb.TsdbQuery) {
	data := jsonutils.NewDict()
	if req.TimeRange != nil {
		data.Set("from", jsonutils.NewInt(req.TimeRange.GetFromAsMsEpoch()))
		data.Set("to", jsonutils.NewInt(req.TimeRange.GetToAsMsEpoch()))
	}

	type queryDto struct {
		RefId         string              `json:"refId"`
		Model         monitor.MetricQuery `json:"model"`
		Datasource    tsdb.DataSource     `json:"datasource"`
		MaxDataPoints int64               `json:"maxDataPoints"`
		IntervalMs    int64               `json:"intervalMs"`
	}

	queries := []*queryDto{}
	for _, q := range req.Queries {
		queries = append(queries, &queryDto{
			RefId:         q.RefId,
			Model:         q.MetricQuery,
			Datasource:    q.DataSource,
			MaxDataPoints: q.MaxDataPoints,
			IntervalMs:    q.IntervalMs,
		})
	}

	data.Set("queries", jsonutils.Marshal(queries))

	context.Logs = append(context.Logs, &monitor.ResultLogEntry{
		Message: "Metric Query",
		Data:    data,
	})
}

func (query *MetricQueryCondition) getRequestQuery(ds *tsdb.DataSource, timeRange *tsdb.TimeRange, debug bool) *tsdb.TsdbQuery {
	querys := make([]*tsdb.Query, 0)
	for _, qc := range query.QueryCons {
		nDs := *ds
		nDs.Database = qc.Query.Model.Database
		querys = append(querys, &tsdb.Query{
			RefId:       strconv.FormatInt(int64(qc.Index), 10),
			MetricQuery: qc.Query.Model,
			DataSource:  nDs,
		})
	}
	req := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Queries:   querys,
		Debug:     debug,
	}
	return req
}
