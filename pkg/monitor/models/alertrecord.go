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
	"database/sql"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/filterclause"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AlertRecordManager *SAlertRecordManager
)

type SAlertRecordManager struct {
	db.SEnabledResourceBaseManager
	db.SStandaloneAnonResourceBaseManager
	SMonitorScopedResourceManager
}

type SAlertRecord struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStandaloneAnonResourceBase
	SMonitorScopedResource

	AlertId   string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Level     string               `charset:"ascii" width:"36" nullable:"false" default:"normal" list:"user" update:"user"`
	State     string               `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" update:"user"`
	SendState string               `width:"36" charset:"ascii" default:"ok" list:"user" update:"user"`
	EvalData  jsonutils.JSONObject `list:"user" update:"user" length:"medium"`
	AlertRule jsonutils.JSONObject `list:"user" update:"user" length:"medium"`
	ResType   string               `width:"36" list:"user" update:"user"`
	ResIds    string               `length:"medium" list:"user"`
}

func init() {
	AlertRecordManager = &SAlertRecordManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SAlertRecord{},
			"alertrecord_tbl",
			"alertrecord",
			"alertrecords",
		),
	}

	AlertRecordManager.SetVirtualObject(AlertRecordManager)
}

func (manager *SAlertRecordManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SAlertRecordManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneAnonResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (m *SAlertRecordManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return m.SMonitorScopedResourceManager.FilterByOwner(ctx, q, man, userCred, ownerId, scope)
}

func (manager *SAlertRecordManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertRecordListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	if query.Alerting {
		alertingQuery := manager.getAlertingRecordQuery().SubQuery()

		q.Join(alertingQuery, sqlchemy.Equals(q.Field("alert_id"), alertingQuery.Field("alert_id"))).Filter(
			sqlchemy.Equals(q.Field("created_at"), alertingQuery.Field("max_created_at")))
	}
	if len(query.Level) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("level"), query.Level))
	}
	if len(query.State) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("state"), query.State))
	}
	if len(query.AlertId) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("alert_id"), query.AlertId))
	} else {
		q = q.IsNotEmpty("res_type").IsNotNull("res_type")
	}
	if len(query.AlertName) != 0 {
		alertQ := GetCommonAlertManager().Query("id").Contains("name", query.AlertName).SubQuery()
		q = q.In("alert_id", alertQ)
	}
	if len(query.ResType) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("res_type"), query.ResType))
	}
	if len(query.ResId) != 0 {
		q.Filter(sqlchemy.ContainsAny(q.Field("res_ids"), []string{query.ResId}))
	}
	if len(query.Filter) != 0 {
		for i, _ := range query.Filter {
			if strings.Contains(query.Filter[i], "trigger_time") {
				timeFilter := strings.ReplaceAll(query.Filter[i], "trigger_time", "created_at")
				filterClause := filterclause.ParseFilterClause(timeFilter)
				q.Filter(filterClause.QueryCondition(q))
			}
		}
	}
	return q, nil
}

func (man *SAlertRecordManager) getAlertingRecordQuery() *sqlchemy.SQuery {
	q := CommonAlertManager.Query("id").IsTrue("enabled").IsNull("used_by")
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("state"), monitor.AlertStateAlerting),
		sqlchemy.Equals(q.Field("state"), monitor.AlertStatePending)))

	alertsQuery := q.SubQuery()
	recordSub := man.Query().SubQuery()

	recordQuery := recordSub.Query(recordSub.Field("alert_id"), sqlchemy.MAX("max_created_at", recordSub.Field("created_at")))
	recordQuery.Equals("state", monitor.AlertStateAlerting)
	recordQuery.In("alert_id", alertsQuery)
	recordQuery.IsNotNull("res_type").IsNotEmpty("res_type")
	recordQuery.GroupBy("alert_id")
	return recordQuery

}

