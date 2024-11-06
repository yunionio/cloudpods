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
	"reflect"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	MonitorResourceManager *SMonitorResourceManager
)

type IMonitorResourceCache interface {
	Get(resId string) (jsonutils.JSONObject, bool)
}

type sMonitorResourceCache struct {
	length int
	sync.Map
}

func (c *sMonitorResourceCache) set(resId string, obj jsonutils.JSONObject) {
	c.Store(resId, obj)
	c.length++
}

func (c *sMonitorResourceCache) remove(resId string) {
	c.Delete(resId)
	c.length--
}

func (c *sMonitorResourceCache) Get(resId string) (jsonutils.JSONObject, bool) {
	obj, ok := c.Load(resId)
	if !ok {
		return nil, false
	}
	return obj.(jsonutils.JSONObject), true
}

func init() {
	MonitorResourceManager = &SMonitorResourceManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SMonitorResource{},
			"monitor_resource_tbl",
			"monitorresource",
			"monitorresources",
		),
		monitorResModelSets: NewModelSets(),
	}
	MonitorResourceManager.SetVirtualObject(MonitorResourceManager)
	RegistryResourceSync(NewGuestResourceSync())
	RegistryResourceSync(NewHostResourceSync())
	RegistryResourceSync(NewRdsResourceSync())
	RegistryResourceSync(NewRedisResourceSync())
	RegistryResourceSync(NewOssResourceSync())
	RegistryResourceSync(NewAccountResourceSync())
	RegistryResourceSync(NewStorageResourceSync())
}

func (manager *SMonitorResourceManager) GetModelSets() *MonitorResModelSets {
	return manager.monitorResModelSets
}

type SMonitorResourceManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager

	monitorResModelSets *MonitorResModelSets
	apih                *apihelper.APIHelper
}

func (manager *SMonitorResourceManager) SetAPIHelper(h *apihelper.APIHelper) {
	if manager.apih != nil {
		panic("MonitorResourceManager's apihelper already set")
	}
	manager.apih = h
}

type SMonitorResource struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	AlertState string `width:"36" charset:"ascii" list:"user" default:"init" update:"user" json:"alert_state"`
	ResId      string `width:"256" charset:"ascii"  index:"true" list:"user" update:"user" json:"res_id"`
	ResType    string `width:"36" charset:"ascii" list:"user" update:"user" json:"res_type"`
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

type SdeleteRes struct {
	resType string
	notIn   []string
	in      []string
}

func (manager *SMonitorResourceManager) DeleteMonitorResources(ctx context.Context, userCred mcclient.TokenCredential, input SdeleteRes) error {
	monitorResources := make([]SMonitorResource, 0)
	errs := make([]error, 0)
	query := manager.Query()
	if len(input.notIn) != 0 {
		query.NotIn("res_id", input.notIn)
	}
	if len(input.in) != 0 {
		query.In("res_id", input.in)
	}
	if len(input.resType) != 0 {
		query.Equals("res_type", input.resType)
	}
	err := db.FetchModelObjects(manager, query, &monitorResources)
	if err != nil {
		return errors.Wrap(err, "SMonitorResourceManager FetchModelObjects when DeleteMonitorResources err")
	}
	for _, res := range monitorResources {
		err := (&res).RealDelete(ctx, userCred)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "delete monitorResource:%s err", res.GetId()))
		}
	}
	if len(errs) != 0 {
		return errors.NewAggregate(errs)
	}
	return nil
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
	if len(query.ResName) != 0 {
		q.Contains("name", query.ResName)
	}
	return q
}

