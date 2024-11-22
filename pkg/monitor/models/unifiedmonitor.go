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

package models

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/promql/v2/pkg/labels"
	"github.com/zexi/influxql-to-metricsql/converter/translator"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	mod "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	mq "yunion.io/x/onecloud/pkg/monitor/metricquery"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

const (
	TELEGRAF_DATABASE = "telegraf"
)

var (
	UnifiedMonitorManager *SUnifiedMonitorManager
)

func init() {
	UnifiedMonitorManager = &SUnifiedMonitorManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SUnifiedMonitorManager{},
			"",
			"unifiedmonitor",
			"unifiedmonitors",
		),
	}
	UnifiedMonitorManager.SetVirtualObject(UnifiedMonitorManager)
}

type SUnifiedMonitorManager struct {
	db.SVirtualResourceBaseManager
}

type SUnifiedMonitorModel struct {
}

func (self *SUnifiedMonitorManager) GetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return DataSourceManager.GetDatabases()
}

func (self *SUnifiedMonitorManager) GetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	filter, err := getTagFilterByRequestQuery(ctx, userCred, query)
	if err != nil {
		return nil, errors.Wrap(err, "getTagFilterByRequestQuery")
	}
	return DataSourceManager.GetMeasurementsWithDescriptionInfos(query, filter)
}

func getTagFilterByRequestQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*monitor.MetricQueryTag, error) {

	scope, _ := query.GetString("scope")
	return filterByScope(ctx, userCred, scope, query)
}

func filterByScope(ctx context.Context, userCred mcclient.TokenCredential, scope string, data jsonutils.JSONObject) (*monitor.MetricQueryTag, error) {
	domainId := jsonutils.GetAnyString(data, db.DomainFetchKeys)
	projectId := jsonutils.GetAnyString(data, db.ProjectFetchKeys)
	if projectId != "" {
		project, err := db.DefaultProjectFetcher(ctx, projectId, domainId)
		if err != nil {
			return nil, errors.Wrap(err, "db.DefaultProjectFetcher")
		}
		projectId = project.GetProjectId()
		domainId = project.GetProjectDomainId()
	}
	if domainId != "" {
		domain, err := db.DefaultDomainFetcher(ctx, domainId)
		if err != nil {
			return nil, errors.Wrap(err, "db.DefaultDomainFetcher")
		}
		domainId = domain.GetProjectDomainId()
		domain.GetProjectId()
	}
	switch scope {
	case "system":
		return nil, nil
	case "domain":
		if domainId == "" {
			domainId = userCred.GetProjectDomainId()
		}
		return getProjectIdsFilterByDomain(domainId)
	default:
		if projectId == "" {
			projectId = userCred.GetProjectId()
		}
		return getProjectIdFilterByProject(projectId)
	}
}

func getTenantIdStr(role string, userCred mcclient.TokenCredential) (*monitor.MetricQueryTag, error) {
	if role == "admin" {
		return nil, nil
	}
	if role == "domainadmin" {
		domainId := userCred.GetDomainId()
		return getProjectIdsFilterByDomain(domainId)
	}
	if role == "member" {
		tenantId := userCred.GetProjectId()
		return getProjectIdFilterByProject(tenantId)
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "not supported role %q", role)
}

func getProjectIdsFilterByDomain(domainId string) (*monitor.MetricQueryTag, error) {
	//s := auth.GetAdminSession(context.Background(), "", "")
	//params := jsonutils.Marshal(map[string]string{"domain_id": domainId})
	//tenants, err := modules.Projects.List(s, params)
	//if err != nil {
	//	return "", errors.Wrap(err, "Projects.List")
	//}
	//var buffer bytes.Buffer
	//buffer.WriteString("( ")
	//for index, tenant := range tenants.Data {
	//	tenantId, _ := tenant.GetString("id")
	//	if index != len(tenants.Data)-1 {
	//		buffer.WriteString(fmt.Sprintf(" %s =~ /%s/ %s ", "tenant_id", tenantId, "OR"))
	//	} else {
	//		buffer.WriteString(fmt.Sprintf(" %s =~ /%s/ ", "tenant_id", tenantId))
	//	}
	//}
	//buffer.WriteString(" )")
	//return buffer.String(), nil
	return &monitor.MetricQueryTag{
		Key:      "domain_id",
		Operator: "=~",
		Value:    fmt.Sprintf("/%s/", domainId),
	}, nil
	//return fmt.Sprintf(`%s =~ /%s/`, "domain_id", domainId), nil
}

