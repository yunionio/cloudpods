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
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	DEFAULT_SHEILD_TIME = 100 // year
)

var (
	AlertRecordShieldManager *SAlertRecordShieldManager
)

type SAlertRecordShieldManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
}

func init() {
	AlertRecordShieldManager = &SAlertRecordShieldManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertRecordShield{},
			"alertrecordshield_tbl",
			"alertrecordshield",
			"alertrecordshields",
		),
	}

	AlertRecordShieldManager.SetVirtualObject(AlertRecordShieldManager)
}

type SAlertRecordShield struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource

	AlertId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" json:"alert_id"`
	ResId   string `width:"36" nullable:"false"  create:"optional" list:"user" update:"user" json:"res_id"`
	ResType string `width:"36" nullable:"false"  create:"optional" list:"user" update:"user" json:"res_type"`

	StartTime time.Time `required:"optional" list:"user" update:"user" json:"start_time"`
	EndTime   time.Time `required:"optional" list:"user" update:"user" json:"end_time"`
}

func (manager *SAlertRecordShieldManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (manager *SAlertRecordShieldManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertRecordShieldListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
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

	q = manager.shieldListByDetailsFeild(q, query)
	return q, nil
}

func (manager *SAlertRecordShieldManager) shieldListByDetailsFeild(query *sqlchemy.SQuery,
	input monitor.AlertRecordShieldListInput) *sqlchemy.SQuery {
	if len(input.ResId) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("res_id"), input.ResId))
	}
	if len(input.ResType) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("res_type"), input.ResType))
	}
	if len(input.AlertId) != 0 {
		query.Filter(sqlchemy.Equals(query.Field("alert_id"), input.AlertId))
	}

	alertQuery := CommonAlertManager.Query().SubQuery()
	if len(input.AlertName) != 0 {
		query.Join(alertQuery, sqlchemy.Equals(query.Field("alert_id"),
			alertQuery.Field("id"))).Filter(sqlchemy.Equals(alertQuery.Field("name"), input.AlertName))
	}
	if len(input.ResName) != 0 {
		resQuery := MonitorResourceManager.Query().SubQuery()
		query.Join(resQuery, sqlchemy.Equals(query.Field("res_id"), resQuery.Field("res_id"))).Filter(sqlchemy.Equals(
			resQuery.Field("name"), input.ResName))
	}
	if input.StartTime != nil {
		query.Filter(sqlchemy.LE(query.Field("start_time"), input.StartTime))
	}
	if input.EndTime != nil {
		query.Filter(sqlchemy.GE(query.Field("end_time"), input.EndTime))
	}
	return query
}

func (man *SAlertRecordShieldManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertRecordListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SAlertRecordShieldManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertRecordShieldDetails {
	rows := make([]monitor.AlertRecordShieldDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertRecordShieldDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SAlertRecordShield).GetMoreDetails(ctx, rows[i])
	}
	return rows
}

func (shield *SAlertRecordShield) GetMoreDetails(ctx context.Context, out monitor.AlertRecordShieldDetails) (monitor.
	AlertRecordShieldDetails, error) {
	commonAlert, err := CommonAlertManager.GetAlert(shield.AlertId)
	if err != nil {
		log.Errorf("GetAlert byId:%s err:%v", shield.AlertId, err)
		return out, nil
	}
	// Alert May delete By someone
	out.AlertName = commonAlert.Name
	alertDetails, err := commonAlert.GetMoreDetails(ctx, monitor.CommonAlertDetails{})
	out.CommonAlertDetails = alertDetails

	resources, err := MonitorResourceManager.GetMonitorResources(monitor.MonitorResourceListInput{ResId: []string{shield.ResId}})
	if err != nil {
		return out, errors.Errorf("getMonitorResource:%s err:%v", shield.ResId, err)
	}
	if len(resources) == 0 {
		return out, nil
	}
	if shield.EndTime.Before(time.Now()) {
		out.Expired = true
	}
	out.ResName = resources[0].Name
	return out, nil
}

func (man *SAlertRecordShieldManager) HasName() bool {
	return false
}

func (man *SAlertRecordShieldManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject,
	data monitor.AlertRecordShieldCreateInput) (monitor.AlertRecordShieldCreateInput, error) {
	if len(data.AlertId) == 0 {
		return data, httperrors.NewInputParameterError("alert_id  is empty")
	}
	alert, err := CommonAlertManager.GetAlert(data.AlertId)
	if err != nil {
		return data, httperrors.NewInputParameterError("get resourceRecord err by:%s,err:%v", data.AlertId, err)
	}
	if len(data.ResName) == 0 {
		return data, httperrors.NewInputParameterError("shield res_name is empty")
	}
	if len(data.ResId) == 0 {
		return data, httperrors.NewInputParameterError("shield res_id is empty")
	} else {
		resources, err := MonitorResourceManager.GetMonitorResources(monitor.MonitorResourceListInput{ResId: []string{data.ResId}})
		if err != nil {
			return data, errors.Errorf("getMonitorResource:%s err:%v", data.ResId, err)
		}
		if len(resources) == 0 {
			return data, httperrors.NewInputParameterError("can not get resource by res_id:%s", data.ResId)
		}
		data.ResType = resources[0].ResType
	}
	if len(data.StartTime) == 0 {
		data.StartTime = time.Now().UTC().Format(timeutils.MysqlTimeFormat)
	}
	if len(data.EndTime) == 0 {
		data.EndTime = time.Now().AddDate(DEFAULT_SHEILD_TIME, 0, 0).UTC().Format(timeutils.MysqlTimeFormat)
	}
	startTime, err := timeutils.ParseTimeStr(data.StartTime)
	if err != nil {
		return data, httperrors.NewInputParameterError("parse start_time: %s err", data.StartTime)
	}
	endTime, err := timeutils.ParseTimeStr(data.EndTime)
	if err != nil {
		return data, httperrors.NewInputParameterError("parse end_time: %s err", data.EndTime)
	}

	if endTime.Before(startTime) {
		return data, httperrors.NewInputParameterError("end_time is before start_time")
	}

	hint := fmt.Sprintf("%s-%s", alert.Name, data.ResName)
	name, err := db.GenerateName(ctx, man, ownerId, hint)
	if err != nil {
		return data, errors.Wrap(err, "get GenerateName err")
	}
	data.Name = name
	return data, nil
}

func (shield *SAlertRecordShield) PostCreate(ctx context.Context,
	userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := new(monitor.AlertRecordShieldCreateInput)
	if err := data.Unmarshal(input); err != nil {
		log.Errorf("post create unmarshal input: %v", err)
		return
	}
	startTime, _ := timeutils.ParseTimeStr(input.StartTime)
	endTime, _ := timeutils.ParseTimeStr(input.EndTime)
	_, err := db.Update(shield, func() error {
		shield.StartTime = startTime
		shield.EndTime = endTime
		return nil
	})
	if err != nil {
		log.Errorf("PostCreate update startTime and endTime err: %v", err)
		return
	}
}

func (manager *SAlertRecordShieldManager) GetRecordShields(input monitor.AlertRecordShieldListInput) (
	[]SAlertRecordShield, error) {
	shields := make([]SAlertRecordShield, 0)
	shieldQuery := manager.Query()
	shieldQuery = manager.shieldListByDetailsFeild(shieldQuery, input)
	err := db.FetchModelObjects(manager, shieldQuery, &shields)
	if err != nil {
		return nil, err
	}
	return shields, nil
}
