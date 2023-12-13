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
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	alertResourceManager *SAlertResourceManager
	adminUsers           *sync.Map
)

func init() {
	// init singleton alertResourceManager
	GetAlertResourceManager()
}

func GetAlertResourceManager() *SAlertResourceManager {
	if alertResourceManager == nil {
		alertResourceManager = &SAlertResourceManager{
			SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
				SAlertResource{},
				"alertresources_tbl",
				"alertresource",
				"alertresources",
			),
		}
		alertResourceManager.SetVirtualObject(alertResourceManager)
	}
	return alertResourceManager
}

type SAlertResourceManager struct {
	db.SStandaloneResourceBaseManager
}

// SAlertResource records alerting single resource, one resource has multi alerts
type SAlertResource struct {
	db.SStandaloneResourceBase
	Type string `charset:"ascii" width:"36" nullable:"false" create:"required" list:"user"`
}

func (m *SAlertResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := m.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	return q, nil
}

func (man *SAlertResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertResourceDetails {
	rows := make([]monitor.AlertResourceDetails, len(objs))
	stdRows := man.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertResourceDetails{
			StandaloneResourceDetails: stdRows[i],
		}
		rows[i] = objs[i].(*SAlertResource).getDetails(rows[i])
	}
	return rows
}

func (m *SAlertResourceManager) ReconcileFromRecord(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, record *SAlertRecord) error {
	matches, err := record.GetEvalData()
	if err != nil {
		return errors.Wrapf(err, "Get record %s eval data", record.GetId())
	}
	oldResources, err := m.getResourceFromAlertId(record.AlertId)
	if err != nil {
		return errors.Wrap(err, "ReconcileFromRecord getResourceFromAlertId error")
	}
	errs := make([]error, 0)
	for _, match := range matches {
		if err := m.reconcileFromRecordMatch(ctx, userCred, ownerId, record, match); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		delErrs := m.deleteOldResource(ctx, userCred, record, oldResources)
		if len(delErrs) != 0 {
			errs = append(errs, delErrs...)
		}
	}
	return errors.NewAggregate(errs)
}

func (m *SAlertResourceManager) deleteOldResource(ctx context.Context, userCred mcclient.TokenCredential,
	record *SAlertRecord, oldResources []SAlertResource) (errs []error) {
	matches, _ := record.GetEvalData()
	needDelResources := make([]SAlertResource, 0)
LoopRes:
	for _, oldResource := range oldResources {
		for _, match := range matches {
			resourceD, _ := GetAlertResourceDriver(match)
			if oldResource.Name == resourceD.GetUniqCond().Name {
				continue LoopRes
			}
		}
		needDelResources = append(needDelResources, oldResource)
	}
	for i, _ := range needDelResources {
		if err := needDelResources[i].DetachAlert(ctx, userCred, record.AlertId); err != nil {
			errs = append(errs, errors.Wrapf(err, "deleteOldResource remove resource %s alert %s",
				needDelResources[i].GetName(),
				record.AlertId))
		}
	}
	return
}

type AlertResourceUniqCond struct {
	Type monitor.AlertResourceType
	Name string
}

func (m *SAlertResourceManager) getResourceFromMatch(ctx context.Context, userCred mcclient.TokenCredential, drv IAlertResourceDriver, match monitor.EvalMatch) (*SAlertResource, error) {
	uniqCond := drv.GetUniqCond()
	if uniqCond == nil {
		return nil, errors.Errorf("alert resource driver %s get %#v uniq match condition is nil", drv.GetType(), match)
	}
	q := m.Query().Equals("type", uniqCond.Type).Equals("name", uniqCond.Name)
	objs := make([]SAlertResource, 0)
	if err := db.FetchModelObjects(m, q, &objs); err != nil {
		return nil, errors.Wrapf(err, "fetch alert resource by condition: %v", uniqCond)
	}
	if len(objs) == 0 {
		return nil, nil
	}
	if len(objs) > 2 {
		return nil, errors.Wrapf(sqlchemy.ErrDuplicateEntry, "duplicate resource match by %#v", uniqCond)
	}
	return &objs[0], nil
}