func getProjectIdFilterByProject(projectId string) (*monitor.MetricQueryTag, error) {
	//return fmt.Sprintf(`%s =~ /%s/`, "tenant_id", projectId), nil
	return &monitor.MetricQueryTag{
		Key:      "tenant_id",
		Operator: "=~",
		Value:    fmt.Sprintf("/%s/", projectId),
	}, nil
}

func (self *SUnifiedMonitorManager) GetPropertyMetricMeasurement(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	metricFunc := monitor.MetricFunc{
		FieldOptType:  monitor.UNIFIED_MONITOR_FIELD_OPT_TYPE,
		FieldOptValue: monitor.UNIFIED_MONITOR_FIELD_OPT_VALUE,
		GroupOptType:  monitor.UNIFIED_MONITOR_GROUPBY_OPT_TYPE,
		GroupOptValue: monitor.UNIFIED_MONITOR_GROUPBY_OPT_VALUE,
	}
	filter, err := getTagFilterByRequestQuery(ctx, userCred, query)
	if err != nil {
		return nil, errors.Wrapf(err, "getTagFilterByRequestQuery %s", query.String())
	}
	rtn, err := DataSourceManager.GetMetricMeasurement(userCred, query, filter)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetricMeasurement by query %s, filter %s", query.String(), filter)
	}
	rtn.(*jsonutils.JSONDict).Add(jsonutils.Marshal(&metricFunc), "func")
	return rtn, nil
}

func (self *SUnifiedMonitorManager) SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration {
	return 5 * time.Minute
}

func (self *SUnifiedMonitorManager) PerformQuery(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (*monitor.MetricsQueryResult, error) {
	tmp := jsonutils.DeepCopy(data)
	self.handleDataPreSignature(ctx, tmp)
	if !options.Options.DisableQuerySignatureCheck {
		if err := ValidateQuerySignature(tmp); err != nil {
			return nil, errors.Wrap(err, "ValidateQuerySignature")
		}
	}
	inputQuery := new(monitor.MetricQueryInput)
	err := data.Unmarshal(inputQuery)
	if err != nil {
		return nil, err
	}
	if len(inputQuery.MetricQuery) == 0 {
		return nil, merrors.NewArgIsEmptyErr("metric_query")
	}
	for _, q := range inputQuery.MetricQuery {
		scope, _ := data.GetString("scope")
		ownId, _ := self.FetchOwnerId(ctx, data)
		if ownId == nil {
			ownId = userCred
		}
		setDefaultValue(q, inputQuery, scope, ownId, false)
		if err := self.ValidateInputQuery(q, inputQuery); err != nil {
			return nil, errors.Wrapf(err, "ValidateInputQuery")
		}
	}

	var groupByTag = make([]string, 0)
	for _, query := range inputQuery.MetricQuery {
		for _, group := range query.Model.GroupBy {
			if group.Type == "tag" {
				groupByTag = append(groupByTag, group.Params[0])
			}
		}
	}

	return self.performQuery(ctx, userCred, inputQuery)
}

func (self *SUnifiedMonitorManager) performQuery(ctx context.Context, userCred mcclient.TokenCredential, inputQuery *monitor.MetricQueryInput) (*monitor.MetricsQueryResult, error) {
	rtn, err := doQuery(userCred, *inputQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "doQuery with input %s", jsonutils.Marshal(inputQuery))
	}

	if len(inputQuery.Soffset) != 0 && len(inputQuery.Slimit) != 0 {
		// seriesTotal := self.fillSearchSeriesTotalQuery(userCred, *inputQuery.MetricQuery[0])
		// do offset and limit
		total := rtn.SeriesTotal
		offset, err := strconv.Atoi(inputQuery.Soffset)
		if err != nil {
			return nil, httperrors.NewInputParameterError("soffset %q is not integer", inputQuery.Soffset)
		}
		limit, err := strconv.Atoi(inputQuery.Slimit)
		if err != nil {
			return nil, httperrors.NewInputParameterError("slimit %q is not integer", inputQuery.Slimit)
		}
		start := offset
		end := start + limit
		if end > int(total) {
			end = int(total)
		}
		ss := rtn.Series
		if start >= end {
			rtn.Series = nil
		} else {
			rtn.Series = ss[start:end]
		}
	}

	fillSerieTags(&rtn.Series)
	return rtn, nil
}

