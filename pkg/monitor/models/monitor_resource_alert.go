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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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

func (obj *SMonitorResourceAlert) UpdateAlertRecordData(input *UpdateMonitorResourceAlertInput, match *monitor.EvalMatch) error {
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

func (m *SMonitorResourceAlertManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *monitor.MonitorResourceJointListInput) (*sqlchemy.SQuery, error) {
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

func (man *SMonitorResourceAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.MonitorResourceJointDetails {
	rows := make([]monitor.MonitorResourceJointDetails, len(objs))
	for i := range rows {
		rows[i] = monitor.MonitorResourceJointDetails{}
		rows[i] = objs[i].(*SMonitorResourceAlert).getMoreDetails(rows[i])
	}
	return rows
}

func (obj *SMonitorResourceAlert) getMoreDetails(detail monitor.MonitorResourceJointDetails) monitor.MonitorResourceJointDetails {
	detail.ResType = obj.ResType
	detail.ResId = obj.MonitorResourceId
	resources, err := MonitorResourceManager.GetMonitorResources(monitor.MonitorResourceListInput{ResId: []string{obj.
		MonitorResourceId}})
	if err != nil {
		log.Errorf("getMonitorResource:%s err:%v", obj.MonitorResourceId, err)
		return detail
	}
	if len(resources) == 0 {
		return detail
	}
	if detail.ResType == "" {
		detail.ResType = resources[0].ResType
	}
	detail.ResName = resources[0].Name
	//detail.ResId = resources[0].ResId

	if len(obj.AlertRecordId) != 0 {
		record, err := AlertRecordManager.GetAlertRecord(obj.AlertRecordId)
		if err != nil {
			log.Errorf("get alertRecord:%s err:%v", obj.AlertRecordId, err)
			return detail
		}
		//detail.AlertRule = record.AlertRule
		detail.SendState = record.SendState
		detail.State = record.State
	}
	alert, err := CommonAlertManager.GetAlert(obj.AlertId)
	if err != nil {
		log.Errorf("SMonitorResourceAlert get alert by id :%s err:%v", obj.AlertId, err)
		return detail
	}
	detail.AlertName = alert.Name
	detail.Level = alert.Level
	silentPeriod, _ := alert.GetSilentPeriod()
	rule, _ := alert.GetAlertRules(silentPeriod)
	detail.AlertRule = jsonutils.Marshal(rule)

	now := time.Now()
	shields, err := AlertRecordShieldManager.GetRecordShields(monitor.AlertRecordShieldListInput{ResId: obj.MonitorResourceId,
		AlertId: obj.AlertId, EndTime: &now})
	if err != nil {
		log.Errorf("SMonitorResourceAlert get GetRecordShields by resId: %s, alertId: %s, err: %v", obj.MonitorResourceId, obj.AlertId, err)
		return detail
	}
	if len(shields) != 0 {
		detail.IsSetShield = true
	}
	return detail
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