func (man *SMonitorResourceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.MonitorResourceListInput,
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

func (self *SMonitorResource) AttachAlert(ctx context.Context, userCred mcclient.TokenCredential, alertId string, metric string, match monitor.EvalMatch) (*SMonitorResourceAlert, error) {
	iModel, _ := db.NewModelObject(MonitorResourceAlertManager)
	input := monitor.MonitorResourceJointCreateInput{
		MonitorResourceId: self.ResId,
		AlertId:           alertId,
		AlertState:        monitor.MONITOR_RESOURCE_ALERT_STATUS_ATTACH,
		Metric:            metric,
		Data:              match,
	}
	data := input.JSON(&input)
	err := data.Unmarshal(iModel)
	if err != nil {
		return nil, errors.Wrap(err, "MonitorResourceJointCreateInput unmarshal to joint err")
	}
	if err := MonitorResourceAlertManager.TableSpec().Insert(ctx, iModel); err != nil {
		return nil, errors.Wrap(err, "insert MonitorResourceJoint model err")
	}
	return iModel.(*SMonitorResourceAlert), nil
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
	_, object := MonitorResourceManager.GetResourceObj(self.ResId)
	if object != nil {
		object.Unmarshal(&out)
	}
	out.AttachAlertCount = int64(len(joints))
	return out
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
		query = manager.FilterByOwner(ctx, query, manager, userCred, owner, rbacscope.TRbacScope(scope))
		query = query.AppendField(sqlchemy.COUNT("count_id", query.Field("id")))
		input := monitor.MonitorResourceListInput{ResType: resType}
		query = manager.FieldListFilter(query, input)
		query.GroupBy(query.Field("alert_state"))
		log.Errorf("query:%s", query.String())
		rows, err := query.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "getMonitorResourceAlert query err")
		}
		defer rows.Close()
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

func (manager *SMonitorResourceManager) UpdateMonitorResourceAttachJointByRecord(ctx context.Context, userCred mcclient.TokenCredential, record *SAlertRecord) error {
	matches, _ := record.GetEvalData()
	input := &UpdateMonitorResourceAlertInput{
		AlertId:       record.AlertId,
		Matches:       matches,
		ResType:       record.ResType,
		AlertState:    record.State,
		SendState:     record.SendState,
		TriggerTime:   record.CreatedAt,
		AlertRecordId: record.GetId(),
	}
	if err := manager.UpdateMonitorResourceAttachJoint(ctx, userCred, input); err != nil {
		return errors.Wrap(err, "UpdateMonitorResourceAttachJoint")
	}
	return nil
}

type UpdateMonitorResourceAlertInput struct {
	AlertId       string
	Matches       []monitor.EvalMatch
	ResType       string
	AlertState    string
	SendState     string
	TriggerTime   time.Time
	AlertRecordId string
}

func (manager *SMonitorResourceManager) UpdateMonitorResourceAttachJoint(ctx context.Context, userCred mcclient.TokenCredential, input *UpdateMonitorResourceAlertInput) error {
	resType := input.ResType
	if resType == monitor.METRIC_RES_TYPE_AGENT {
		resType = monitor.METRIC_RES_TYPE_GUEST
	}
	matches := input.Matches
	errs := make([]error, 0)
	matchResourceIds := make([]string, 0)
	for _, match := range matches {
		resId := monitor.GetMeasurementResourceId(match.Tags, input.ResType)
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
			err := res.UpdateAttachJoint(ctx, userCred, input, match)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "UpdateAttachJoint"))
			}
		}
	}
	resourceAlerts, err := MonitorResourceAlertManager.GetJoinsByListInput(monitor.MonitorResourceJointListInput{
		AlertId:    input.AlertId,
		AlertState: input.AlertState,
	})
	if err != nil {
		return errors.Wrapf(err, "get monitor_resource_joint by alertId: %s", input.AlertId)
	}
	deleteJointIds := make([]int64, 0)
	for _, joint := range resourceAlerts {
		metricName := joint.Metric
		isMetricFound := false
		for _, match := range matches {
			if match.Metric == metricName {
				isMetricFound = true
				break
			}
		}
		if utils.IsInStringArray(joint.MonitorResourceId, matchResourceIds) && isMetricFound {
			continue
		}
		deleteJointIds = append(deleteJointIds, joint.RowId)
	}
	if len(deleteJointIds) != 0 {
		err = MonitorResourceAlertManager.DetachJoint(ctx, userCred, monitor.MonitorResourceJointListInput{JointId: deleteJointIds})
		if err != nil {
			return errors.Wrapf(err, "DetachJoint by alertId:%s err", input.AlertId)
		}
	}
	return errors.NewAggregate(errs)
}

