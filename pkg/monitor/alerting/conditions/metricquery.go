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
	mq "yunion.io/x/onecloud/pkg/monitor/metricquery"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

func init() {
	mq.RegisterMetricQuery("metricquery", func(model []*monitor.AlertCondition) (mq.MetricQuery, error) {
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
			return nil, errors.Wrapf(err, "from value %q", qc.Query.From)
		}

		if err := validators.ValidateToValue(qc.Query.To); err != nil {
			return nil, errors.Wrapf(err, "to value %q", qc.Query.To)
		}
		qc.setResType()
		qc.Query.DataSourceId = q.DataSourceId
		cond.QueryCons = append(cond.QueryCons, *qc)
	}

	return cond, nil
}

func (query *MetricQueryCondition) ExecuteQuery(userCred mcclient.TokenCredential, skipCheckSeries bool) (*mq.Metrics, error) {
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

	queryInfluxdb := func() (*queryResult, error) {
		startTime := time.Now()
		queryResult, err := query.executeQuery(&evalContext, timeRange)
		if err != nil {
			return nil, err
		}
		log.Debugf("query metrics from influxdb elapsed: %s", time.Since(startTime))
		return queryResult, nil
	}

	if query.noCheckSeries(skipCheckSeries) {
		qr, err = queryInfluxdb()
		if err != nil {
			return nil, err
		}
		metrics := mq.Metrics{
			Series: make(tsdb.TimeSeriesSlice, 0),
			Metas:  qr.metas,
		}
		metrics.Series = qr.series
		return &metrics, nil
	}

	qInfluxdbCh := make(chan bool, 0)
	qRegionCh := make(chan bool, 0)

	go func() {
		qr, err = queryInfluxdb()
		qInfluxdbCh <- true
	}()

	var (
		regionErr error
		ress      []jsonutils.JSONObject
		resMap    = make(map[string]jsonutils.JSONObject)
	)
	go func() {
		startTime := time.Now()

		s := auth.GetSession(ctx, userCred, "")
		ress, regionErr = firstCond.getOnecloudResources(s, false)
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
		log.Debugf("get resources from region elapsed: %s", time.Since(startTime))
		qRegionCh <- true
	}()

	<-qInfluxdbCh
	<-qRegionCh

	startTime := time.Now()
	//for _, serie := range queryResult.series {
	//	isLatestOfSerie, resource := firstCond.serieIsLatestResource(resMap, serie)
	//	if !isLatestOfSerie {
	//		continue
	//	}
	//	firstCond.FillSerieByResourceField(resource, serie)
	//	metrics.Series = append(metrics.Series, serie)
	//}
	metrics := mq.Metrics{
		Series: make(tsdb.TimeSeriesSlice, 0),
		Metas:  qr.metas,
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
	if containGlob {
		return false
	}
	return true
}

func (c *MetricQueryCondition) executeQuery(context *alerting.EvalContext, timeRange *tsdb.TimeRange) (*queryResult, error) {
	ds, err := models.DataSourceManager.GetSource(c.QueryCons[0].Query.DataSourceId)
	if err != nil {
		return nil, errors.Wrapf(err, "Cound not find datasource %v", c.QueryCons[0].Query.DataSourceId)
	}

	req := c.getRequestQuery(ds, timeRange, context.IsDebug)
	result := make(tsdb.TimeSeriesSlice, 0)
	metas := make([]tsdb.QueryResultMeta, 0)

	if context.IsDebug {
		setContextLog(context, req)
	}

	resp, err := c.HandleRequest(context.Ctx, ds.ToTSDBDataSource(""), req)
	if err != nil {
		if err == gocontext.DeadlineExceeded {
			return nil, errors.Error("Alert execution exceeded the timeout")
		}
		log.Errorf("metricQuery HandleRequest error:%v", err)
		return nil, err
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

func (query *MetricQueryCondition) getRequestQuery(ds *models.SDataSource, timeRange *tsdb.TimeRange, debug bool) *tsdb.TsdbQuery {
	querys := make([]*tsdb.Query, 0)
	for _, qc := range query.QueryCons {
		querys = append(querys, &tsdb.Query{
			RefId:       strconv.FormatInt(int64(qc.Index), 10),
			MetricQuery: qc.Query.Model,
			DataSource:  *ds.ToTSDBDataSource(qc.Query.Model.Database),
		})
	}
	req := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Queries:   querys,
		Debug:     debug,
	}
	return req
}
