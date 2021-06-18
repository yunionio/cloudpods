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

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mc_mds "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
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
	Evaluator     AlertEvaluator
	Operator      string
	HandleRequest tsdb.HandleRequestFunc
	ResType       string
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

	allResources, err := c.GetQueryResources()
	if err != nil {
		return nil, errors.Wrap(err, "GetQueryResources err")
	}
	for _, series := range seriesList {
		if len(c.ResType) != 0 {
			isLatestOfSerie, resource := c.serieIsLatestResource(allResources, series)
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
			match, err := c.NewEvalMatch(context, *series, meta, reducedValue, valStrArr)
			if err != nil {
				return nil, errors.Wrap(err, "NewEvalMatch error")
			}
			matches = append(matches, match)
		}
		if reducedValue != nil && !evalMatch {
			match, err := c.NewEvalMatch(context, *series, meta, reducedValue, valStrArr)
			if err != nil {
				return nil, errors.Wrap(err, "NewEvalMatch error")
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

	return &alerting.ConditionResult{
		Firing:             evalMatchCount > 0,
		NoDataFound:        emptySeriesCount == len(seriesList),
		Operator:           c.Operator,
		EvalMatches:        matches,
		AlertOkEvalMatches: alertOkmatches,
	}, nil
}

func (c *QueryCondition) serieIsLatestResource(resources []jsonutils.JSONObject,
	series *tsdb.TimeSeries) (bool, jsonutils.JSONObject) {
	tagId := monitor.MEASUREMENT_TAG_ID[c.ResType]
	if len(tagId) == 0 {
		tagId = "host_id"
	}
	seriId := series.Tags[tagId]
	for _, resource := range resources {
		id, _ := resource.GetString("id")
		if seriId == id {
			return true, resource
		}
	}
	return false, nil
}

func (c *QueryCondition) FillSerieByResourceField(resource jsonutils.JSONObject,
	series *tsdb.TimeSeries) {
	tagKeyRelationMap := c.getTagKeyRelationMap()
	fieldMap, _ := resource.GetMap()
	for field, v := range fieldMap {
		val, _ := v.GetString()
		for tagKey, resourceKey := range tagKeyRelationMap {
			if resourceKey == field {
				series.Tags[tagKey] = val
			}
		}
	}
}

func (c *QueryCondition) NewEvalMatch(context *alerting.EvalContext, series tsdb.TimeSeries,
	meta *tsdb.QueryResultMeta, value *float64, valStrArr []string) (*monitor.EvalMatch, error) {
	evalMatch := new(monitor.EvalMatch)
	alertDetails, err := c.GetCommonAlertDetails(context)
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
	evalMatch.ValueStr = alerting.RationalizeValueFromUnit(*value, alertDetails.FieldDescription.Unit,
		alertDetails.FieldOpt)
	if alertDetails.GetPointStr {
		evalMatch.ValueStr = c.jointPointStr(series, evalMatch.ValueStr, valStrArr)
	}
	c.FetchCustomizeEvalMatch(context, evalMatch, alertDetails)
	//c.newRuleDescription(context, alertDetails)
	//evalMatch.Condition = c.GenerateFormatCond(meta, queryKeyInfo).String()
	msg := fmt.Sprintf("%s.%s %s %s", alertDetails.Measurement, alertDetails.Field,
		alertDetails.Comparator, alerting.RationalizeValueFromUnit(alertDetails.Threshold, evalMatch.Unit, ""))
	if len(context.Rule.Message) == 0 {
		context.Rule.Message = msg
	}
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

func (c *QueryCondition) GetCommonAlertDetails(context *alerting.EvalContext) (*monitor.CommonAlertMetricDetails, error) {
	alert, err := models.CommonAlertManager.GetAlert(context.Rule.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetAlert to NewEvalMatch error")
	}
	settings, _ := alert.GetSettings()
	alertDetails := alert.GetCommonAlertMetricDetailsFromAlertCondition(c.Index, &settings.Conditions[c.Index])
	return alertDetails, nil
}

func (c *QueryCondition) jointPointStr(series tsdb.TimeSeries, value string, valStrArr []string) string {
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
	series tsdb.TimeSeriesSlice
	metas  []tsdb.QueryResultMeta
}

func (c *QueryCondition) executeQuery(evalCtx *alerting.EvalContext, timeRange *tsdb.TimeRange) (*queryResult, error) {
	ds, err := models.DataSourceManager.GetSource(c.Query.DataSourceId)
	if err != nil {
		return nil, errors.Wrapf(err, "Cound not find datasource %v", c.Query.DataSourceId)
	}

	req := c.getRequestForAlertRule(ds, timeRange, evalCtx.IsDebug)
	result := make(tsdb.TimeSeriesSlice, 0)
	metas := make([]tsdb.QueryResultMeta, 0)

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

	resp, err := c.HandleRequest(evalCtx.Ctx, ds.ToTSDBDataSource(c.Query.Model.Database), req)
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
			c.Query.Model.GroupBy[i].Params = []string{monitor.MEASUREMENT_TAG_ID[metricMeasurement.ResType]}
		}
	}
}

func (c *QueryCondition) setResType() {
	var resType = monitor.METRIC_RES_TYPE_HOST
	if len(c.Query.Model.GroupBy) == 0 {
		return
	}
	metricMeasurement, _ := models.MetricMeasurementManager.GetCache().Get(c.Query.Model.Measurement)
	if metricMeasurement != nil {
		resType = metricMeasurement.ResType
		c.ResType = resType
	}
	if len(resType) != 0 && c.Query.Model.GroupBy[0].Params[0] != monitor.
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
	}
	if c.Query.Model.Database == monitor.METRIC_DATABASE_TELE {
		c.ResType = resType
	}
}

