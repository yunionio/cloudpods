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
	"context"
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mc_mds "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/datasource"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

func init() {
	alerting.RegisterCondition("query", func(model *monitor.AlertCondition, index int) (alerting.Condition, error) {
		return newQueryCondition(model, index)
	})
	RestryFetchImp(monitor.METRIC_DATABASE_METER, &meterFetchImp{})
}

// QueryCondition is responsible for issue and query. reduce the
// timeseries into single values and evaluate if they are firing or not.
type QueryCondition struct {
	Index         int
	Query         AlertQuery
	Reducer       Reducer
	ReducerOrder  monitor.ResultReducerOrder
	Evaluator     AlertEvaluator
	Operator      string
	HandleRequest tsdb.HandleRequestFunc
	ResType       string
}

// AlertQuery contains information about what datasource a query
// should be send to and the query object.
type AlertQuery struct {
	Model monitor.MetricQuery
	From  string
	To    string
}

type FormatCond struct {
	QueryMeta    *monitor.QueryResultMeta
	QueryKeyInfo string
	Reducer      string
	Evaluator    AlertEvaluator
}

type iEvalMatchFetch interface {
	FetchCustomizeEvalMatch(context *alerting.EvalContext, evalMatch *monitor.EvalMatch,
		alertDetails *monitor.CommonAlertMetricDetails) error
}

var iFetchImp map[string]iEvalMatchFetch

func RestryFetchImp(db string, imp iEvalMatchFetch) {
	if iFetchImp == nil {
		iFetchImp = make(map[string]iEvalMatchFetch)
	}
	iFetchImp[db] = imp
}

func GetFetchImpByDb(db string) iEvalMatchFetch {
	if iFetchImp == nil {
		return nil
	}
	return iFetchImp[db]
}