func (m *SAlertResourceManager) getResourceFromAlertId(alertId string) ([]SAlertResource, error) {
	searchResourceIdQuery := GetAlertResourceAlertManager().Query(GetAlertResourceAlertManager().GetMasterFieldName())
	searchResourceIdQuery = searchResourceIdQuery.Equals(GetAlertResourceAlertManager().GetSlaveFieldName(), alertId)
	query := m.Query().In("id", searchResourceIdQuery.SubQuery())
	objs := make([]SAlertResource, 0)
	if err := db.FetchModelObjects(m, query, &objs); err != nil {
		return nil, errors.Wrapf(err, "getResourceFromAlertId:%s error", alertId)
	}
	return objs, nil
}

func (m *SAlertResourceManager) reconcileFromRecordMatch(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, record *SAlertRecord, match monitor.EvalMatch) error {
	drv, err := GetAlertResourceDriver(match)
	if err != nil {
		return errors.Wrap(err, "get resource driver by match")
	}
	res, err := m.getResourceFromMatch(ctx, userCred, drv, match)
	if err != nil {
		return errors.Wrap(err, "getResourceFromMatch")
	}
	recordState := record.GetState()
	if recordState == monitor.AlertStateOK {
		// remove matched resource's related alert record
		if res == nil {
			return nil
		}
		if err := res.DetachAlert(ctx, userCred, record.AlertId); err != nil {
			return errors.Wrapf(err, "remove resource %s alert %s", res.GetName(), record.AlertId)
		}
	} else if recordState == monitor.AlertStateAlerting {
		// create or update resource's related alert record
		if err := m.createOrUpdateFromRecord(ctx, userCred, ownerId, drv, record, match, res); err != nil {
			return errors.Wrap(err, "create or update record resource")
		}
	}
	return nil
}

func (m *SAlertResourceManager) createOrUpdateFromRecord(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	drv IAlertResourceDriver,
	record *SAlertRecord,
	match monitor.EvalMatch,
	res *SAlertResource,
) error {
	if res == nil {
		return m.createFromRecord(ctx, userCred, ownerId, drv, record, match)
	} else {
		return res.updateFromRecord(ctx, userCred, drv, record, match)
	}
}

func (m *SAlertResourceManager) createFromRecord(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	drv IAlertResourceDriver,
	record *SAlertRecord,
	match monitor.EvalMatch,
) error {
	uniqCond := drv.GetUniqCond()
	input := &monitor.AlertResourceCreateInput{
		StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{
			Name: uniqCond.Name,
		},
		Type: uniqCond.Type,
	}
	obj, err := db.DoCreate(m, ctx, userCred, nil, input.JSON(input), ownerId)
	if err != nil {
		return errors.Wrapf(err, "create alert resource by data %s", input.JSON(input))
	}
	if err := obj.(*SAlertResource).attachAlert(ctx, record, match); err != nil {
		return errors.Wrapf(err, "resource %s attch alert by matches %v", obj.GetName(), match)
	}
	return nil
}

func (res *SAlertResource) GetJointAlert(alertId string) (*SAlertResourceAlert, error) {
	jm := GetAlertResourceAlertManager()
	jObj, err := jm.GetJointAlert(res, alertId)
	if err != nil {
		return nil, errors.Wrapf(err, "get joint alert by alertId %s", alertId)
	}
	return jObj, nil
}

func (res *SAlertResource) isAttach2Alert(alertId string) (*SAlertResourceAlert, bool, error) {
	jobj, err := res.GetJointAlert(alertId)
	if err != nil {
		return nil, false, err
	}
	if jobj == nil {
		return nil, false, nil
	}
	return jobj, true, nil
}

func (res *SAlertResource) getJointAlerts() ([]SAlertResourceAlert, error) {
	jm := GetAlertResourceAlertManager()
	jObjs := make([]SAlertResourceAlert, 0)
	q := jm.Query().Equals(jm.GetMasterFieldName(), res.GetId())
	if err := db.FetchModelObjects(jm, q, &jObjs); err != nil {
		return nil, errors.Wrapf(err, "get resource %s joint alerts", res.GetName())
	}
	return jObjs, nil
}

func (res *SAlertResource) getAttachedAlerts() ([]SCommonAlert, error) {
	jm := GetAlertResourceAlertManager()
	alerts := make([]SCommonAlert, 0)
	q := CommonAlertManager.Query()
	sq := jm.Query("alert_id").Equals(jm.GetMasterFieldName(), res.GetId()).SubQuery()
	q = q.In("id", sq)
	if err := db.FetchModelObjects(CommonAlertManager, q, &alerts); err != nil {
		return nil, errors.Wrapf(err, "get resource %s attached alerts", res.GetName())
	}
	return alerts, nil
}