func (self *SMonitorResource) UpdateAttachJoint(ctx context.Context, userCred mcclient.TokenCredential, input *UpdateMonitorResourceAlertInput, match monitor.EvalMatch) error {
	joints, err := MonitorResourceAlertManager.GetJoinsByListInput(
		monitor.MonitorResourceJointListInput{
			MonitorResourceId: self.ResId,
			AlertId:           input.AlertId,
			Metric:            match.Metric,
		})
	if err != nil {
		return errors.Wrapf(err, "SMonitorResource: %s(%s) get joints by monitorResourceId %q , metric %q and alertId %q", self.Name, self.Id, self.ResId, match.Metric, input.AlertId)
	}
	errs := make([]error, 0)
	updateJoints := make([]SMonitorResourceAlert, 0)
	for _, joint := range joints {
		if joint.Metric == match.Metric {
			tmpJoint := joint
			updateJoints = append(updateJoints, tmpJoint)
		}
	}
	// 报警时发现没有进行关联，增加attach
	if len(updateJoints) == 0 {
		newJoint, err := self.AttachAlert(ctx, userCred, input.AlertId, match.Metric, match)
		if err != nil {
			log.Errorf("attach alert error: %s", err)
		}
		log.Infof("Attach Alert joint: %#v, match: %s", newJoint, jsonutils.Marshal(match))
		if err := newJoint.UpdateAlertRecordData(input, &match); err != nil {
			errs = append(errs, errors.Wrapf(err, "new joint %s:%s %s:%s UpdateAlertRecordData err",
				MonitorResourceAlertManager.GetMasterFieldName(), self.ResId,
				MonitorResourceAlertManager.GetSlaveFieldName(), input.AlertId))
		}
	} else {
		for _, joint := range updateJoints {
			err := joint.UpdateAlertRecordData(input, &match)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "joint %s:%s %s:%s UpdateAlertRecordData err",
					MonitorResourceAlertManager.GetMasterFieldName(), self.ResId,
					MonitorResourceAlertManager.GetSlaveFieldName(), input.AlertId))
			}
		}
	}
	if err := self.UpdateAlertState(); err != nil {
		errs = append(errs, errors.Wrapf(err, "UpdateAlertState"))
	}
	return errors.NewAggregate(errs)
}

func (manager *SMonitorResourceManager) GetResourceObj(id string) (bool, jsonutils.JSONObject) {
	for _, set := range manager.GetModelSets().ModelSetList() {
		setRv := reflect.ValueOf(set)
		mRv := setRv.MapIndex(reflect.ValueOf(id))
		if mRv.IsValid() {
			return true, jsonutils.Marshal(mRv.Interface())
		}
	}
	return false, nil
}

func (manager *SMonitorResourceManager) GetResourceObjByResType(typ string) (bool, []jsonutils.JSONObject) {
	manager.GetModelSets()
	for _, set := range manager.GetModelSets().ModelSetList() {
		if _, ok := set.(IMonitorResModelSet); !ok {
			continue
		}
		if set.(IMonitorResModelSet).GetResType() != typ {
			continue
		}
		setRv := reflect.ValueOf(set)
		objects := make([]jsonutils.JSONObject, 0)
		for _, kRv := range setRv.MapKeys() {
			mRv := setRv.MapIndex(kRv)
			objects = append(objects, jsonutils.Marshal(mRv.Interface()))
		}
		return true, objects
	}
	return false, nil
}