func (self *SUnifiedMonitorManager) fillSearchSeriesTotalQuery(userCred mcclient.TokenCredential, fork monitor.AlertQuery) int64 {
	newGroupByPart := make([]monitor.MetricQueryPart, 0)
	newGroupByPart = append(newGroupByPart, fork.Model.GroupBy[0])
	fork.Model.GroupBy = newGroupByPart
	forkInputQury := new(monitor.MetricQueryInput)
	forkInputQury.MetricQuery = []*monitor.AlertQuery{&fork}
	rtn, err := doQuery(userCred, *forkInputQury)
	if err != nil {
		log.Errorf("exec forkInputQury err:%v", err)
		return 0
	}
	return int64(len(rtn.Series))
}

func (self *SUnifiedMonitorManager) handleDataPreSignature(ctx context.Context, data jsonutils.JSONObject) {
	scope, _ := data.GetString("scope")
	isIdentityName, _ := data.Bool("identity_name")
	switch scope {
	case "system":
	case "domain":
		domain, err := data.GetString("project_domain")
		if err == nil {
			domainObj, _ := db.DefaultDomainFetcher(ctx, domain)
			if isIdentityName {
				domain = domainObj.Name
			}
			data.(*jsonutils.JSONDict).Remove("project_domain")
			data.(*jsonutils.JSONDict).Set("domain_id", jsonutils.NewString(domain))
		}
	default:
		project, err := data.GetString("project")
		if err == nil {
			domain, _ := data.GetString("project_domain")
			tenant, _ := db.DefaultProjectFetcher(ctx, project, domain)
			if isIdentityName {
				project = tenant.Name
			}
			data.(*jsonutils.JSONDict).Remove("project")
			data.(*jsonutils.JSONDict).Set("project_id", jsonutils.NewString(project))
		}
	}
}

func doQuery(userCred mcclient.TokenCredential, query monitor.MetricQueryInput) (*monitor.MetricsQueryResult, error) {
	conds := make([]*monitor.AlertCondition, 0)
	for _, q := range query.MetricQuery {
		if q.To == "" {
			q.To = query.To
		}
		if q.From == "" {
			q.From = query.From
		}
		if q.Model.Interval == "" {
			q.Model.Interval = query.Interval
		}
		condition := monitor.AlertCondition{
			Type:  monitor.ConditionTypeMetricQuery,
			Query: *q,
		}
		if q.ResultReducer != nil {
			condition.Reducer = *q.ResultReducer
			condition.ReducerOrder = q.ResultReducerOrder
		}
		conds = append(conds, &condition)
	}
	factory := mq.GetQueryFactories()[monitor.ConditionTypeMetricQuery]
	metricQ, err := factory(conds)
	if err != nil {
		return nil, errors.Wrap(err, "factory")
	}
	metrics, err := metricQ.ExecuteQuery(userCred, query.Scope, query.SkipCheckSeries)
	if err != nil {
		return nil, errors.Wrap(err, "ExecuteQuery")
	}
	// drop metas contains raw_query
	if !query.ShowMeta {
		metrics.Metas = nil
	}
	metrics.SeriesTotal = int64(len(metrics.Series))
	return metrics, nil
}

