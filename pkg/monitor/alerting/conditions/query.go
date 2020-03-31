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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

func init() {
	alerting.RegisterCondition("query", func(model *monitor.AlertCondition, index int) (alerting.Condition, error) {
		return newQueryCondition(model, index)
	})
}

// QueryCondition is responsible for issue and query. reduce the
// timeseries into single values and evaluate if they are firing or not.
type QueryCondition struct {
	Index         int
	Query         AlertQuery
	Reducer       *queryReducer
	Evaluator     AlertEvaluator
	Operator      string
	HandleRequest tsdb.HandleRequestFunc
}

// AlertQuery contains information about what datasource a query
// should be send to and the query object.
type AlertQuery struct {
	Model        monitor.MetricQuery
	DataSourceId string
	From         string
	To           string
}

type FormatCond struct {
	QueryMeta *tsdb.QueryResultMeta
	Reducer   string
	Evaluator AlertEvaluator
}

func (c *QueryCondition) GenerateFormatCond(meta *tsdb.QueryResultMeta) *FormatCond {
	return &FormatCond{
		QueryMeta: meta,
		Reducer:   c.Reducer.Type,
		Evaluator: c.Evaluator,
	}
}
func (c FormatCond) String() string {
	if c.QueryMeta != nil {
		return fmt.Sprintf("%s(%q) %s", c.Reducer, c.QueryMeta.RawQuery, c.Evaluator.String())
	}
	return "no_data"
}

func (c *QueryCondition) filterTags(tags map[string]string) map[string]string {
	ret := make(map[string]string)
	for key, val := range tags {
		if strings.HasSuffix(key, "_id") {
			continue
		}
		ret[key] = val
	}
	return ret
}

// Eval evaluates te `QueryCondition`.
func (c *QueryCondition) Eval(context *alerting.EvalContext) (*alerting.ConditionResult, error) {
	timeRange := tsdb.NewTimeRange(c.Query.From, c.Query.To)

	ret, err := c.executeQuery(context, timeRange)
	if err != nil {
		return nil, err
	}
	seriesList := ret.series
	metas := ret.metas

	emptySeriesCount := 0
	evalMatchCount := 0
	var matches []*monitor.EvalMatch

	for idx, series := range seriesList {
		reducedValue := c.Reducer.Reduce(series)
		evalMatch := c.Evaluator.Eval(reducedValue)

		if reducedValue == nil {
			emptySeriesCount++
		}

		if context.IsTestRun {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: fmt.Sprintf("Condition[%d]: Eval: %v, Metric: %s, Value: %v", c.Index, evalMatch, series.Name, reducedValue),
			})
		}

		if evalMatch {
			evalMatchCount++
		}
		tags := c.filterTags(series.Tags)
		matches = append(matches, &monitor.EvalMatch{
			Condition: c.GenerateFormatCond(&metas[idx]).String(),
			Metric:    series.Name,
			Value:     reducedValue,
			Tags:      tags,
		})
	}

	// handle no series special case
	if len(seriesList) == 0 {
		// eval condition for null value
		evalMatch := c.Evaluator.Eval(nil)

		if context.IsTestRun {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: fmt.Sprintf("Condition: Eval: %v, Query returned No Series (reduced to null/no value)", evalMatch),
			})
		}

		if evalMatch {
			evalMatchCount++
			matches = append(matches, &monitor.EvalMatch{
				Metric: "NoData",
				Value:  nil,
			})
		}
	}

	return &alerting.ConditionResult{
		Firing:      evalMatchCount > 0,
		NoDataFound: emptySeriesCount == len(seriesList),
		Operator:    c.Operator,
		EvalMatches: matches,
	}, nil
}

type queryResult struct {
	series tsdb.TimeSeriesSlice
	metas  []tsdb.QueryResultMeta
}

func (c *QueryCondition) executeQuery(context *alerting.EvalContext, timeRange *tsdb.TimeRange) (*queryResult, error) {
	ds, err := models.DataSourceManager.GetSource(c.Query.DataSourceId)
	if err != nil {
		return nil, errors.Wrapf(err, "Cound not find datasource %v", c.Query.DataSourceId)
	}

	req := c.getRequestForAlertRule(ds, timeRange, context.IsDebug)
	result := make(tsdb.TimeSeriesSlice, 0)
	metas := make([]tsdb.QueryResultMeta, 0)

	if context.IsDebug {
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
			Message: fmt.Sprintf("Condition[%d]: Query", c.Index),
			Data:    data,
		})
	}

	resp, err := c.HandleRequest(context.Ctx, ds.ToTSDBDataSource(""), req)
	if err != nil {
		if err == gocontext.DeadlineExceeded {
			return nil, errors.Error("Alert execution exceeded the timeout")
		}

		return nil, errors.Wrap(err, "tsdb.HandleRequest() error")
	}

	for _, v := range resp.Results {
		if v.Error != nil {
			return nil, errors.Wrap(err, "tsdb.HandleResult() response")
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
				Message: fmt.Sprintf("Condition[%d]: Query Result", c.Index),
				Data:    queryResultData,
			})
		}
	}

	return &queryResult{
		series: result,
		metas:  metas,
	}, nil
}

func (c *QueryCondition) getRequestForAlertRule(ds *models.SDataSource, timeRange *tsdb.TimeRange, debug bool) *tsdb.TsdbQuery {
	req := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Queries: []*tsdb.Query{
			{
				RefId:       "A",
				MetricQuery: c.Query.Model,
				DataSource:  *ds.ToTSDBDataSource(c.Query.Model.Database),
			},
		},
		Debug: debug,
	}
	return req
}

func newQueryCondition(model *monitor.AlertCondition, index int) (*QueryCondition, error) {
	cond := new(QueryCondition)
	cond.Index = index
	cond.HandleRequest = tsdb.HandleRequest

	q := model.Query
	cond.Query.Model = q.Model
	cond.Query.From = q.From
	cond.Query.To = q.To

	if err := validators.ValidateFromValue(cond.Query.From); err != nil {
		return nil, errors.Wrapf(err, "from value %q", cond.Query.From)
	}

	if err := validators.ValidateToValue(cond.Query.To); err != nil {
		return nil, errors.Wrapf(err, "to value %q", cond.Query.To)
	}

	cond.Query.DataSourceId = q.DataSourceId
	reducer := model.Reducer
	cond.Reducer = newSimpleReducer(reducer.Type)

	evaluator, err := NewAlertEvaluator(&model.Evaluator)
	if err != nil {
		return nil, fmt.Errorf("error in condition %v: %v", index, err)
	}
	cond.Evaluator = evaluator
	operator := model.Operator
	if operator == "" {
		operator = "and"
	}
	cond.Operator = operator

	return cond, nil
}