func (manager *SMonitorResourceManager) SyncManually(ctx context.Context) {
	manager.apih.RunManually(ctx)
}

func (manager *SMonitorResourceManager) SyncResources(ctx context.Context, mss *MonitorResModelSets) error {
	userCred := auth.AdminCredential()
	errs := make([]error, 0)
	log.Infof("start sync monitorresource")
	aliveIds := make([]string, 0)
	for _, set := range mss.ModelSetList() {
		setRv := reflect.ValueOf(set)
		needSync, typ := manager.GetSetType(set)
		log.Infof("Type: %s, length: %d", typ, len(setRv.MapKeys()))
		if !needSync {
			log.Infof("Type: %s don't need sync", typ)
			continue
		}
		for _, kRv := range setRv.MapKeys() {
			mRv := setRv.MapIndex(kRv)
			//log.Errorf("resID:%s", kRv.String())
			input := monitor.MonitorResourceListInput{
				ResId: []string{kRv.String()},
			}
			res, err := MonitorResourceManager.GetMonitorResources(input)
			if err != nil {
				return errors.Wrapf(err, "GetMonitorResources by input: %s", jsonutils.Marshal(input).String())
			}
			if mRv.IsValid() {
				aliveIds = append(aliveIds, kRv.String())
				obj := jsonutils.Marshal(mRv.Interface())
				if len(res) == 0 {
					// no find to create
					createData := newMonitorResourceCreateInput(obj, typ)
					_, err = db.DoCreate(MonitorResourceManager, ctx, userCred, nil, createData,
						userCred)
					if err != nil {
						name, _ := createData.GetString("name")
						errs = append(errs, errors.Wrapf(err, "monitorResource:%s resType:%s DoCreate err", name, typ))
					}
					continue
				}
				_, err = db.Update(&res[0], func() error {
					obj.(*jsonutils.JSONDict).Remove("id")
					(&res[0]).ResType = typ
					obj.Unmarshal(&res[0])
					return nil
				})
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "monitorResource:%s Update err", res[0].Name))
					continue
				}
				res[0].PostUpdate(ctx, userCred, jsonutils.NewDict(), newMonitorResourceCreateInput(obj, typ))
				continue
			}
			if len(res) != 0 {
				log.Infof("delete monitor resource,resId: %s,resType: %s", res[0].ResId, res[0].ResType)
				err := (&res[0]).RealDelete(ctx, userCred)
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "delete monitorResource:%s err", res[0].GetId()))
				}
			}
		}
		err := manager.DeleteMonitorResources(ctx, userCred, SdeleteRes{notIn: aliveIds, resType: typ})
		if err != nil {
			return err
		}
	}
	log.Infof("SMonitorResourceManager SyncResources End")
	err := CommonAlertManager.Run(ctx)
	if err != nil {
		log.Errorf("CommonAlertManager UpdateMonitorResourceJoint err:%v", err)
	}
	return errors.NewAggregate(errs)
}

func (manager *SMonitorResourceManager) GetSetType(set apihelper.IModelSet) (bool, string) {
	if iset, ok := set.(IMonitorResModelSet); ok {
		return iset.NeedSync(), iset.GetResType()
	}
	return false, "NONE"
}

func newMonitorResourceCreateInput(input jsonutils.JSONObject, typ string) jsonutils.JSONObject {
	monitorResource := jsonutils.DeepCopy(input).(*jsonutils.JSONDict)
	id, _ := monitorResource.GetString("id")
	monitorResource.Add(jsonutils.NewString(id), "res_id")
	monitorResource.Remove("id")
	monitorResource.Add(jsonutils.NewString(typ), "res_type")
	if monitorResource.Contains("metadata") {
		metadata, _ := monitorResource.Get("metadata")
		monitorResource.Add(metadata, "__meta__")
	}

	return monitorResource
}