func (self *SUnifiedMonitorManager) ValidateInputQuery(query *monitor.AlertQuery, input *monitor.MetricQueryInput) error {
	if input.From == "" {
		input.From = "1h"
	}
	if input.To == "" {
		input.To = "now"
	}
	if input.Interval == "" {
		input.Interval = "5m"
	}

	if query.From == "" {
		query.From = input.From
	}
	if query.Model.Interval == "" {
		query.Model.Interval = input.Interval
	}
	if query.To == "" {
		query.To = input.To
	}
	if _, err := time.ParseDuration(query.Model.Interval); err != nil {
		return httperrors.NewInputParameterError("Invalid interval format: %s", query.Model.Interval)
	}
	return validators.ValidateSelectOfMetricQuery(*query)
}

func setDefaultValue(
	query *monitor.AlertQuery,
	inputQuery *monitor.MetricQueryInput,
	scope string, ownerId mcclient.IIdentityProvider,
	isAlert bool) {
	query.From = inputQuery.From
	query.To = inputQuery.To
	query.Model.Interval = inputQuery.Interval

	metricMeasurement, _ := MetricMeasurementManager.GetCache().Get(query.Model.Measurement)

	checkQueryGroupBy(query, inputQuery, isAlert)

	if len(inputQuery.Interval) != 0 {
		query.Model.GroupBy = append(query.Model.GroupBy,
			monitor.MetricQueryPart{
				Type:   "time",
				Params: []string{"$interval"},
			},
			monitor.MetricQueryPart{
				Type:   "fill",
				Params: []string{"none"},
			})
	}

	// HACK: not set slimit and soffset, getting all series then do offset and limit
	// if len(inputQuery.Slimit) != 0 && len(inputQuery.Soffset) != 0 {
	// 	query.Model.GroupBy = append(query.Model.GroupBy,
	// 		monitor.MetricQueryPart{Type: "slimit", Params: []string{inputQuery.Slimit}},
	// 		monitor.MetricQueryPart{Type: "soffset", Params: []string{inputQuery.Soffset}},
	// 	)
	// }

	if query.Model.Database == "" {
		database := ""
		if metricMeasurement == nil {
			log.Warningf("Not found measurement %s from metrics measurement cache", query.Model.Measurement)
		} else {
			database = metricMeasurement.Database
		}
		if database == "" {
			// hack: query from default telegraf database if no metric measurement matched
			database = TELEGRAF_DATABASE
		}
		query.Model.Database = database
	}

	drv, _ := DataSourceManager.GetTSDBDriver()
	query = drv.FillSelect(query, isAlert)

	var projectId, domainId string
	switch rbacscope.TRbacScope(scope) {
	case rbacscope.ScopeProject:
		projectId = ownerId.GetProjectId()
		containId := false
		for _, tagFilter := range query.Model.Tags {
			if tagFilter.Key == "tenant_id" {
				containId = true
				break
			}
		}
		if !containId {
			query.Model.Tags = append(query.Model.Tags, monitor.MetricQueryTag{
				Key:       "tenant_id",
				Operator:  "=",
				Value:     projectId,
				Condition: "and",
			})
		}
	case rbacscope.ScopeDomain:
		domainId = ownerId.GetProjectDomainId()
		containId := false
		for _, tagFilter := range query.Model.Tags {
			if tagFilter.Key == "domain_id" {
				containId = true
				break
			}
		}
		if !containId {
			query.Model.Tags = append(query.Model.Tags, monitor.MetricQueryTag{
				Key:       "domain_id",
				Operator:  "=",
				Value:     domainId,
				Condition: "and",
			})
		}
	}
	if metricMeasurement != nil && metricMeasurement.ResType == hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE {
		query.Model.Tags = append(query.Model.Tags, monitor.MetricQueryTag{
			Key:       hostconsts.TELEGRAF_TAG_KEY_RES_TYPE,
			Operator:  "=",
			Value:     hostconsts.TELEGRAF_TAG_ONECLOUD_RES_TYPE,
			Condition: "and",
		})
	}
}

