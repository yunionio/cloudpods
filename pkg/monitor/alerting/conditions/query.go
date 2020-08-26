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
	"yunion.io/x/pkg/utils"

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
	Reducer       Reducer
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
	QueryMeta    *tsdb.QueryResultMeta
	QueryKeyInfo string
	Reducer      string
	Evaluator    AlertEvaluator
}

func (c *QueryCondition) GenerateFormatCond(meta *tsdb.QueryResultMeta, metric string) *FormatCond {
	return &FormatCond{
		QueryMeta:    meta,
		QueryKeyInfo: metric,
		Reducer:      c.Reducer.GetType(),
		Evaluator:    c.Evaluator,
	}
}
func (c FormatCond) String() string {
	if c.QueryMeta != nil {
		return fmt.Sprintf("%s(%q) %s", c.Reducer, c.QueryMeta.RawQuery, c.Evaluator.String())
	}
	return "no_data"
}

func (c *QueryCondition) filterTags(tags map[string]string, details monitor.CommonAlertMetricDetails) map[string]string {
	ret := make(map[string]string)
	for key, val := range tags {
		if strings.HasSuffix(key, "_id") {
			continue
		}
		if len(val) == 0 {
			continue
		}
		if tag, ok := monitor.MEASUREMENT_TAG_KEYWORD[details.ResType]; ok {
			if key == tag {
				ret["name"] = val
			}
		}
		if strings.Contains(key, "ip") {
			ret["ip"] = val
		}
		ret[key] = val
	}
	for _, tag := range []string{"brand", "platform", "hypervisor"} {
		if val, ok := ret[tag]; ok {
			ret["brand"] = val
			break
		}
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

	for _, series := range seriesList {
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
		var meta *tsdb.QueryResultMeta
		if len(metas) > 0 {
			//the relation metas with series is 1 to more
			meta = &metas[0]
		}
		if evalMatch {
			evalMatch, err := c.NewEvalMatch(context, *series, meta, reducedValue)
			if err != nil {
				return nil, errors.Wrap(err, "NewEvalMatch error")
			}
			matches = append(matches, evalMatch)
		}
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

func (c *QueryCondition) NewEvalMatch(context *alerting.EvalContext, series tsdb.TimeSeries,
	meta *tsdb.QueryResultMeta, value *float64) (*monitor.EvalMatch,
	error) {
	evalMatch := new(monitor.EvalMatch)
	alert, err := models.CommonAlertManager.GetAlert(context.Rule.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlert to NewEvalMatch error")
	}
	settings, _ := alert.GetSettings()
	alertDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(c.Index, settings.Conditions[c.Index])
	evalMatch.Metric = fmt.Sprintf("%s.%s", alertDetails.Measurement, alertDetails.Field)
	queryKeyInfo := ""
	if len(alertDetails.MeasurementDisplayName) > 0 && len(alertDetails.FieldDescription.DisplayName) > 0 {
		queryKeyInfo = fmt.Sprintf("%s.%s", alertDetails.MeasurementDisplayName, alertDetails.FieldDescription.DisplayName)
	}
	if len(queryKeyInfo) == 0 {
		queryKeyInfo = evalMatch.Metric
	}
	msg := fmt.Sprintf("%s.%s %s %.4f ", alertDetails.Measurement, alertDetails.Field,
		alertDetails.Comparator, alertDetails.Threshold)
	if len(context.Rule.Message) == 0 {
		context.Rule.Message = msg
	}
	evalMatch.Condition = c.GenerateFormatCond(meta, queryKeyInfo).String()
	evalMatch.Tags = c.filterTags(series.Tags, *alertDetails)
	evalMatch.Unit = alertDetails.FieldDescription.Unit
	evalMatch.Value = value
	evalMatch.ValueStr = c.RationalizeValueFromUnit(*value, alertDetails.FieldDescription.Unit)
	return evalMatch, nil
}

var fileSize = []string{"bps", "Bps", "byte"}

func (c *QueryCondition) RationalizeValueFromUnit(value float64, unit string) string {
	if utils.IsInStringArray(unit, fileSize) {
		if unit == "byte" {
			return (formatFileSize(value, unit, float64(1024)))
		}
		return formatFileSize(value, unit, float64(1000))
	}
	return fmt.Sprintf("%0.4f %s", value, unit)
}

// 单位转换 保留四位小数
func formatFileSize(fileSize float64, unit string, unitsize float64) (size string) {
	if fileSize < unitsize {
		return fmt.Sprintf("%.4f%s", fileSize, unit)
	} else if fileSize < (unitsize * unitsize) {
		return fmt.Sprintf("%.4fK%s", float64(fileSize)/float64(unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.4fM%s", float64(fileSize)/float64(unitsize*unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.4fG%s", float64(fileSize)/float64(unitsize*unitsize*unitsize), unit)
	} else if fileSize < (unitsize * unitsize * unitsize * unitsize * unitsize) {
		return fmt.Sprintf("%.4fT%s", float64(fileSize)/float64(unitsize*unitsize*unitsize*unitsize), unit)
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.4fE%s", float64(fileSize)/float64(unitsize*unitsize*unitsize*unitsize*unitsize), unit)
	}
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
	//reducer := model.Reducer
	//cond.Reducer = newSimpleReducer(reducer.Type)
	reducer, err := NewAlertReducer(&model.Reducer)
	if err != nil {
		return nil, fmt.Errorf("error in condition %v: %v", index, err)
	}
	cond.Reducer = reducer
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
