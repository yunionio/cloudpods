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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	MonitorResourceAlertManager *SMonitorResourceAlertManager
)

func init() {
	MonitorResourceAlertManager = &SMonitorResourceAlertManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			SMonitorResourceAlert{},
			"monitor_resource_alert_tbl",
			"monitorresourcealert",
			"monitorresourcealerts",
			MonitorResourceManager, CommonAlertManager),
	}
	MonitorResourceAlertManager.SetVirtualObject(MonitorResourceAlertManager)
}

// +onecloud:swagger-gen-model-singular=monitorresourcealert
// +onecloud:swagger-gen-model-plural=monitorresourcealerts
type SMonitorResourceAlertManager struct {
	db.SJointResourceBaseManager
	SMonitorScopedResourceManager
}

type SMonitorResourceAlert struct {
	db.SJointResourceBase

	MonitorResourceId string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true" json:"monitor_resource_id"`
	AlertId           string               `width:"36" charset:"ascii" list:"user" create:"required" index:"true"`
	AlertRecordId     string               `width:"36" charset:"ascii" list:"user" update:"user"`
	ResType           string               `width:"36" charset:"ascii" list:"user" update:"user" json:"res_type"`
	Metric            string               `width:"36" charset:"ascii" list:"user" create:"required" json:"metric"`
	AlertState        string               `width:"18" charset:"ascii" default:"init" list:"user" update:"user"`
	SendState         string               `width:"18" charset:"ascii" default:"ok" list:"user" update:"user"`
	TriggerTime       time.Time            `list:"user"  update:"user" json:"trigger_time"`
	Data              jsonutils.JSONObject `list:"user"  update:"user"`
}

func (manager *SMonitorResourceAlertManager) GetMasterFieldName() string {
	return "monitor_resource_id"
}

func (manager *SMonitorResourceAlertManager) GetSlaveFieldName() string {
	return "alert_id"
}

func (manager *SMonitorResourceAlertManager) DetachJoint(ctx context.Context, userCred mcclient.TokenCredential,
	input monitor.MonitorResourceJointListInput) error {
	joints, err := manager.GetJoinsByListInput(input)
	if err != nil {
		return errors.Wrapf(err, "SMonitorResourceAlertManager DetachJoint when  GetJoinsByListInput err,input:%v", input)
	}
	errs := make([]error, 0)
	for _, joint := range joints {
		err := joint.Delete(ctx, nil)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "joint %s:%s ,%s:%s", manager.GetMasterFieldName(),
				joint.MonitorResourceId, manager.GetSlaveFieldName(), joint.AlertId))
			continue
		}
		resources, err := MonitorResourceManager.GetMonitorResources(monitor.MonitorResourceListInput{ResId: []string{joint.
			MonitorResourceId}})
		if err != nil {
			errs = append(errs, err)
		}
		for _, res := range resources {
			err := res.UpdateAlertState()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.NewAggregate(errs)
}

func (obj *SMonitorResourceAlert) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SMonitorResourceAlert) GetAlert() (*SCommonAlert, error) {
	sObj, err := CommonAlertManager.FetchById(obj.AlertId)
	if err != nil {
		return nil, err
	}
	return sObj.(*SCommonAlert), nil
}

func (manager *SMonitorResourceAlertManager) GetJoinsByListInput(input monitor.MonitorResourceJointListInput) ([]SMonitorResourceAlert, error) {
	joints := make([]SMonitorResourceAlert, 0)
	query := manager.Query()

	if len(input.MonitorResourceId) != 0 {
		query.Equals(manager.GetMasterFieldName(), input.MonitorResourceId)
	}
	if len(input.AlertId) != 0 {
		query.Equals(manager.GetSlaveFieldName(), input.AlertId)
	}
	if len(input.Metric) > 0 {
		query.Equals("metric", input.Metric)
	}
	if len(input.JointId) != 0 {
		query.In("row_id", input.JointId)
	}
	if len(input.AlertState) > 0 {
		query = query.Equals("alert_state", input.AlertState)
	}
	err := db.FetchModelObjects(manager, query, &joints)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects by GetJoinsByMasterId:%s err", input.MonitorResourceId)
	}
	return joints, nil
}