func (man *SAlertRecordManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()

	input := new(monitor.AlertRecordListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "AlertRecordManager.CustomizeFilterList.Unmarshal")
	}
	if input.ResName != "" {
		filters.Append(func(item jsonutils.JSONObject) (bool, error) {
			evalMatchs := make([]monitor.EvalMatch, 0)
			if item.Contains("eval_data") {
				err := item.Unmarshal(&evalMatchs, "eval_data")
				if err != nil {
					return false, errors.Wrap(err, "record Unmarshal evalMatchs error")
				}
				for _, match := range evalMatchs {
					for k, v := range match.Tags {
						if strings.Contains(k, "name") && strings.Contains(v, input.ResName) {
							return true, nil
						}
					}
				}
			}
			return false, nil
		})
	}
	return filters, nil
}

func (man *SAlertRecordManager) GetAlertRecord(id string) (*SAlertRecord, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SAlertRecord), nil
}

func (man *SAlertRecordManager) GetAlertRecordsByAlertId(id string) ([]SAlertRecord, error) {
	records := make([]SAlertRecord, 0)
	query := man.Query()
	query = query.Equals("alert_id", id)
	err := db.FetchModelObjects(man, query, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (manager *SAlertRecordManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	if field == "res_type" {
		resTypeQuery := MetricMeasurementManager.Query("res_type").Distinct()
		return resTypeQuery, nil
	}
	q, err = manager.SStandaloneAnonResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (man *SAlertRecordManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertRecordListInput,
) (*sqlchemy.SQuery, error) {
	/*var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	/*q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}*/
	return q, nil
}

func (man *SAlertRecordManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertRecordDetails {
	rows := make([]monitor.AlertRecordDetails, len(objs))
	stdRows := man.SStandaloneAnonResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	alertIds := sets.NewString()
	records := make([]*SAlertRecord, len(objs))
	for i := range rows {
		rows[i] = monitor.AlertRecordDetails{
			StandaloneAnonResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:        scopedRows[i],
		}
		record := objs[i].(*SAlertRecord)
		alertIds.Insert(record.AlertId)
		records[i] = record
	}

	alerts := map[string]SCommonAlert{}
	err := db.FetchModelObjectsByIds(CommonAlertManager, "id", alertIds.List(), &alerts)
	if err != nil {
		log.Errorf("FetchModelObjectsByIds common alert error: %v", err)
		return rows
	}
	for i := range rows {
		if alert, ok := alerts[records[i].AlertId]; ok {
			rows[i].AlertName = alert.Name
			rows[i].TriggerTime = records[i].CreatedAt
		}
		evalMatches, err := records[i].GetEvalData()
		if err != nil {
			continue
		}
		for j := range evalMatches {
			evalMatches[j] = records[i].filterTags(evalMatches[j])
		}
		rows[i].ResNum = int64(len(evalMatches))
	}
	return rows
}

func (record *SAlertRecord) filterTags(match monitor.EvalMatch) monitor.EvalMatch {
	for key, _ := range match.Tags {
		if strings.HasSuffix(key, "_id") {
			delete(match.Tags, key)
		}
	}
	return match
}

func (man *SAlertRecordManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, data monitor.AlertRecordCreateInput) (monitor.AlertRecordCreateInput, error) {
	return data, nil
}

func (record *SAlertRecord) GetEvalData() ([]monitor.EvalMatch, error) {
	ret := make([]monitor.EvalMatch, 0)
	if record.EvalData == nil {
		return ret, nil
	}
	if err := record.EvalData.Unmarshal(&ret); err != nil {
		return nil, errors.Wrap(err, "unmarshal evalMatchs error")
	}
	return ret, nil
}

func (record *SAlertRecord) CustomizeCreate(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	/*err := record.SMonitorScopedResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}*/
	alert, err := AlertManager.GetAlert(record.AlertId)
	if err != nil {
		return errors.Wrapf(err, "GetAlert %s", record.AlertId)
	}
	record.DomainId = alert.GetDomainId()
	record.ProjectId = alert.GetProjectId()
	obj, err := db.NewModelObject(AlertRecordManager)
	if err != nil {
		return errors.Wrapf(err, "NewModelObject %s", AlertRecordManager.Keyword())
	}
	q := AlertRecordManager.Query().Equals("alert_id", record.AlertId).Desc("created_at")
	if err := q.First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		} else {
			return errors.Wrapf(err, "Get latest alertrecord error by alertId:%s", record.AlertId)
		}
	}
	latestRecord := obj.(*SAlertRecord)
	if latestRecord.GetState() == monitor.AlertStateAlerting && record.GetState() == monitor.AlertStateOK {
		err := record.unionEvalMatch(latestRecord)
		if err != nil {
			return errors.Wrap(err, "unionEvalMatch error")
		}
	}
	// fill res_ids
	matches, err := record.GetEvalData()
	if err != nil {
		return errors.Wrap(err, "GetEvalData error")
	}
	resIds := []string{}
	for _, match := range matches {
		for k, v := range match.Tags {
			switch k {
			case "vm_id":
				if record.ResType == monitor.METRIC_RES_TYPE_AGENT || record.ResType == monitor.METRIC_RES_TYPE_GUEST {
					resIds = append(resIds, v)
				}
			case "host_id":
				if record.ResType == monitor.METRIC_RES_TYPE_HOST {
					resIds = append(resIds, v)
				}
			case "cloudaccount_id":
				if record.ResType == monitor.METRIC_RES_TYPE_CLOUDACCOUNT {
					resIds = append(resIds, v)
				}
			case "id":
				resIds = append(resIds, v)
			}
		}
	}
	record.ResIds = strings.Join(resIds, ",")
	return nil
}

func (record *SAlertRecord) unionEvalMatch(alertingRecord *SAlertRecord) error {
	matches, err := record.GetEvalData()
	if err != nil {
		return err
	}
	alertingMatches, err := alertingRecord.GetEvalData()
	if err != nil {
		return err
	}
	newEvalMatchs := make([]monitor.EvalMatch, 0)
getNewMatchTag:
	for i, _ := range matches {
		for j, _ := range alertingMatches {
			if matches[i].Tags["name"] == alertingMatches[j].Tags["name"] {
				newEvalMatchs = append(newEvalMatchs, matches[i])
				continue getNewMatchTag
			}
		}
	}
	record.EvalData = jsonutils.Marshal(&newEvalMatchs)
	return nil
}

func (record *SAlertRecord) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	record.SStandaloneAnonResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	err := MonitorResourceManager.UpdateMonitorResourceAttachJointByRecord(ctx, userCred, record)
	if err != nil {
		log.Errorf("UpdateMonitorResourceAttachJointByRecord error: %v", err)
	}
	if err := GetAlertResourceManager().ReconcileFromRecord(ctx, userCred, ownerId, record); err != nil {
		log.Errorf("Reconcile from alert record error: %v", err)
		return
	}

	// err = GetAlertResourceManager().NotifyAlertResourceCount(ctx)
	// if err != nil {
	//	log.Errorf("NotifyAlertResourceCount error: %v", err)
	//	return
	// }
}

func (record *SAlertRecord) GetState() monitor.AlertStateType {
	return monitor.AlertStateType(record.State)
}

func (manager *SAlertRecordManager) DeleteRecordsOfThirtyDaysAgo(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	records := make([]SAlertRecord, 0)
	query := manager.Query()
	query = query.LE("created_at", timeutils.MysqlTime(time.Now().Add(-time.Hour*24*30)))
	err := db.FetchModelObjects(manager, query, &records)
	if err != nil {
		log.Errorf("fetch records ofthirty days ago err:%v", err)
		return
	}
	for i, _ := range records {
		err := db.DeleteModel(ctx, userCred, &records[i])
		if err != nil {
			log.Errorf("delete expire record: %s err: %v", records[i].GetId(), err)
		}
	}
}

func (manager *SAlertRecordManager) GetPropertyTotalAlert(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(monitor.AlertRecordListInput)
	err := query.Unmarshal(input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal AlertRecordListInput error")
	}
	alertRess, err := MonitorResourceAlertManager.GetNowAlertingAlerts(ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "getNowAlertingRecord error")
	}
	alertCountMap := jsonutils.NewDict()
	for _, res := range alertRess {
		var count int64 = 1
		if alertCountMap.Contains(res.ResType) {
			resTypeCount, _ := alertCountMap.Int(res.ResType)
			count = count + resTypeCount
		}
		alertCountMap.Set(res.ResType, jsonutils.NewInt(count))
	}
	return alertCountMap, nil
}