func checkQueryGroupBy(query *monitor.AlertQuery, inputQuery *monitor.MetricQueryInput, isAlert bool) {
	if len(query.Model.GroupBy) != 0 {
		return
	}
	metricMeasurement, _ := MetricMeasurementManager.GetCache().Get(query.Model.Measurement)
	if inputQuery.Unit || metricMeasurement == nil && query.Model.Database == monitor.METRIC_DATABASE_METER {
		return
	}
	tagId := ""
	if metricMeasurement != nil {
		tagId = monitor.GetMeasurementTagIdKeyByResType(metricMeasurement.ResType)
	}
	drv, _ := DataSourceManager.GetTSDBDriver()
	query = drv.FillGroupBy(query, inputQuery, tagId, isAlert)
}

func fillSerieTags(series *monitor.TimeSeriesSlice) {
	for i, serie := range *series {
		for _, tag := range []string{"brand", "platform", "hypervisor"} {
			if val, ok := serie.Tags[tag]; ok {
				serie.Tags["brand"] = val
				break
			}
		}
		for _, tag := range []string{
			"source", "status", hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE,
			hostconsts.TELEGRAF_TAG_KEY_RES_TYPE,
			"is_vm", "os_type", "domain_name", "region",
			labels.MetricName, translator.UNION_RESULT_NAME,
		} {
			if _, ok := serie.Tags[tag]; ok {
				delete(serie.Tags, tag)
			}
		}
		if val, ok := serie.Tags[VICTORIA_METRICS_DB_TAG_KEY]; ok {
			if val == VICTORIA_METRICS_DB_TAG_VAL_TELEGRAF {
				delete(serie.Tags, VICTORIA_METRICS_DB_TAG_KEY)
			}
		}
		(*series)[i] = serie
	}
}

func (self *SUnifiedMonitorManager) GetPropertySimpleQuery(ctx context.Context, userCred mcclient.TokenCredential, input *monitor.SimpleQueryInput) (jsonutils.JSONObject, error) {
	if len(input.Database) == 0 {
		input.Database = "telegraf"
	}
	if len(input.MetricName) == 0 {
		return nil, httperrors.NewMissingParameterError("metric_name")
	}
	metric := strings.Split(input.MetricName, ".")
	if len(metric) != 2 {
		return nil, httperrors.NewInputParameterError("invalid metric_name %s", input.MetricName)
	}
	measurement, field := metric[0], metric[1]

	data := mod.NewMetricQueryInputWithDB(input.Database, measurement).SkipCheckSeries(true)
	data.Selects().Select(field)

	where := data.Where()
	if len(input.Id) > 0 {
		where.Equal("id", input.Id)
	}
	if input.EndTime.IsZero() {
		input.EndTime = time.Now()
	}
	if input.StartTime.IsZero() {
		input.StartTime = input.EndTime.Add(time.Hour * -1)
	}
	if input.EndTime.Sub(input.StartTime).Hours() > 1 {
		return nil, httperrors.NewInputParameterError("The query interval is greater than one hour")
	}
	for k, v := range input.Tags {
		where.Equal(k, v)
	}
	data.From(input.StartTime).To(input.EndTime).Interval("5m")

	queryData := data.ToQueryData()
	dbRtn, err := self.performQuery(ctx, userCred, queryData)
	if err != nil {
		return nil, errors.Wrapf(err, "performQuery with data: %s", jsonutils.Marshal(queryData))
	}
	ret := []monitor.SimpleQueryOutput{}

	for _, s := range dbRtn.Series {
		id, ok := s.Tags["id"]
		if !ok {
			log.Warningf("Not found id from series: %s", jsonutils.Marshal(s))
			continue
		}
		for _, point := range s.Points {
			if len(point) != 2 {
				log.Warningf("invalid series: %s", jsonutils.Marshal(s))
				break
			}
			timestamp := point[len(point)-1]
			valPtr, ok := point[0].(*float64)
			if !ok || valPtr == nil {
				log.Warningf("invalid series point: %#v", point)
				break
			}
			ret = append(ret, monitor.SimpleQueryOutput{
				Id:    id,
				Time:  time.UnixMilli(int64(timestamp.(float64))),
				Value: *valPtr,
			})
		}
	}
	return jsonutils.Marshal(map[string]interface{}{"values": ret}), nil
}