func (obj *SMonitorResourceAlert) UpdateAlertRecordData(ctx context.Context, userCred mcclient.TokenCredential, input *UpdateMonitorResourceAlertInput, match *monitor.EvalMatch) error {
	sendState := input.SendState
	if _, ok := match.Tags[monitor.ALERT_RESOURCE_RECORD_SHIELD_KEY]; ok {
		sendState = monitor.SEND_STATE_SHIELD
	}
	if _, err := db.Update(obj, func() error {
		if input.AlertRecordId != "" {
			obj.AlertRecordId = input.AlertRecordId
		}
		obj.ResType = input.ResType
		obj.AlertState = input.AlertState
		obj.SendState = sendState
		obj.TriggerTime = input.TriggerTime
		obj.Metric = match.Metric
		obj.Data = jsonutils.Marshal(match)
		return nil
	}); err != nil {
		return errors.Wrap(err, "db.Update")
	}
	if err := obj.GetModelManager().GetExtraHook().AfterPostUpdate(ctx, userCred, obj, jsonutils.NewDict(), jsonutils.NewDict()); err != nil {
		log.Warningf("UpdateAlertRecordData after post update hook error: %v", err)
	}
	return nil
}

func (obj *SMonitorResourceAlert) GetData() (*monitor.EvalMatch, error) {
	if obj.Data == nil {
		return nil, errors.Errorf("data is nil")
	}
	match := new(monitor.EvalMatch)
	if err := obj.Data.Unmarshal(match); err != nil {
		return nil, errors.Wrap(err, "unmarshal to monitor.EvalMatch")
	}
	return match, nil
}

func (m *SMonitorResourceAlertManager) GetNowAlertingAlerts(ctx context.Context, userCred mcclient.TokenCredential, input *monitor.AlertRecordListInput) ([]SMonitorResourceAlert, error) {
	ownerId, err := m.FetchOwnerId(ctx, jsonutils.Marshal(&input))
	if err != nil {
		return nil, errors.Wrapf(err, "FetchOwnerId by input: %s", jsonutils.Marshal(input))
	}
	if ownerId == nil {
		ownerId = userCred
	}

	alertsQuery := CommonAlertManager.Query("id").Equals("state", monitor.AlertStateAlerting).IsTrue("enabled").
		IsNull("used_by")
	alertsQuery = CommonAlertManager.FilterByOwner(ctx, alertsQuery, CommonAlertManager, userCred, ownerId, rbacscope.String2Scope(input.Scope))

	alertRess := make([]SMonitorResourceAlert, 0)
	q := m.Query()
	q = q.Equals("alert_state", monitor.AlertStateAlerting)
	q = q.In("alert_id", alertsQuery)
	if err := db.FetchModelObjects(m, q, &alertRess); err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects by GetNowAlertingAlerts err")
	}
	return alertRess, nil
}

func (m *SMonitorResourceAlertManager) FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery {
	input := &monitor.MonitorResourceJointListInput{}
	if err := params.Unmarshal(input); err != nil {
		log.Errorf("FilterByParams unmarshal params error: %v", err)
		return q
	}
	if input.Metric != "" {
		q = q.Equals("metric", input.Metric)
	}
	return q
}