func (c *QueryCondition) GetQueryResources() ([]jsonutils.JSONObject, error) {
	allHosts, err := c.getOnecloudResources()
	if err != nil {
		return nil, errors.Wrap(err, "getOnecloudHosts error")
	}
	allHosts = c.filterAllResources(allHosts)
	return allHosts, nil
}

func (c *QueryCondition) getOnecloudResources() ([]jsonutils.JSONObject, error) {
	var err error
	allResources := make([]jsonutils.JSONObject, 0)

	query := jsonutils.NewDict()
	query.Add(jsonutils.NewStringArray([]string{"running", "ready"}), "status")
	query.Add(jsonutils.NewString("true"), "admin")
	//if len(c.Query.Model.Tags) != 0 {
	//	query, err = c.convertTagsQuery(evalContext, query)
	//	if err != nil {
	//		return nil, errors.Wrap(err, "NoDataQueryCondition convertTagsQuery error")
	//	}
	//}
	switch c.ResType {
	case monitor.METRIC_RES_TYPE_HOST:
		query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	case monitor.METRIC_RES_TYPE_GUEST:
		allResources, err = ListAllResources(&mc_mds.Servers, query)
	case monitor.METRIC_RES_TYPE_AGENT:
		allResources, err = ListAllResources(&mc_mds.Servers, query)
	case monitor.METRIC_RES_TYPE_RDS:
		allResources, err = ListAllResources(&mc_mds.DBInstance, query)
	case monitor.METRIC_RES_TYPE_REDIS:
		allResources, err = ListAllResources(&mc_mds.ElasticCache, query)
	case monitor.METRIC_RES_TYPE_OSS:
		allResources, err = ListAllResources(&mc_mds.Buckets, query)
	case monitor.METRIC_RES_TYPE_CLOUDACCOUNT:
		query.Remove("status")
		query.Add(jsonutils.NewBool(true), "enabled")
		allResources, err = ListAllResources(&mc_mds.Cloudaccounts, query)
	case monitor.METRIC_RES_TYPE_TENANT:
		allResources, err = ListAllResources(&mc_mds.Projects, query)
	case monitor.METRIC_RES_TYPE_DOMAIN:
		allResources, err = ListAllResources(&mc_mds.Domains, query)
	case monitor.METRIC_RES_TYPE_STORAGE:
		query.Remove("status")
		allResources, err = ListAllResources(&mc_mds.Storages, query)
	default:
		query := jsonutils.NewDict()
		query.Set("brand", jsonutils.NewString(hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND))
		query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	}

	if err != nil {
		return nil, errors.Wrap(err, "NoDataQueryCondition Host list error")
	}
	return allResources, nil
}

func ListAllResources(manager modulebase.Manager, params *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.NewInt(0), "limit")
	params.Add(jsonutils.NewBool(true), "details")
	var count int
	session := auth.GetAdminSession(context.Background(), "", "")
	objs := make([]jsonutils.JSONObject, 0)
	for {
		params.Set("offset", jsonutils.NewInt(int64(count)))
		result, err := manager.List(session, params)
		if err != nil {
			return nil, errors.Wrapf(err, "list %s resources with params %s", manager.KeyString(), params.String())
		}
		for _, data := range result.Data {
			objs = append(objs, data)
		}
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
			if c.Query.Model.Tags[i].Operator == "=" {
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
