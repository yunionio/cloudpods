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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	MonitorResourceManager *SMonitorResourceManager
)

func init() {
	MonitorResourceManager = &SMonitorResourceManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SMonitorResource{},
			"monitor_resource_tbl",
			"monitorresource",
			"monitorresources",
		),
	}
	MonitorResourceManager.SetVirtualObject(MonitorResourceManager)
	RegistryResourceSync(NewGuestResourceSync())
	RegistryResourceSync(NewHostResourceSync())
}

type SMonitorResourceManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SMonitorResource struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	AlertState string `width:"36" charset:"ascii" list:"user" default:"init" update:"user" json:"alert_state"`
	ResId      string `width:"256" charset:"ascii"  index:"true" list:"user" update:"user" json:"res_id"`
	ResType    string `width:"36" charset:"ascii" list:"user" update:"user" json:"res_type"`
}

func (manager *SMonitorResourceManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	for _, sync := range GetResourceSyncMap() {
		err := sync.SyncResources(ctx, userCred, jsonutils.NewDict())
		if err != nil {
			log.Errorf("resType:%s SyncResources err:%v", sync.SyncType(), err)
		}
	}
	err := CommonAlertManager.Run(ctx)
	if err != nil {
		log.Errorf("CommonAlertManager UpdateMonitorResourceJoint err:%v", err)
		return
	}
	log.Infoln("====SMonitorResourceManager SyncResources End====")
}

func (manager *SMonitorResourceManager) GetMonitorResources(input monitor.MonitorResourceListInput) ([]SMonitorResource, error) {
	monitorResources := make([]SMonitorResource, 0)
	query := manager.Query()
	if input.OnlyResId {
		query = query.AppendField(query.Field("id"), query.Field("res_id"))
	}
	query = manager.FieldListFilter(query, input)
	err := db.FetchModelObjects(manager, query, &monitorResources)
	if err != nil {
		return nil, errors.Wrap(err, "SMonitorResourceManager FetchModelObjects err")
	}
	return monitorResources, nil
}

func (manager *SMonitorResourceManager) GetMonitorResourceById(id string) (*SMonitorResource, error) {
	iModel, err := db.FetchById(manager, id)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("GetMonitorResourceById:%s err", id))
	}
	return iModel.(*SMonitorResource), nil
}

func (manager *SMonitorResourceManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.MonitorResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	q = manager.FieldListFilter(q, query)

	return q, nil
}

func (manager *SMonitorResourceManager) FieldListFilter(q *sqlchemy.SQuery, query monitor.MonitorResourceListInput) *sqlchemy.SQuery {
	if len(query.ResType) != 0 {
		q.Equals("res_type", query.ResType)
	}
	if len(query.ResId) != 0 {
		q.In("res_id", query.ResId)
	}
	return q
}

func (man *SMonitorResourceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.SuggestSysAlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SMonitorResourceManager) HasName() bool {
	return false
}

func (man *SMonitorResourceManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.MonitorResourceCreateInput) (monitor.MonitorResourceCreateInput, error) {
	//rule 查询到资源信息后没有将资源id，进行转换
	if len(data.ResId) == 0 {
		return data, httperrors.NewInputParameterError("not found res_id %q", data.ResId)
	}
	if len(data.ResType) == 0 {
		return data, httperrors.NewInputParameterError("not found res_type %q", data.ResType)
	}
	return data, nil
}

func (self *SMonitorResource) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return nil
}

func (man *SMonitorResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.MonitorResourceDetails {
	rows := make([]monitor.MonitorResourceDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.MonitorResourceDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*SMonitorResource).getMoreDetails(rows[i])
	}
	return rows
}

func (self *SMonitorResource) AttachAlert(ctx context.Context, userCred mcclient.TokenCredential, alertId string) error {
	iModel, _ := db.NewModelObject(MonitorResourceAlertManager)
	input := monitor.MonitorResourceJointCreateInput{
		MonitorResourceId: self.ResId,
		AlertId:           alertId,
		AlertState:        monitor.MONITOR_RESOURCE_ALERT_STATUS_ATTACH,
	}
	data := input.JSON(&input)
	err := data.Unmarshal(iModel)
	if err != nil {
		return errors.Wrap(err, "MonitorResourceJointCreateInput unmarshal to joint err")
	}
	if err := MonitorResourceAlertManager.TableSpec().Insert(ctx, iModel); err != nil {
		return errors.Wrap(err, "insert MonitorResourceJoint model err")
	}
	return nil
}

func (self *SMonitorResource) UpdateAlertState() error {
	joints, _ := MonitorResourceAlertManager.GetJoinsByListInput(monitor.MonitorResourceJointListInput{MonitorResourceId: self.ResId})
	jointState := monitor.MONITOR_RESOURCE_ALERT_STATUS_ATTACH
	if len(joints) == 0 {
		jointState = monitor.MONITOR_RESOURCE_ALERT_STATUS_INIT
	}
	for _, joint := range joints {
		if joint.AlertState == monitor.MONITOR_RESOURCE_ALERT_STATUS_ALERTING && time.Now().Sub(joint.
			TriggerTime) < time.Minute*30 {
			jointState = monitor.MONITOR_RESOURCE_ALERT_STATUS_ALERTING
		}
	}
	_, err := db.Update(self, func() error {
		self.AlertState = jointState
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SMonitorResource:%s UpdateAlertState err", self.Name)
	}
	return nil
}

func (self *SMonitorResource) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.DetachJoint(ctx, userCred)
	if err != nil {
		return err
	}
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SMonitorResource) DetachJoint(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := MonitorResourceAlertManager.DetachJoint(ctx, userCred,
		monitor.MonitorResourceJointListInput{MonitorResourceId: self.ResId})
	if err != nil {
		return errors.Wrap(err, "SMonitorResource DetachJoint err")
	}
	return nil
}