func (res *SAlertResource) updateFromRecord(ctx context.Context, userCred mcclient.TokenCredential,
	drv IAlertResourceDriver, record *SAlertRecord, match monitor.EvalMatch) error {
	jObj, err := res.GetJointAlert(record.AlertId)
	if err != nil {
		return errors.Wrapf(err, "get joint alert by id %s", record.AlertId)
	}
	if jObj == nil {
		if err := res.attachAlert(ctx, record, match); err != nil {
			return errors.Wrapf(err, "resource %s update from record %s attch alert by matches %v", res.GetName(), record.AlertId, match)
		}
	} else {
		if err := jObj.UpdateData(record, &match); err != nil {
			return errors.Wrapf(err, "update joint object by matches %v", match)
		}
		if _, err := db.Update(res, func() error {
			res.Type = string(drv.GetType())
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (res *SAlertResource) attachAlert(ctx context.Context, record *SAlertRecord, match monitor.EvalMatch) error {
	_, attached, err := res.isAttach2Alert(record.AlertId)
	if err != nil {
		return errors.Wrap(err, "check isAttach2Alert")
	}
	if attached {
		return errors.Errorf("%s resource has attached to alert %s", res.GetName(), record.AlertId)
	}

	// create resource joint alert record into db
	jm := GetAlertResourceAlertManager()
	jObj, err := db.NewModelObject(jm)
	input := &monitor.AlertResourceAttachInput{
		AlertResourceId: res.GetId(),
		AlertId:         record.AlertId,
		AlertRecordId:   record.GetId(),
		Data:            match,
	}
	data := input.JSON(input)
	if err := data.Unmarshal(jObj); err != nil {
		return errors.Wrap(err, "unmarshal to resource joint alert")
	}
	jObj.(*SAlertResourceAlert).TriggerTime = record.CreatedAt
	if err := jm.TableSpec().Insert(ctx, jObj); err != nil {
		return errors.Wrap(err, "insert joint model")
	}
	return nil
}

func (res *SAlertResource) LogPrefix() string {
	return fmt.Sprintf("%s/%s/%s", res.GetId(), res.GetName(), res.Type)
}

func (res *SAlertResource) DetachAlert(ctx context.Context, userCred mcclient.TokenCredential, alertId string) error {
	log.Errorf("=====resource %s detach alert %s", res.GetName(), alertId)
	jObj, attached, err := res.isAttach2Alert(alertId)
	if err != nil {
		return errors.Wrap(err, "check isAttach2Alert")
	}
	if !attached {
		return nil
	}
	if err := jObj.Detach(ctx, userCred); err != nil {
		return errors.Wrap(err, "detach joint alert")
	}
	count, err := res.getAttachedAlertCount()
	if err != nil {
		return errors.Wrap(err, "getAttachedAlertCount when detachAlert")
	}
	if count == 0 {
		if err := res.SStandaloneResourceBase.Delete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "delete alert resource %s when no alert attached", res.LogPrefix())
		}
		log.Infof("alert resource %s deleted when no alert attached", res.LogPrefix())
	}
	return nil
}

func (res *SAlertResource) GetType() monitor.AlertResourceType {
	return monitor.AlertResourceType(res.Type)
}

func (res *SAlertResource) getAttachedAlertCount() (int, error) {
	alerts, err := res.getAttachedAlerts()
	if err != nil {
		return 0, errors.Wrap(err, "getAttachedAlerts")
	}
	return len(alerts), nil
}

func (res *SAlertResource) getDetails(input monitor.AlertResourceDetails) monitor.AlertResourceDetails {
	jointAlerts, err := res.getJointAlerts()
	if err != nil {
		log.Errorf("get %s resource joint alerts error: %v", res.GetName(), err)
	} else {
		input.Count = len(jointAlerts)
	}
	tags := make(map[string]string, 0)
	for _, jObj := range jointAlerts {
		data, err := jObj.GetData()
		if err != nil {
			log.Errorf("get resource %s joint object data error: %v", res.LogPrefix(), err)
			continue
		}
		for k, v := range data.Tags {
			tags[k] = v
		}
	}
	input.Tags = tags
	return input
}

func (res *SAlertResource) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := res.SStandaloneResourceBase.CustomizeDelete(ctx, userCred, query, data); err != nil {
		return errors.Wrap(err, "SStandaloneResourceBase.CustomizeDelete")
	}
	alerts, err := res.getJointAlerts()
	if err != nil {
		return errors.Wrap(err, "getJointAlerts when customize delete")
	}
	for _, alert := range alerts {
		if err := alert.Detach(ctx, userCred); err != nil {
			return errors.Wrap(err, "detach joint alert")
		}
	}
	return nil
}

/*func (manager *SAlertResourceManager) NotifyAlertResourceCount(ctx context.Context) error {
	log.Errorln("exec NotifyAlertResourceCount func")
	cn, err := manager.getResourceCount()
	if err != nil {
		return err
	}
	alertResourceCount := resourceCount{
		AlertResourceCount: cn,
	}
	if adminUsers == nil {
		manager.GetAdminRoleUsers(ctx, nil, true)
	}
	adminUsersTmp := *adminUsers
	ids := make([]string, 0)
	adminUsersTmp.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	if len(ids) == 0 {
		return fmt.Errorf("no find users in receivers has admin role")
	}
	//if len(ids) != 0 {
	//	notifyclient.RawNotifyWithCtx(ctx, ids, false, npk.NotifyByWebConsole, npk.NotifyPriorityCritical,
	//		"alertResourceCount", jsonutils.Marshal(&alertResourceCount))
	//	return nil
	//} else {
	//	return fmt.Errorf("no find users in receivers has admin role")
	//}
	// manager.sendWebsocketInfo(ids, alertResourceCount)
	return nil
}*/

type resourceCount struct {
	AlertResourceCount int `json:"alert_resource_count"`
}

func (manager *SAlertResourceManager) getResourceCount() (int, error) {
	query := manager.Query("id")
	cn, err := query.CountWithError()
	if err != nil {
		return cn, errors.Wrap(err, "SAlertResourceManager get resource count error")
	}

	return cn, nil
}

func (manager *SAlertResourceManager) GetAdminRoleUsers(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	if adminUsers == nil {
		adminUsers = new(sync.Map)
	}
	offset := 0
	query := jsonutils.NewDict()
	session := auth.GetAdminSession(ctx, "")
	rid, err := identity.RolesV3.GetId(session, "admin", jsonutils.NewDict())
	if err != nil {
		errors.Errorf("get role admin id error: %v", err)
		return
	}
	query.Add(jsonutils.NewString(rid), "role", "id")
	for {
		query.Set("offset", jsonutils.NewInt(int64(offset)))
		result, err := identity.RoleAssignments.List(session, query)
		if err != nil {
			errors.Errorf("get admin role list by query: %s, error: %v", query, err)
			return
		}
		for _, roleAssign := range result.Data {
			userId, err := roleAssign.GetString("user", "id")
			if err != nil {
				log.Errorf("get user.id from roleAssign %s: %v", roleAssign, err)
				continue
			}
			//_, err = .NotifyReceiver.GetById(session, userId, jsonutils.NewDict())
			//if err != nil {
			//	log.Errorf("Recipients GetById err:%v", err)
			//	continue
			//}
			adminUsers.Store(userId, roleAssign)
		}
		offset = result.Offset + len(result.Data)
		if offset >= result.Total {
			break
		}
	}
}

/*
func (manager *SAlertResourceManager) sendWebsocketInfo(uids []string, alertResourceCount resourceCount) {
	session := auth.GetAdminSession(context.Background(), "", "")
	params := jsonutils.NewDict()
	params.Set("obj_type", jsonutils.NewString("monitor"))
	params.Set("obj_id", jsonutils.NewString(""))
	params.Set("obj_name", jsonutils.NewString(""))
	params.Set("success", jsonutils.JSONTrue)
	params.Set("action", jsonutils.NewString("alertResourceCount"))
	params.Set("ignore_alert", jsonutils.JSONTrue)
	params.Set("notes", jsonutils.NewString(fmt.Sprintf("priority=%s; content=%s", string(notify.NotifyPriorityCritical),
		jsonutils.Marshal(&alertResourceCount).String())))
	for _, uid := range uids {
		params.Set("user_id", jsonutils.NewString(uid))
		params.Set("user", jsonutils.NewString(uid))
		_, err := websocket.Websockets.Create(session, params)
		if err != nil {
			log.Errorf("websocket send info err:%v", err)
		}
	}
}*/