func (c *QueryCondition) GenerateFormatCond(meta *monitor.QueryResultMeta, metric string) *FormatCond {
	return &FormatCond{
		QueryMeta:    meta,
		QueryKeyInfo: metric,
		Reducer:      string(c.Reducer.GetType()),
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
		//if strings.HasSuffix(key, "_id") {
		//	continue
		//}
		if len(val) == 0 {
			continue
		}
		if tag, ok := monitor.MEASUREMENT_TAG_KEYWORD[details.ResType]; ok {
			if key == tag {
				ret["name"] = val
			}
		}
		if strings.Contains(key, "ip") && key != "host_ip" {
			ret["ip"] = val
		}
		ret[key] = val
	}
	if _, ok := ret["ip"]; !ok {
		ret["ip"] = tags["host_ip"]
	}
	if _, ok := ret["name"]; !ok {
		ret["name"] = tags["host"]
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
	var alertOkmatches []*monitor.EvalMatch

	alert, err := models.CommonAlertManager.GetAlert(context.Rule.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlert to NewEvalMatch error")
	}

	for _, series := range seriesList {
		if len(c.ResType) != 0 {
			isLatestOfSerie, resource := c.serieIsLatestResource(nil, series)
			if !isLatestOfSerie {
				continue
			}
			c.FillSerieByResourceField(resource, series)
		}
		reducedValue, valStrArr := c.Reducer.Reduce(series)
		evalMatch := c.Evaluator.Eval(reducedValue)

		if reducedValue == nil {
			emptySeriesCount++
		}

		if context.IsTestRun {
			context.Logs = append(context.Logs, &monitor.ResultLogEntry{
				Message: fmt.Sprintf("Condition[%d]: EvalMatch: %v, Metric: %s, Value: %s", c.Index, evalMatch, series.Name, jsonutils.Marshal(reducedValue)),
			})
		}

		if evalMatch {
			evalMatchCount++
		}
		var meta *monitor.QueryResultMeta
		if len(metas) > 0 {
			//the relation metas with series is 1 to more
			meta = &metas[0]
		}
		if evalMatch {
			match, err := c.NewEvalMatch(alert, context, *series, meta, reducedValue, valStrArr, evalMatch)
			if err != nil {
				return nil, errors.Wrap(err, "NewEvalMatch error")
			}
			matches = append(matches, match)
		}
		if reducedValue != nil && !evalMatch {
			match, err := c.NewEvalMatch(alert, context, *series, meta, reducedValue, valStrArr, evalMatch)
			if err != nil {
				return nil, errors.Wrap(err, "NewEvalMatch error")
			}
			resId := monitor.GetResourceIdFromTagWithDefault(match.Tags, c.ResType)
			if err := OkEvalMatchSetIsRecovery(alert, resId, match); err != nil {
				log.Warningf("[Query] set eval match %s to recovered: %v", jsonutils.Marshal(match), err)
			}
			alertOkmatches = append(alertOkmatches, match)
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
	if evalMatchCount == 0 && len(seriesList) != 0 {
		log.Debugf("sql-meta: %s", metas[0].RawQuery)
	}

	return &alerting.ConditionResult{
		Firing:             evalMatchCount > 0,
		NoDataFound:        emptySeriesCount == len(seriesList),
		Operator:           c.Operator,
		EvalMatches:        matches,
		AlertOkEvalMatches: alertOkmatches,
	}, nil
}

func (c *QueryCondition) serieIsLatestResource(resources map[string]jsonutils.JSONObject, series *monitor.TimeSeries) (bool, jsonutils.JSONObject) {
	resId := monitor.GetResourceIdFromTagWithDefault(series.Tags, c.ResType)
	if len(resources) != 0 {
		resource, ok := resources[resId]
		if !ok {
			return false, nil
		}
		got, obj := models.MonitorResourceManager.GetResourceObj(resId)
		if got {
			return true, obj
		} else {
			log.Warningf("not found resource %s %s from cache, use list item directly", c.ResType, resId)
			return true, resource
		}
	}
	return models.MonitorResourceManager.GetResourceObj(resId)
}

func (c *QueryCondition) FillSerieByResourceField(resource jsonutils.JSONObject,
	series *monitor.TimeSeries) {
	//startTime := time.Now()
	//defer func() {
	//	log.Debugf("--FillSerieByResourceField: %s", time.Since(startTime))
	//}()
	tagKeyRelationMap := c.getTagKeyRelationMap()
	for tagKey, resourceKey := range tagKeyRelationMap {
		val, err := resource.Get(resourceKey)
		if err != nil {
			continue
		}
		series.Tags[tagKey], _ = val.GetString()
	}
}

func (c *QueryCondition) NewEvalMatch(
	alert *models.SCommonAlert,
	context *alerting.EvalContext,
	series monitor.TimeSeries,
	meta *monitor.QueryResultMeta,
	value *float64, valStrArr []string, isMatch bool) (*monitor.EvalMatch, error) {
	evalMatch := new(monitor.EvalMatch)
	alertDetails, err := c.GetCommonAlertDetails(alert)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlert to NewEvalMatch error")
	}
	evalMatch.Metric = fmt.Sprintf("%s.%s", alertDetails.Measurement, alertDetails.Field)
	queryKeyInfo := ""
	if len(alertDetails.MeasurementDisplayName) > 0 && len(alertDetails.FieldDescription.DisplayName) > 0 {
		queryKeyInfo = fmt.Sprintf("%s.%s", alertDetails.MeasurementDisplayName, alertDetails.FieldDescription.DisplayName)
	}
	if len(queryKeyInfo) == 0 {
		queryKeyInfo = evalMatch.Metric
	}
	evalMatch.Unit = alertDetails.FieldDescription.Unit
	evalMatch.Tags = c.filterTags(series.Tags, *alertDetails)
	evalMatch.Value = value
	evalMatch.ValueStr = models.RationalizeValueFromUnit(*value, alertDetails.FieldDescription.Unit,
		alertDetails.FieldOpt)
	if alertDetails.GetPointStr {
		evalMatch.ValueStr = c.jointPointStr(series, evalMatch.ValueStr, valStrArr)
	}
	c.FetchCustomizeEvalMatch(context, evalMatch, alertDetails)
	//c.newRuleDescription(context, alertDetails)
	//evalMatch.Condition = c.GenerateFormatCond(meta, queryKeyInfo).String()
	msg := fmt.Sprintf("%s.%s %s %s", alertDetails.Measurement, alertDetails.Field,
		alertDetails.Comparator, models.RationalizeValueFromUnit(alertDetails.Threshold, evalMatch.Unit, ""))
	if len(context.Rule.Message) == 0 {
		context.Rule.Message = msg
	}
	// 避免重复指标
	if isMatch {
		op := alertDetails.Operator
		if op != "" && c.Index > 0 {
			msg = fmt.Sprintf("%s %s", strings.ToUpper(op), msg)
		}
		msgSet := sets.NewString(context.Rule.TriggeredMessages...)
		if !msgSet.Has(msg) {
			context.Rule.TriggeredMessages = append(context.Rule.TriggeredMessages, msg)
		}
	}
	evalMatch.AlertDetails = jsonutils.Marshal(alertDetails)
	return evalMatch, nil
}

func (c *QueryCondition) FetchCustomizeEvalMatch(context *alerting.EvalContext, evalMatch *monitor.EvalMatch,
	alertDetails *monitor.CommonAlertMetricDetails) {
	imp := GetFetchImpByDb(alertDetails.DB)
	if imp != nil {
		err := imp.FetchCustomizeEvalMatch(context, evalMatch, alertDetails)
		if err != nil {
			log.Errorf("%s FetchCustomizeEvalMatch err:%v", alertDetails.DB, err)
		}
	}
}

type meterFetchImp struct {
}

func (m *meterFetchImp) FetchCustomizeEvalMatch(context *alerting.EvalContext, evalMatch *monitor.EvalMatch,
	alertDetails *monitor.CommonAlertMetricDetails) error {
	meterCustomizeConfig := new(monitor.MeterCustomizeConfig)
	if context.Rule.CustomizeConfig == nil {
		return nil
	}
	err := context.Rule.CustomizeConfig.Unmarshal(meterCustomizeConfig)
	if err != nil {
		return err
	}
	//evalMatch.ValueStr = evalMatch.ValueStr + " " + meterCustomizeConfig.UnitDesc
	evalMatch.Unit = meterCustomizeConfig.UnitDesc
	return nil
}

func (c *QueryCondition) GetCommonAlertDetails(alert *models.SCommonAlert) (*monitor.CommonAlertMetricDetails, error) {
	settings, _ := alert.GetSettings()
	alertDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(c.Index, &settings.Conditions[c.Index])
	return alertDetails, nil
}

func (c *QueryCondition) jointPointStr(series monitor.TimeSeries, value string, valStrArr []string) string {
	str := ""
	for i := 0; i < len(valStrArr); i++ {
		if i == 0 {
			str = fmt.Sprintf("%s=%s", series.Columns[i], value)
			continue
		}
		str = fmt.Sprintf("%s,%s=%s", str, series.Columns[i], valStrArr[i])
	}
	return str
}

type queryResult struct {
	series        monitor.TimeSeriesSlice
	metas         []monitor.QueryResultMeta
	reducedResult *monitor.ReducedResult
}

func (c *QueryCondition) executeQuery(evalCtx *alerting.EvalContext, timeRange *tsdb.TimeRange) (*queryResult, error) {
	req := c.getRequestForAlertRule(timeRange, evalCtx.IsDebug)
	result := make(monitor.TimeSeriesSlice, 0)
	metas := make([]monitor.QueryResultMeta, 0)

	if evalCtx.IsDebug {
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

		evalCtx.Logs = append(evalCtx.Logs, &monitor.ResultLogEntry{
			Message: fmt.Sprintf("Condition[%d]: Query", c.Index),
			Data:    data,
		})
	}

	ds, err := datasource.GetDefaultSource(c.Query.Model.Database)
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultDataSource")
	}

	resp, err := c.HandleRequest(evalCtx.Ctx, ds, req)
	if err != nil {
		if err == context.DeadlineExceeded {
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

		if evalCtx.IsTestRun {
			queryResultData["series"] = v.Series
		}

		if evalCtx.IsDebug {
			queryResultData["meta"] = v.Meta
		}

		if evalCtx.IsTestRun || evalCtx.IsDebug {
			evalCtx.Logs = append(evalCtx.Logs, &monitor.ResultLogEntry{
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

func (c *QueryCondition) getRequestForAlertRule(timeRange *tsdb.TimeRange, debug bool) *tsdb.TsdbQuery {
	ds, _ := datasource.GetDefaultSource(c.Query.Model.Database)
	req := &tsdb.TsdbQuery{
		TimeRange: timeRange,
		Queries: []*tsdb.Query{
			{
				RefId:       "A",
				MetricQuery: c.Query.Model,
				DataSource:  *ds,
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

	//reducer := model.Reducer
	//cond.Reducer = newSimpleReducer(reducer.Type)
	reducer, err := NewAlertReducer(&model.Reducer)
	if err != nil {
		return nil, fmt.Errorf("error in condition %v: %v", index, err)
	}
	cond.Reducer = reducer
	cond.ReducerOrder = model.ReducerOrder
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

	cond.checkGroupByField()
	cond.setResType()

	return cond, nil
}

func (c *QueryCondition) checkGroupByField() {
	metricMeasurement, _ := models.MetricMeasurementManager.GetCache().Get(c.Query.Model.Measurement)
	if metricMeasurement == nil {
		return
	}
	for i, group := range c.Query.Model.GroupBy {
		if group.Params[0] == "*" {
			c.Query.Model.GroupBy[i].Params = []string{monitor.GetMeasurementTagIdKeyByResType(metricMeasurement.ResType)}
		}
	}
}

func (c *QueryCondition) setResType() {
	metricMeasurement, _ := models.MetricMeasurementManager.GetCache().Get(c.Query.Model.Measurement)
	if metricMeasurement != nil {
		c.ResType = metricMeasurement.ResType
	}
	var resType = monitor.METRIC_RES_TYPE_HOST
	if len(c.Query.Model.GroupBy) == 0 {
		return
	}
	// NOTE: shouldn't set ResType when tenant_id and domain_id within GroupBy
	/* if len(resType) != 0 && c.Query.Model.GroupBy[0].Params[0] != monitor.
		MEASUREMENT_TAG_ID[resType] {
		for _, groupBy := range c.Query.Model.GroupBy {
			tag := groupBy.Params[0]
			if tag == "tenant_id" {
				resType = monitor.METRIC_RES_TYPE_TENANT
				break
			}
			if tag == "domain_id" {
				resType = monitor.METRIC_RES_TYPE_DOMAIN
				break
			}
		}
	} */
	if c.Query.Model.Database == monitor.METRIC_DATABASE_TELE {
		if c.ResType == "" {
			c.ResType = resType
		}
	}
}

func (c *QueryCondition) GetQueryResources(s *mcclient.ClientSession, scope string, showDetails bool) ([]jsonutils.JSONObject, error) {
	allRes, err := c.getOnecloudResources(s, scope, showDetails)
	if err != nil {
		return nil, errors.Wrap(err, "getOnecloudResources error")
	}
	allRes = c.filterAllResources(allRes)
	return allRes, nil
}

func (c *QueryCondition) getOnecloudResources(s *mcclient.ClientSession, scope string, showDetails bool) ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	queryStatus := []string{"running", "ready"}
	// query.Add(jsonutils.NewString("true"), "admin")
	//if len(c.Query.Model.Tags) != 0 {
	//	query, err = c.convertTagsQuery(evalContext, query)
	//	if err != nil {
	//		return nil, errors.Wrap(err, "NoDataQueryCondition convertTagsQuery error")
	//	}
	//}
	var (
		err          error
		manager      modulebase.Manager
		allResources = make([]jsonutils.JSONObject, 0)
	)
	switch c.ResType {
	case monitor.METRIC_RES_TYPE_HOST:
		models.SetQueryHostType(query)
		queryStatus = append(queryStatus, "unknown")
		query.Set("enabled", jsonutils.NewInt(1))
		manager = &mc_mds.Hosts
	case monitor.METRIC_RES_TYPE_GUEST:
		manager = &mc_mds.Servers
	case monitor.METRIC_RES_TYPE_AGENT:
		manager = &mc_mds.Servers
	case monitor.METRIC_RES_TYPE_RDS:
		manager = &mc_mds.DBInstance
	case monitor.METRIC_RES_TYPE_REDIS:
		manager = &mc_mds.ElasticCache
	case monitor.METRIC_RES_TYPE_OSS:
		manager = &mc_mds.Buckets
	case monitor.METRIC_RES_TYPE_CLOUDACCOUNT:
		query.Remove("status")
		query.Add(jsonutils.NewBool(true), "enabled")
		manager = &mc_mds.Cloudaccounts
	case monitor.METRIC_RES_TYPE_TENANT:
		manager = &identity.Projects
	case monitor.METRIC_RES_TYPE_DOMAIN:
		manager = &identity.Domains
	case monitor.METRIC_RES_TYPE_STORAGE:
		query.Remove("status")
		manager = &mc_mds.Storages
	default:
		query := jsonutils.NewDict()
		query.Set("brand", jsonutils.NewString(hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND))
		models.SetQueryHostType(query)
		manager = &mc_mds.Hosts
	}

	query.Add(jsonutils.NewStringArray(queryStatus), "status")
	allResources, err = ListAllResources(s, manager, query, scope, showDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "ListAllResources for %s with query %s, scope: %s, showDetails: %v", manager.GetKeyword(), query, scope, showDetails)
	}
	return allResources, nil
}

func ListAllResources(s *mcclient.ClientSession, manager modulebase.Manager, params *jsonutils.JSONDict, scope string, showDetails bool) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	if s.GetToken().HasSystemAdminPrivilege() {
		params.Add(jsonutils.NewString("system"), "scope")
	}
	if scope != "" {
		params.Add(jsonutils.NewString(scope), "scope")
	}
	params.Add(jsonutils.NewInt(int64(options.Options.APIListBatchSize)), "limit")
	params.Add(jsonutils.NewBool(showDetails), "details")
	var count int
	objs := make([]jsonutils.JSONObject, 0)
	for {
		params.Set("offset", jsonutils.NewInt(int64(count)))
		result, err := manager.List(s, params)
		if err != nil {
			return nil, errors.Wrapf(err, "list %s resources with params %s", manager.KeyString(), params.String())
		}
		objs = append(objs, result.Data...)
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	return objs, nil
}

func (c *QueryCondition) filterAllResources(resources []jsonutils.JSONObject) []jsonutils.JSONObject {
	if len(c.Query.Model.Tags) == 0 {
		return resources
	}
	filterIdMap := make(map[string]jsonutils.JSONObject)
	filterQuery := c.getFilterQuery()
	intKey := make([]int, 0)
	if len(filterQuery) != 0 {
		for key, _ := range filterQuery {
			intKey = append(intKey, key)
		}
		sort.Ints(intKey)
		minKey := intKey[0]
		if minKey != 0 {
			filterQuery[0] = minKey - 1
		}
	} else {
		filterQuery[0] = len(c.Query.Model.Tags) - 1
	}
	for start, end := range filterQuery {
		filterResources := c.getFilterResources(start, end, resources)
		filterIdMap = c.fillFilterRes(filterResources, filterIdMap)
	}
	filterRes := make([]jsonutils.JSONObject, 0)
	for _, obj := range filterIdMap {
		filterRes = append(filterRes, obj)
	}
	return filterRes
}

func (c *QueryCondition) getFilterQuery() map[int]int {
	length := len(c.Query.Model.Tags)
	tagIndexMap := make(map[int]int)
	for i := 0; i < length; i++ {
		if c.Query.Model.Tags[i].Condition == "OR" {
			andIndex := c.getTheAndOfConditionor(i + 1)
			if andIndex == i+1 {
				tagIndexMap[i] = i
				continue
			}
			if andIndex == length {
				for j := i; j < length; j++ {
					tagIndexMap[j] = j
				}
				break
			}
			tagIndexMap[i] = andIndex
			i = andIndex
		}
	}
	return tagIndexMap
}

func (c *QueryCondition) getFilterResources(start int, end int,
	resources []jsonutils.JSONObject) []jsonutils.JSONObject {
	relationMap := c.getTagKeyRelationMap()
	tmp := resources
	for i := start; i <= end; i++ {
		tag := c.Query.Model.Tags[i]
		if tag.Key == hostconsts.TELEGRAF_TAG_KEY_RES_TYPE {
			continue
		}
		relationKey := relationMap[tag.Key]
		if len(relationKey) == 0 {
			continue
		}
		filterObj := make([]jsonutils.JSONObject, 0)
		for _, res := range tmp {
			val, _ := res.GetString(relationKey)
			op := c.Query.Model.Tags[i].Operator
			if op == "=" || len(op) == 0 {
				if val == c.Query.Model.Tags[i].Value {
					filterObj = append(filterObj, res)
				}
			}
			if c.Query.Model.Tags[i].Operator == "!=" {
				if val != c.Query.Model.Tags[i].Value {
					filterObj = append(filterObj, res)
				}
			}
		}
		tmp = filterObj
		if len(tmp) == 0 {
			return tmp
		}
	}
	return tmp
}

func (c *QueryCondition) fillFilterRes(filterRes []jsonutils.JSONObject,
	filterIdMap map[string]jsonutils.JSONObject) map[string]jsonutils.JSONObject {
	for _, res := range filterRes {
		id, _ := res.GetString("id")
		if _, ok := filterIdMap[id]; !ok {
			filterIdMap[id] = res
		}
	}
	return filterIdMap
}

func (c *QueryCondition) getTheAndOfConditionor(start int) int {
	for i := start; i < len(c.Query.Model.Tags); i++ {
		if c.Query.Model.Tags[i].Condition != "AND" {
			return i
		}
	}
	return len(c.Query.Model.Tags)
}

func (c *QueryCondition) getTagKeyRelationMap() map[string]string {
	relationMap := make(map[string]string)
	switch c.ResType {
	case monitor.METRIC_RES_TYPE_HOST:
		relationMap = monitor.HostTags
	case monitor.METRIC_RES_TYPE_GUEST:
		relationMap = monitor.ServerTags
	case monitor.METRIC_RES_TYPE_RDS:
		relationMap = monitor.RdsTags
	case monitor.METRIC_RES_TYPE_REDIS:
		relationMap = monitor.RedisTags
	case monitor.METRIC_RES_TYPE_OSS:
		relationMap = monitor.OssTags
	case monitor.METRIC_RES_TYPE_CLOUDACCOUNT:
		relationMap = monitor.CloudAccountTags
	case monitor.METRIC_RES_TYPE_TENANT:
		relationMap = monitor.TenantTags
	case monitor.METRIC_RES_TYPE_DOMAIN:
		relationMap = monitor.DomainTags
	case monitor.METRIC_RES_TYPE_STORAGE:
		relationMap = monitor.StorageTags
	case monitor.METRIC_RES_TYPE_AGENT:
		relationMap = monitor.ServerTags
	default:
		relationMap = monitor.HostTags
	}
	return relationMap
}