func (m *SMonitorResourceAlertManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *monitor.MonitorResourceJointListInput) (*sqlchemy.SQuery, error) {
	// 如果指定了时间段、top 和 alert_id 参数，执行特殊的 top 查询
	if input.Top != nil {
		// 加上 top 的参数校验
		if input.Top == nil || *input.Top <= 0 {
			return nil, httperrors.NewInputParameterError("top must be specified and greater than 0")
		}
		if input.StartTime.IsZero() || input.EndTime.IsZero() {
			return nil, httperrors.NewInputParameterError("start_time and end_time must be specified")
		}
		if input.StartTime.After(input.EndTime) {
			return nil, httperrors.NewInputParameterError("start_time must be before end_time")
		}
		return m.getTopResourcesByMetricAndAlertCount(ctx, q, userCred, input)
	}

	var err error
	q, err = m.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, input.JointResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SJointResourceBaseManager ListItemFilter err")
	}
	alertSq := AlertManager.Query("id").SubQuery()
	q = q.In("alert_id", alertSq)
	if len(input.AlertState) > 0 {
		q = q.Equals("alert_state", input.AlertState)
	}
	if input.Alerting {
		q = q.Equals("alert_state", monitor.AlertStateAlerting)
		resQ := MonitorResourceManager.Query("res_id")
		resQ, err = MonitorResourceManager.ListItemFilter(ctx, resQ, userCred, monitor.MonitorResourceListInput{})
		if err != nil {
			return q, errors.Wrap(err, "Get monitor in Query err")
		}
		resQ = m.SMonitorScopedResourceManager.FilterByOwner(ctx, resQ, m, userCred, userCred, rbacscope.TRbacScope(input.Scope))
		q = q.Filter(sqlchemy.In(q.Field("monitor_resource_id"), resQ.SubQuery()))
	}
	if len(input.SendState) != 0 {
		q = q.Equals("send_state", input.SendState)
	}
	if len(input.ResType) != 0 {
		q = q.Equals("res_type", input.ResType)
	}
	if len(input.ResName) != 0 {
		resQ := MonitorResourceManager.Query("res_id")
		resQ, err = MonitorResourceManager.ListItemFilter(ctx, resQ, userCred,
			monitor.MonitorResourceListInput{ResName: input.ResName})
		if err != nil {
			return q, errors.Wrap(err, "Get monitor in Query err")
		}
		q = q.Filter(sqlchemy.In(q.Field("monitor_resource_id"), resQ.SubQuery()))
	}
	alertQuery := CommonAlertManager.Query("id")
	alertQuery = m.SMonitorScopedResourceManager.FilterByOwner(ctx, alertQuery, m, userCred, userCred, rbacscope.TRbacScope(input.Scope))
	if len(input.AlertName) != 0 {
		CommonAlertManager.FieldListFilter(alertQuery, monitor.CommonAlertListInput{Name: input.AlertName})
		q = q.Filter(sqlchemy.In(q.Field(m.GetSlaveFieldName()), alertQuery.SubQuery()))
	}
	if len(input.Level) != 0 {
		CommonAlertManager.FieldListFilter(alertQuery, monitor.CommonAlertListInput{Level: input.Level})
		q = q.Filter(sqlchemy.In(q.Field(m.GetSlaveFieldName()), alertQuery.SubQuery()))
	}
	if len(input.MonitorResourceId) != 0 {
		q = q.Filter(sqlchemy.Equals(q.Field("monitor_resource_id"), input.MonitorResourceId))
	}
	if !input.AllState {
		q = q.Filter(sqlchemy.In(q.Field(m.GetSlaveFieldName()), alertQuery.SubQuery()))
		q = q.Filter(sqlchemy.In(q.Field("alert_record_id"), AlertRecordManager.Query("id").SubQuery()))
	}
	return q, nil
}

func (m *SMonitorResourceAlertManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()

	if query.Contains("ip") {
		ip, _ := query.GetString("ip")
		ipF := func(obj jsonutils.JSONObject) (bool, error) {
			match := new(monitor.EvalMatch)
			if err := obj.Unmarshal(match, "data"); err != nil {
				log.Warningf("unmarshal data of object %s: %v", obj, err)
				return false, nil
			}
			if tagIp, ok := match.Tags["ip"]; ok {
				if strings.Contains(tagIp, ip) {
					return true, nil
				}
			}
			return false, nil
		}
		filters.Append(ipF)
	}

	return filters, nil
}