// 获取过去一天的报警历史分布
func (manager *SAlertRecordManager) GetPropertyHistoryAlert(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (*monitor.AlertRecordHistoryAlert, error) {
	q := manager.Query().GE("created_at", time.Now().Add(-time.Hour*24)).IsNotEmpty("eval_data").NotEquals("state", monitor.AlertStateOK)
	alerts := []SAlertRecord{}
	err := q.All(&alerts)
	if err != nil {
		return nil, errors.Wrap(err, "q.All")
	}
	ret := map[string]map[string]map[string]int64{}
	domainIds, projectIds := []string{}, []string{}
	domainMap := map[string]string{}
	projectMap := map[string]string{}
	for _, alert := range alerts {
		if _, ok := ret[alert.DomainId]; !ok {
			ret[alert.DomainId] = map[string]map[string]int64{}
			domainIds = append(domainIds, alert.DomainId)
		}
		if _, ok := ret[alert.DomainId][alert.ProjectId]; !ok {
			ret[alert.DomainId][alert.ProjectId] = map[string]int64{}
			projectIds = append(projectIds, alert.ProjectId)
		}
		if _, ok := ret[alert.DomainId][alert.ProjectId][alert.ResType]; !ok {
			ret[alert.DomainId][alert.ProjectId][alert.ResType] = 0
		}
		eval := make([]monitor.EvalMatch, 0)
		err := alert.EvalData.Unmarshal(&eval)
		if err != nil {
			continue
		}
		ret[alert.DomainId][alert.ProjectId][alert.ResType] += int64(len(eval))
	}
	domains := []db.STenant{}
	err = db.TenantCacheManager.GetDomainQuery().In("id", domainIds).All(&domains)
	if err != nil {
		return nil, errors.Wrap(err, "GetDomainQuery.In.All")
	}
	for _, domain := range domains {
		domainMap[domain.Id] = domain.Name
	}

	projects := []db.STenant{}
	err = db.TenantCacheManager.GetTenantQuery().In("id", projectIds).All(&projects)
	if err != nil {
		return nil, errors.Wrap(err, "GetTenantQuery.In.All")
	}
	for _, project := range projects {
		projectMap[project.Id] = project.Name
	}
	result := &monitor.AlertRecordHistoryAlert{}
	for domainId, projects := range ret {
		for projectId, resTypes := range projects {
			for resType, alert := range resTypes {
				result.Data = append(result.Data, monitor.AlertRecordHistoryAlertData{
					ProjectId: projectId,
					Project:   projectMap[projectId],
					DomainId:  domainId,
					Domain:    domainMap[domainId],
					ResType:   resType,
					ResNum:    alert,
				})
			}
		}
	}
	return result, nil
}

// GetPropertyProjectAlertResourceCount 获取指定时间段内各项目下的报警资源数量
func (manager *SAlertRecordManager) GetPropertyProjectAlertResourceCount(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input monitor.ProjectAlertResourceCountInput,
) (*monitor.ProjectAlertResourceCount, error) {
	// 验证时间段参数
	if input.StartTime.IsZero() || input.EndTime.IsZero() {
		return nil, httperrors.NewInputParameterError("start_time and end_time must be specified")
	}
	if input.StartTime.After(input.EndTime) {
		return nil, httperrors.NewInputParameterError("start_time must be before end_time")
	}

	// 构建查询
	q := manager.Query()
	q = q.GE("created_at", input.StartTime).LE("created_at", input.EndTime)
	q = q.IsNotEmpty("res_ids")

	// 应用权限过滤
	scope := rbacscope.ScopeSystem
	if input.Scope != "" {
		scope = rbacscope.TRbacScope(input.Scope)
	}
	q = manager.SMonitorScopedResourceManager.FilterByOwner(ctx, q, manager, userCred, userCred, scope)

	// 如果指定了 ResType，添加过滤条件
	if input.ResType != "" {
		q = q.Equals("res_type", input.ResType)
	}

	// 如果指定了 AlertId，添加过滤条件
	if input.AlertId != "" {
		q = q.Equals("alert_id", input.AlertId)
	}

	// 执行查询获取所有记录
	alerts := make([]SAlertRecord, 0)
	err := q.All(&alerts)
	if err != nil {
		return nil, errors.Wrap(err, "query alert records")
	}

	// 按 scope 分组统计唯一资源数量
	// systemResourceSet = set of resource IDs (system scope)
	// domainResourceSet[domainId] = set of resource IDs (domain scope)
	// projectResourceSet[domainId][projectId] = set of resource IDs (project scope)
	systemResourceSet := sets.NewString()
	domainResourceSet := make(map[string]sets.String)
	projectResourceSet := make(map[string]map[string]sets.String)
	domainIds := sets.NewString()
	projectIds := sets.NewString()

	for _, alert := range alerts {
		if len(alert.ResIds) == 0 {
			continue
		}
		domainId := alert.DomainId
		projectId := alert.ProjectId

		// 解析 res_ids（逗号分隔）
		resIds := strings.Split(alert.ResIds, ",")
		for _, resId := range resIds {
			resId = strings.TrimSpace(resId)
			if len(resId) == 0 {
				continue
			}

			// 根据 domainId 和 projectId 判断 scope
			if domainId == "" && projectId == "" {
				// system scope
				systemResourceSet.Insert(resId)
			} else if domainId != "" && projectId == "" {
				// domain scope
				domainIds.Insert(domainId)
				if domainResourceSet[domainId] == nil {
					domainResourceSet[domainId] = sets.NewString()
				}
				domainResourceSet[domainId].Insert(resId)
			} else if domainId != "" && projectId != "" {
				// project scope
				domainIds.Insert(domainId)
				projectIds.Insert(projectId)
				if projectResourceSet[domainId] == nil {
					projectResourceSet[domainId] = make(map[string]sets.String)
				}
				if projectResourceSet[domainId][projectId] == nil {
					projectResourceSet[domainId][projectId] = sets.NewString()
				}
				projectResourceSet[domainId][projectId].Insert(resId)
			}
		}
	}

	// 获取项目和域的名称
	domainMap := make(map[string]string)
	if domainIds.Len() > 0 {
		domains := []db.STenant{}
		err = db.TenantCacheManager.GetDomainQuery().In("id", domainIds.List()).All(&domains)
		if err != nil {
			return nil, errors.Wrap(err, "GetDomainQuery.In.All")
		}
		for _, domain := range domains {
			domainMap[domain.Id] = domain.Name
		}
	}

	projectMap := make(map[string]string)
	if projectIds.Len() > 0 {
		projects := []db.STenant{}
		err = db.TenantCacheManager.GetTenantQuery().In("id", projectIds.List()).All(&projects)
		if err != nil {
			return nil, errors.Wrap(err, "GetTenantQuery.In.All")
		}
		for _, project := range projects {
			projectMap[project.Id] = project.Name
		}
	}

	// 构建返回结果
	result := &monitor.ProjectAlertResourceCount{
		Data: make([]monitor.ProjectAlertResourceCountData, 0),
	}

	// system scope
	if systemResourceSet.Len() > 0 {
		result.Data = append(result.Data, monitor.ProjectAlertResourceCountData{
			Scope:    string(rbacscope.ScopeSystem),
			ResCount: int64(systemResourceSet.Len()),
		})
	}

	// domain scope
	for domainId, resourceSet := range domainResourceSet {
		if resourceSet.Len() > 0 {
			result.Data = append(result.Data, monitor.ProjectAlertResourceCountData{
				Scope:    string(rbacscope.ScopeDomain),
				DomainId: domainId,
				Domain:   domainMap[domainId],
				ResCount: int64(resourceSet.Len()),
			})
		}
	}

	// project scope
	for domainId, projects := range projectResourceSet {
		for projectId, resourceSet := range projects {
			if resourceSet.Len() > 0 {
				result.Data = append(result.Data, monitor.ProjectAlertResourceCountData{
					Scope:     string(rbacscope.ScopeProject),
					DomainId:  domainId,
					Domain:    domainMap[domainId],
					ProjectId: projectId,
					Project:   projectMap[projectId],
					ResCount:  int64(resourceSet.Len()),
				})
			}
		}
	}

	return result, nil
}
