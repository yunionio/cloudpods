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
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AlertRecordManager *SAlertRecordManager
)

type SAlertRecordManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
}

type SAlertRecord struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource

	AlertId   string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Level     string               `charset:"ascii" width:"36" nullable:"false" default:"normal" list:"user" update:"user"`
	State     string               `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" update:"user"`
	SendState string               `width:"36" charset:"ascii" default:"ok" list:"user" update:"user"`
	EvalData  jsonutils.JSONObject `list:"user" update:"user"`
	AlertRule jsonutils.JSONObject `list:"user" update:"user"`
	ResType   string               `width:"36" list:"user" update:"user"`
}

func init() {
	AlertRecordManager = &SAlertRecordManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAlertRecord{},
			"alertrecord_tbl",
			"alertrecord",
			"alertrecords",
		),
	}

	AlertRecordManager.SetVirtualObject(AlertRecordManager)
}

func (manager *SAlertRecordManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SAlertRecordManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (manager *SAlertRecordManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertRecordListInput,
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
	if len(query.ResType) != 0 {
		q.Filter(sqlchemy.Equals(q.Field("res_type"), query.ResType))
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
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (man *SAlertRecordManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertRecordDetails {
	rows := make([]monitor.AlertRecordDetails, len(objs))
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertRecordDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
		}
		rows[i], _ = objs[i].(*SAlertRecord).GetMoreDetails(rows[i])
	}
	return rows
}

func (record *SAlertRecord) GetMoreDetails(out monitor.AlertRecordDetails) (monitor.AlertRecordDetails, error) {
	evalMatchs := make([]monitor.EvalMatch, 0)
	if record.EvalData != nil {
		err := record.EvalData.Unmarshal(&evalMatchs)
		if err != nil {
			return out, errors.Wrap(err, "record Unmarshal evalMatchs error")
		}
		for i, _ := range evalMatchs {
			evalMatchs[i] = record.filterTags(evalMatchs[i])
		}
		out.ResNum = int64(len(evalMatchs))
	}
	commonAlert, _ := CommonAlertManager.GetAlert(record.AlertId)
	out.AlertName = commonAlert.GetName()
	return out, nil
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
	err := record.SMonitorScopedResource.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
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
	record.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	err := MonitorResourceManager.UpdateMonitorResourceAttachJoint(ctx, userCred, record)
	if err != nil {
		log.Errorf("UpdateMonitorResourceAttachJoint error: %v", err)
	}
	if err := GetAlertResourceManager().ReconcileFromRecord(ctx, userCred, ownerId, record); err != nil {
		log.Errorf("Reconcile from alert record error: %v", err)
		return
	}
	err = GetAlertResourceManager().NotifyAlertResourceCount(ctx)
	if err != nil {
		log.Errorf("NotifyAlertResourceCount error: %v", err)
		return
	}
}

func (record *SAlertRecord) GetState() monitor.AlertStateType {
	return monitor.AlertStateType(record.State)
}

func (manager *SAlertRecordManager) DeleteRecordsOfThirtyDaysAgo(ctx context.Context, userCred mcclient.TokenCredential,
	isStart bool) {
	records := make([]SAlertRecord, 0)
	query := manager.Query()
	query = query.LE("created_at", timeutils.MysqlTime(time.Now().Add(-time.Hour*24*30)))
	log.Errorf("query:%s", query.String())
	err := db.FetchModelObjects(manager, query, &records)
	if err != nil {
		log.Errorf("fetch records ofthirty days ago err:%v", err)
		return
	}
	for i, _ := range records {
		err := db.DeleteModel(ctx, userCred, &records[i])
		if err != nil {
			log.Errorf("delete expire record:%s err:%v", records[i].GetId(), err)
		}
	}
}

func (manager *SAlertRecordManager) AllowGetPropertyTotalAlert(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (manager *SAlertRecordManager) GetPropertyTotalAlert(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(monitor.AlertRecordListInput)
	err := query.Unmarshal(input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal AlertRecordListInput error")
	}
	alertRecords, err := manager.getNowAlertingRecord(ctx, userCred, *input)
	if err != nil {
		return nil, errors.Wrap(err, "getNowAlertingRecord error")
	}
	if input.Details != nil && *input.Details {

	}
	alertCountMap := jsonutils.NewDict()
	for _, record := range alertRecords {
		evalMatches, err := record.GetEvalData()
		if err != nil {
			return nil, errors.Wrapf(err, "get record:%s evalData error", record.GetId())
		}
		count := int64(len(evalMatches))
		if alertCountMap.Contains(record.ResType) {
			resTypeCount, _ := alertCountMap.Int(record.ResType)
			count = count + resTypeCount
		}
		alertCountMap.Set(record.ResType, jsonutils.NewInt(count))
	}
	return alertCountMap, nil
}

func (manager *SAlertRecordManager) getNowAlertingRecord(ctx context.Context, userCred mcclient.TokenCredential,
	input monitor.AlertRecordListInput) ([]SAlertRecord, error) {
	//now := time.Now()
	//startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 1, now.Location())
	ownerId, err := manager.FetchOwnerId(context.Background(), jsonutils.Marshal(&input))
	if err != nil {
		return nil, errors.Wrap(err, "FetchOwnerId error")
	}
	if ownerId == nil {
		ownerId = userCred
	}
	query := manager.Query()
	query = manager.FilterByOwner(query, ownerId, rbacutils.String2Scope(input.Scope))
	//query = query.GE("created_at", startTime.UTC().Format(timeutils.MysqlTimeFormat))
	query = query.Equals("state", monitor.AlertStateAlerting)
	query = query.IsNotNull("res_type").IsNotEmpty("res_type").Desc("created_at")

	if len(input.ResType) != 0 {
		query = query.Equals("res_type", input.ResType)
	}

	alertsQuery := CommonAlertManager.Query("id").Equals("state", monitor.AlertStateAlerting).IsTrue("enabled").
		IsNull("used_by")
	alertsQuery = CommonAlertManager.FilterByOwner(alertsQuery, userCred, rbacutils.String2Scope(input.Scope))
	alerts := make([]SCommonAlert, 0)
	records := make([]SAlertRecord, 0)
	err = db.FetchModelObjects(CommonAlertManager, alertsQuery, &alerts)
	if err != nil {
		return nil, err
	}
	for _, alert := range alerts {
		tmp := *query
		recordModel, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		if err := (&tmp).Equals("alert_id", alert.GetId()).First(recordModel); err == nil {
			records = append(records, *(recordModel.(*SAlertRecord)))
		}
	}
	return records, nil
}