func (m *SMonitorResourceAlertManager) GetMonitorResourceAlert(resId, alertId, metric string) (*SMonitorResourceAlert, error) {
	joint, err := db.FetchJointByIds(m, resId, alertId, jsonutils.Marshal(&monitor.MonitorResourceJointListInput{
		Metric: metric,
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "FetchJointByIds with master id %s and slave id %s and metric %s", resId, alertId, metric)
	}
	return joint.(*SMonitorResourceAlert), nil
}

// getTopResourcesByMetricAndAlertCount 查询指定时间段内，某个监控策略下各监控指标报警资源最多的 top N 资源
func (m *SMonitorResourceAlertManager) getTopResourcesByMetricAndAlertCount(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input *monitor.MonitorResourceJointListInput,
) (*sqlchemy.SQuery, error) {
	// 验证时间段和 top 参数
	startTime, endTime, top, err := validateTopQueryInput(input.TopQueryInput)
	if err != nil {
		return nil, err
	}

	// 查询指定时间段内的 AlertRecord，获取 alert_id、res_ids 和 alert_rule（用于解析 metric）
	recordQuery := AlertRecordManager.Query("id", "alert_id", "res_ids", "res_type", "alert_rule")
	if len(input.AlertId) > 0 {
		recordQuery = recordQuery.Equals("alert_id", input.AlertId)
	}
	recordQuery = recordQuery.GE("created_at", startTime).LE("created_at", endTime)
	recordQuery = recordQuery.IsNotEmpty("res_ids")

	// 如果指定了 ResType，添加过滤条件
	if len(input.ResType) > 0 {
		recordQuery = recordQuery.Equals("res_type", input.ResType)
	}

	// 执行查询获取所有记录
	type RecordRow struct {
		Id        string
		AlertId   string
		ResIds    string
		ResType   string
		AlertRule jsonutils.JSONObject
	}
	rows := make([]RecordRow, 0)
	err = recordQuery.All(&rows)
	if err != nil {
		return nil, errors.Wrap(err, "query alert records")
	}

	// 按照 resId、alert_id 和 metric 三个维度统计告警数量
	// resourceAlertMetricCount[resId][alertId][metric] = count
	type ResourceAlertMetricKey struct {
		ResId   string
		AlertId string
		Metric  string
	}
	resourceAlertMetricCount := make(map[ResourceAlertMetricKey]int)

	for _, row := range rows {
		if len(row.ResIds) == 0 {
			continue
		}
		// 从 AlertRule 中解析 metric
		var alertRules []*monitor.AlertRecordRule
		if row.AlertRule != nil {
			if err := row.AlertRule.Unmarshal(&alertRules); err != nil {
				log.Warningf("unmarshal alert_rule error: %v", err)
				continue
			}
		}
		if len(alertRules) == 0 {
			continue
		}

		// 解析 res_ids（逗号分隔）
		resIds := strings.Split(row.ResIds, ",")
		for _, resId := range resIds {
			resId = strings.TrimSpace(resId)
			if len(resId) == 0 {
				continue
			}
			// 如果指定了 ResType，需要匹配 res_type
			if len(input.ResType) > 0 && row.ResType != input.ResType {
				continue
			}
			// 对于每个 metric，统计 (resId, alertId, metric) 组合的数量
			for _, rule := range alertRules {
				if len(rule.Metric) == 0 {
					continue
				}
				key := ResourceAlertMetricKey{
					ResId:   resId,
					AlertId: row.AlertId,
					Metric:  rule.Metric,
				}
				resourceAlertMetricCount[key]++
			}
		}
	}

	// 转换为切片并按报警数量排序
	type ResourceAlertMetricCount struct {
		ResId   string
		AlertId string
		Metric  string
		Count   int
	}
	counts := make([]ResourceAlertMetricCount, 0, len(resourceAlertMetricCount))
	for key, count := range resourceAlertMetricCount {
		counts = append(counts, ResourceAlertMetricCount{
			ResId:   key.ResId,
			AlertId: key.AlertId,
			Metric:  key.Metric,
			Count:   count,
		})
	}

	// 按报警数量降序排序
	for i := 0; i < len(counts)-1; i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[i].Count < counts[j].Count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	// 获取全局 top N 的 (resId, alertId, metric) 组合
	topN := min(top, len(counts))
	topCombinations := make([]ResourceAlertMetricCount, 0, topN)
	for i := 0; i < topN; i++ {
		topCombinations = append(topCombinations, counts[i])
	}
	log.Infof("top %d combinations: %v", top, topCombinations)

	if len(topCombinations) == 0 {
		// 如果没有找到任何记录，返回空查询
		return q.FilterByFalse(), nil
	}

	// 构建查询条件：匹配 top N 的 (resId, alertId, metric) 组合
	q = m.Query()
	conditions := make([]sqlchemy.ICondition, 0, len(topCombinations))
	for _, combo := range topCombinations {
		cond := sqlchemy.AND(
			sqlchemy.Equals(q.Field("monitor_resource_id"), combo.ResId),
			sqlchemy.Equals(q.Field("alert_id"), combo.AlertId),
			sqlchemy.Equals(q.Field("metric"), combo.Metric),
		)
		conditions = append(conditions, cond)
	}
	q = q.Filter(sqlchemy.OR(conditions...))

	// 应用其他过滤条件
	if len(input.AlertState) > 0 {
		q = q.Equals("alert_state", input.AlertState)
	}
	if input.Alerting {
		q = q.Equals("alert_state", monitor.AlertStateAlerting)
		resQ := MonitorResourceManager.Query("res_id")
		resQ, err = MonitorResourceManager.ListItemFilter(ctx, resQ, userCred, monitor.MonitorResourceListInput{})
		if err != nil {
			return q, errors.Wrap(err, "Get monitor in Query err")
		}
		resQ = m.SMonitorScopedResourceManager.FilterByOwner(ctx, resQ, m, userCred, userCred, rbacscope.TRbacScope(input.Scope))
		q = q.Filter(sqlchemy.In(q.Field("monitor_resource_id"), resQ.SubQuery()))
	}
	if len(input.SendState) != 0 {
		q = q.Equals("send_state", input.SendState)
	}
	if len(input.ResType) != 0 {
		q = q.Equals("res_type", input.ResType)
	}
	if len(input.Metric) != 0 {
		q = q.Equals("metric", input.Metric)
	}

	return q, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type SAlertRecordCount struct {
	Count   int
	ResIds  string
	AlertId string
}

func (man *SMonitorResourceAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.MonitorResourceJointDetails {
	input := &monitor.MonitorResourceJointListInput{}
	query.Unmarshal(input)
	rows := make([]monitor.MonitorResourceJointDetails, len(objs))
	alertRecordIds := make([]string, len(objs))
	alertIds := make([]string, len(objs))
	resourceIds := make([]string, len(objs))
	for i := range rows {
		obj := objs[i].(*SMonitorResourceAlert)
		alertRecordIds[i] = obj.AlertRecordId
		alertIds[i] = obj.AlertId
		resourceIds[i] = obj.MonitorResourceId
	}
	records := map[string]SAlertRecord{}
	err := db.FetchModelObjectsByIds(AlertRecordManager, "id", alertRecordIds, &records)
	if err != nil {
		log.Errorf("fetch alert records error: %v", err)
		return rows
	}
	alerts := map[string]SCommonAlert{}
	err = db.FetchModelObjectsByIds(CommonAlertManager, "id", alertIds, &alerts)
	if err != nil {
		log.Errorf("fetch alerts error: %v", err)
		return rows
	}
	resources := map[string]SMonitorResource{}
	err = db.FetchModelObjectsByIds(MonitorResourceManager, "res_id", resourceIds, &resources)
	if err != nil {
		log.Errorf("fetch monitor resources error: %v", err)
		return rows
	}
	shields := make([]SAlertRecordShield, 0)
	err = AlertRecordShieldManager.Query().GE("end_time", time.Now()).In("res_id", resourceIds).In("alert_id", alertIds).All(&shields)
	if err != nil {
		log.Errorf("fetch alert record shields error: %v", err)
		return rows
	}
	shieldsMap := map[string]bool{}
	for _, shield := range shields {
		shieldsMap[fmt.Sprintf("%s-%s", shield.ResId, shield.AlertId)] = true
	}
	recordCountMap := map[string][]SAlertRecordCount{}
	if !input.StartTime.IsZero() && !input.EndTime.IsZero() {
		sq := AlertRecordManager.Query().GE("created_at", input.StartTime).LE("created_at", input.EndTime).In("alert_id", alertIds).SubQuery()
		q := sq.Query(
			sqlchemy.COUNT("count", sq.Field("id")),
			sq.Field("alert_id"),
			sq.Field("res_ids"),
		).GroupBy(sq.Field("alert_id"), sq.Field("res_ids"))

		recordCount := []SAlertRecordCount{}
		err = q.All(&recordCount)
		if err != nil {
			log.Errorf("fetch alert records error: %v", err)
			return rows
		}
		for i := range recordCount {
			_, ok := recordCountMap[recordCount[i].AlertId]
			if !ok {
				recordCountMap[recordCount[i].AlertId] = make([]SAlertRecordCount, 0)
			}
			recordCountMap[recordCount[i].AlertId] = append(recordCountMap[recordCount[i].AlertId], recordCount[i])
		}
	}
	for i := range rows {
		rows[i] = monitor.MonitorResourceJointDetails{}
		obj := objs[i].(*SMonitorResourceAlert)
		rows[i].ResId = obj.MonitorResourceId
		rows[i].ResType = obj.ResType
		if record, ok := records[obj.AlertRecordId]; ok {
			rows[i].SendState = record.SendState
			rows[i].State = record.State
		}
		if alert, ok := alerts[obj.AlertId]; ok {
			rows[i].AlertName = alert.Name
			rows[i].Level = alert.Level
			silentPeriod, _ := alert.GetSilentPeriod()
			rule, _ := alert.GetAlertRules(silentPeriod)
			rows[i].AlertRule = jsonutils.Marshal(rule)
		}
		if res, ok := resources[obj.MonitorResourceId]; ok {
			rows[i].ResName = res.Name
			rows[i].ResType = res.ResType
			rows[i].MonitorResourceObjectId = res.GetId()
		}
		if _, ok := shieldsMap[fmt.Sprintf("%s-%s", obj.MonitorResourceId, obj.AlertId)]; ok {
			rows[i].IsSetShield = true
		}
		if recordCount, ok := recordCountMap[obj.AlertId]; ok {
			rows[i].AlertCount = 0
			for _, record := range recordCount {
				if strings.Contains(record.ResIds, obj.MonitorResourceId) {
					rows[i].AlertCount += record.Count
				}
			}
		}
	}
	return rows
}

func (manager *SMonitorResourceAlertManager) ResourceScope() rbacscope.TRbacScope {
	return manager.SScopedResourceBaseManager.ResourceScope()
}

func (manager *SMonitorResourceAlertManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (m *SMonitorResourceAlertManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return q
}

func (m *SMonitorResourceAlertManager) GetResourceAlert(alertId string, resourceId string, metric string) (*SMonitorResourceAlert, error) {
	q := m.Query().Equals("alert_id", alertId).Equals("monitor_resource_id", resourceId).Equals("metric", metric)
	obj := new(SMonitorResourceAlert)
	if err := q.First(obj); err != nil {
		return nil, errors.Wrapf(err, "query resource alert by alert_id: %q, resource_id: %q, metric: %q", alertId, resourceId, metric)
	}
	return obj, nil
}

// GetAlertRecords 通过 MonitorResourceAlert 查询历史的 AlertRecord
// 查询条件：相同的 AlertId 且 ResIds 包含该 MonitorResourceId 的所有记录
func (obj *SMonitorResourceAlert) GetAlertRecords() ([]SAlertRecord, error) {
	records := make([]SAlertRecord, 0)
	q := AlertRecordManager.Query()
	q = q.Equals("alert_id", obj.AlertId)
	// ResIds 字段存储的是逗号分隔的资源ID列表，使用 ContainsAny 查询
	q = q.Filter(sqlchemy.ContainsAny(q.Field("res_ids"), []string{obj.MonitorResourceId}))
	q = q.Desc("created_at") // 按创建时间倒序，最新的在前
	err := db.FetchModelObjects(AlertRecordManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "get historical alert records by alert_id: %q, monitor_resource_id: %q", obj.AlertId, obj.MonitorResourceId)
	}
	return records, nil
}