func (self *SMonitorResource) getMoreDetails(out monitor.MonitorResourceDetails) monitor.MonitorResourceDetails {
	joints, err := MonitorResourceAlertManager.GetJoinsByListInput(monitor.
		MonitorResourceJointListInput{MonitorResourceId: self.ResId})
	if err != nil {
		log.Errorf("getMoreDetails err:%v", err)
	}
	out.AttachAlertCount = int64(len(joints))
	return out
}

func (manager *SMonitorResourceManager) AllowGetPropertyAlert(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

type AlertStatusCount struct {
	CountId    int64
	AlertState string
}

func (manager *SMonitorResourceManager) GetPropertyAlert(ctx context.Context, userCred mcclient.TokenCredential,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	scope, _ := data.GetString("scope")
	if len(scope) == 0 {
		scope = "system"
	}
	result := jsonutils.NewDict()
	for resType, _ := range GetResourceSyncMap() {
		query := manager.Query("alert_state")
		owner, _ := manager.FetchOwnerId(ctx, data)
		if owner == nil {
			owner = userCred
		}
		manager.FilterByOwner(query, owner, rbacutils.TRbacScope(scope))
		query.AppendField(sqlchemy.COUNT("count_id", query.Field("id")))
		input := monitor.MonitorResourceListInput{ResType: resType}
		query = manager.FieldListFilter(query, input)
		query.GroupBy(query.Field("alert_state"))
		log.Errorf("query:%s", query.String())
		rows, err := query.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "getMonitorResourceAlert query err")
		}
		total := int64(0)
		resTypeDict := jsonutils.NewDict()
		for rows.Next() {
			row := new(AlertStatusCount)
			err := query.Row2Struct(rows, row)
			if err != nil {
				return nil, errors.Wrap(err, "MonitorResource Row2Struct err")
			}
			resTypeDict.Add(jsonutils.NewInt(row.CountId), row.AlertState)
			total += row.CountId
		}
		resTypeDict.Add(jsonutils.NewInt(total), "total")
		result.Add(resTypeDict, resType)
	}
	return result, nil
}

func (manager *SMonitorResourceManager) UpdateMonitorResourceAttachJoint(ctx context.Context,
	userCred mcclient.TokenCredential, alertRecord *SAlertRecord) error {
	if !utils.IsInStringArray(alertRecord.ResType, []string{monitor.METRIC_RES_TYPE_HOST,
		monitor.METRIC_RES_TYPE_GUEST, monitor.METRIC_RES_TYPE_AGENT}) {
		return nil
	}
	resType := alertRecord.ResType
	if resType == monitor.METRIC_RES_TYPE_AGENT {
		resType = monitor.METRIC_RES_TYPE_GUEST
	}
	matches, _ := alertRecord.GetEvalData()
	errs := make([]error, 0)
	matchResourceIds := make([]string, 0)
	for _, matche := range matches {
		resId := matche.Tags[monitor.MEASUREMENT_TAG_ID[alertRecord.ResType]]
		if len(resId) == 0 {
			continue
		}
		matchResourceIds = append(matchResourceIds, resId)
		monitorResources, err := manager.GetMonitorResources(monitor.MonitorResourceListInput{ResType: resType, ResId: []string{resId}})
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "SMonitorResourceManager GetMonitorResources by resId:%s err", resId))
			continue
		}
		for _, res := range monitorResources {
			err := res.UpdateAttachJoint(alertRecord, matche)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	resourceAlerts, err := MonitorResourceAlertManager.GetJoinsByListInput(monitor.MonitorResourceJointListInput{AlertId: alertRecord.AlertId})
	if err != nil {
		return errors.Wrapf(err, "get monitor_resource_joint by alertId:%s err", alertRecord.AlertId)
	}
	deleteJointIds := make([]int64, 0)
	for _, joint := range resourceAlerts {
		if utils.IsInStringArray(joint.MonitorResourceId, matchResourceIds) {
			continue
		}
		deleteJointIds = append(deleteJointIds, joint.RowId)
	}
	if len(deleteJointIds) != 0 {
		err = MonitorResourceAlertManager.DetachJoint(ctx, userCred, monitor.MonitorResourceJointListInput{JointId: deleteJointIds})
		if err != nil {
			return errors.Wrapf(err, "DetachJoint by alertId:%s err", alertRecord.AlertId)
		}
	}
	return errors.NewAggregate(errs)
}

func (self *SMonitorResource) UpdateAttachJoint(alertRecord *SAlertRecord, match monitor.EvalMatch) error {
	joints, err := MonitorResourceAlertManager.GetJoinsByListInput(monitor.MonitorResourceJointListInput{MonitorResourceId: self.
		ResId, AlertId: alertRecord.AlertId})
	if err != nil {
		return errors.Wrapf(err, "SMonitorResource:%s UpdateAttachJoint err", self.Name)
	}
	errs := make([]error, 0)
	for _, joint := range joints {
		err := joint.UpdateAlertRecordData(alertRecord, &match)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "joint %s:%s %s:%s UpdateAlertRecordData err",
				MonitorResourceAlertManager.GetMasterFieldName(), self.ResId,
				MonitorResourceAlertManager.GetSlaveFieldName(), alertRecord.AlertId))
		}
	}
	self.UpdateAlertState()
	return errors.NewAggregate(errs)

}
