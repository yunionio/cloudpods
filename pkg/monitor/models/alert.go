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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	AlertMetadataTitle = "alert_title"
)

var (
	AlertManager *SAlertManager
)

func init() {
	AlertManager = NewAlertManager(SAlert{}, "alert", "alerts")
}

type AlertTestRunner interface {
	DoTest(ruleDef *SAlert, userCred mcclient.TokenCredential, input monitor.AlertTestRunInput) (*monitor.AlertTestRunOutput, error)
}

type SAlertManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager

	tester AlertTestRunner
}

func NewAlertManager(dt interface{}, keyword, keywordPlural string) *SAlertManager {
	man := &SAlertManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt,
			"alerts_tbl",
			keyword,
			keywordPlural),
	}
	man.SetVirtualObject(man)
	return man
}

func (man *SAlertManager) SetTester(tester AlertTestRunner) {
	man.tester = tester
}

func (man *SAlertManager) GetTester() AlertTestRunner {
	return man.tester
}

func (man *SAlertManager) FetchAllAlerts() ([]SAlert, error) {
	objs := make([]SAlert, 0)
	q := man.Query()
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return objs, nil
}

type SAlert struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	Frequency int64                `nullable:"false" list:"user" create:"required" update:"user"`
	Settings  jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
	Level     string               `charset:"ascii" width:"36"nullable:"false" default:"normal" list:"user" update:"user"`
	Message   string               `charset:"utf8" list:"user" update:"user"`
	UsedBy    string               `charset:"ascii" list:"user"`

	// Silenced       bool
	ExecutionError      string               `charset:"utf8" list:"user"`
	For                 int64                `nullable:"false" list:"user"`
	EvalData            jsonutils.JSONObject `list:"user" list:"user"`
	State               string               `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user"`
	NoDataState         string               `width:"36" charset:"ascii" nullable:"false" default:"pending" list:"user"`
	ExecutionErrorState string               `width:"36" charset:"ascii" nullable:"false" default:"alerting" list:"user"`
	LastStateChange     time.Time            `list:"user"`
	StateChanges        int                  `default:"0" nullable:"false" list:"user"`
}

func (alert *SAlert) IsEnable() bool {
	return alert.Enabled.Bool()
}

func (alert *SAlert) SetEnable() error {
	_, err := db.Update(alert, func() error {
		alert.SetEnabled(true)
		alert.State = string(monitor.AlertStatePending)
		return nil
	})
	return err
}

func (alert *SAlert) SetDisable() error {
	_, err := db.Update(alert, func() error {
		alert.SEnabledResourceBase.SetEnabled(false)
		alert.State = string(monitor.AlertStatePaused)
		return nil
	})
	return err
}

func (alert *SAlert) SetUsedBy(usedBy string) error {
	_, err := db.Update(alert, func() error {
		alert.UsedBy = usedBy
		return nil
	})
	return err
}

func (alert *SAlert) SetTitle(ctx context.Context, t string) error {
	return alert.SetMetadata(ctx, AlertMetadataTitle, t, nil)
}

func (alert *SAlert) GetTitle() string {
	return alert.GetMetadata(AlertMetadataTitle, nil)
}

func (alert *SAlert) ShouldUpdateState(newState monitor.AlertStateType) bool {
	return monitor.AlertStateType(alert.State) != newState
}

func (alert *SAlert) GetSettings() (*monitor.AlertSetting, error) {
	setting := new(monitor.AlertSetting)
	if alert.Settings == nil {
		return setting, nil
	}
	if err := alert.Settings.Unmarshal(setting); err != nil {
		return nil, errors.Wrapf(err, "alert %s unmarshal", alert.GetId())
	}
	return setting, nil
}

type AlertRuleTags map[string]AlertRuleTag

type AlertRuleTag struct {
	Key   string
	Value string
}

func setAlertDefaultSetting(setting *monitor.AlertSetting, dsId string) *monitor.AlertSetting {
	for idx, cond := range setting.Conditions {
		cond = setAlertDefaultCondition(cond, dsId)
		setting.Conditions[idx] = cond
	}
	return setting
}

func setAlertDefaultCreateData(data monitor.AlertCreateInput, dsId string) monitor.AlertCreateInput {
	setting := setAlertDefaultSetting(&data.Settings, dsId)
	data.Settings = *setting
	enable := true
	if data.Enabled == nil {
		data.Enabled = &enable
	}
	return data
}

func setAlertDefaultCondition(cond monitor.AlertCondition, dsId string) monitor.AlertCondition {
	if cond.Type == "" {
		cond.Type = "query"
	}
	if cond.Query.To == "" {
		cond.Query.To = "now"
	}
	if cond.Operator == "" {
		cond.Operator = "and"
	}
	if cond.Query.DataSourceId == "" {
		cond.Query.DataSourceId = dsId
	}
	return cond
}

func (man *SAlertManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, data monitor.AlertCreateInput) (monitor.AlertCreateInput, error) {
	ds, err := DataSourceManager.GetDefaultSource()
	if err != nil {
		return data, errors.Wrap(err, "get default data source")
	}
	data = setAlertDefaultCreateData(data, ds.GetId())
	if err := validators.ValidateAlertCreateInput(data); err != nil {
		return data, err
	}
	return data, nil
}

func (man *SAlertManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (man *SAlertManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (a *SAlert) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.AlertDetails, error) {
	return monitor.AlertDetails{}, nil
}

func (man *SAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertDetails {
	rows := make([]monitor.AlertDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertDetails{
			VirtualResourceDetails: virtRows[i],
		}
	}
	return rows
}

func (man *SAlertManager) GetAlert(id string) (*SAlert, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return obj.(*SAlert), nil
}

func GetMeasurementField(metric string) (string, string, error) {
	parts := strings.Split(metric, ".")
	if len(parts) != 2 {
		return "", "", httperrors.NewInputParameterError("metric %s is invalid format, usage <measurement>.<field>", metric)
	}
	measurement, field := parts[0], parts[1]
	return measurement, field, nil
}

func IsQuerySelectHasField(selects monitor.MetricQuerySelect, field string) bool {
	for _, s := range selects {
		if s.Type == "field" && len(s.Params) == 1 {
			if s.Params[0] == field {
				return true
			}
		}
	}
	return false
}

func (man *SAlertManager) CustomizeFilterList(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject) (
	*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()
	return filters, nil
}

func (alert *SAlert) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	alert.LastStateChange = time.Now()
	alert.State = string(monitor.AlertStateUnknown)
	return alert.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (alert *SAlert) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return db.IsProjectAllowPerform(userCred, alert, "enable")
}

func (alert *SAlert) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(alert, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (alert *SAlert) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return db.IsProjectAllowPerform(userCred, alert, "disable")
}

func (alert *SAlert) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(alert, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

const (
	ErrAlertChannotChangeStateOnPaused = errors.Error("Cannot change state on pause alert")
)

type AlertSetStateInput struct {
	State          monitor.AlertStateType
	EvalData       jsonutils.JSONObject
	ExecutionError string
}

func (alert *SAlert) SetState(input AlertSetStateInput) error {
	if alert.State == string(monitor.AlertStatePaused) {
		return ErrAlertChannotChangeStateOnPaused
	}
	if alert.State == string(input.State) {
		return nil
	}
	_, err := db.Update(alert, func() error {
		alert.State = string(input.State)
		alert.LastStateChange = time.Now()
		alert.EvalData = input.EvalData
		alert.ExecutionError = input.ExecutionError
		alert.StateChanges = alert.StateChanges + 1
		return nil
	})
	return err
}

func (alert *SAlert) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input monitor.AlertUpdateInput) (monitor.AlertUpdateInput, error) {
	if input.Settings != nil {
		updateSettings := jsonutils.NewDict()
		updateSettings.Update(alert.Settings)
		updateSettings.Update(jsonutils.Marshal(input.Settings))
		input.Settings = new(monitor.AlertSetting)
		if err := updateSettings.Unmarshal(input.Settings); err != nil {
			return input, err
		}
	}
	var err error
	input.VirtualResourceBaseUpdateInput, err = alert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}

	return input, nil
}

func (alert *SAlert) IsAttachNotification(noti *SNotification) (bool, error) {
	q := AlertNotificationManager.Query().Equals("notification_id", noti.GetId()).Equals("alert_id", alert.GetId())
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (alert *SAlert) GetNotificationsQuery() *sqlchemy.SQuery {
	return AlertNotificationManager.Query().Equals("alert_id", alert.GetId())
}

func (alert *SAlert) GetNotifications() ([]SAlertnotification, error) {
	notis := make([]SAlertnotification, 0)
	q := alert.GetNotificationsQuery().Asc("index")
	if err := db.FetchModelObjects(AlertNotificationManager, q, &notis); err != nil {
		return nil, err
	}
	return notis, nil
}

func (alert *SAlert) getNotificationIndex() (int8, error) {
	notis, err := alert.GetNotifications()
	if err != nil {
		return -1, err
	}
	var max uint
	for i := 0; i < len(notis); i++ {
		if uint(notis[i].Index) > max {
			max = uint(notis[i].Index)
		}
	}

	idxs := make([]int, max+1)
	for i := 0; i < len(notis); i++ {
		idxs[notis[i].Index] = 1
	}

	// find first idx not set
	for i := 0; i < len(idxs); i++ {
		if idxs[i] != 1 {
			return int8(i), nil
		}
	}

	return int8(max + 1), nil
}

func (alert *SAlert) AttachNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	noti *SNotification,
	state monitor.AlertNotificationStateType,
	usedBy string) (*SAlertnotification, error) {
	attached, err := alert.IsAttachNotification(noti)
	if err != nil {
		return nil, err
	}
	if attached {
		return nil, httperrors.NewNotAcceptableError("alert already attached to notification")
	}

	defer lockman.ReleaseObject(ctx, alert)
	lockman.LockObject(ctx, alert)
	alertNoti := new(SAlertnotification)
	alertNoti.AlertId = alert.GetId()
	alertNoti.Index, err = alert.getNotificationIndex()
	alertNoti.NotificationId = noti.GetId()
	if err != nil {
		return nil, err
	}
	alertNoti.State = string(state)
	alertNoti.UsedBy = usedBy
	if err := alertNoti.DoSave(ctx, userCred); err != nil {
		return nil, err
	}
	return alertNoti, nil
}

func (alert *SAlert) SetFor(forTime time.Duration) error {
	_, err := db.Update(alert, func() error {
		alert.For = int64(forTime)
		return nil
	})
	return err
}

func (alert *SAlert) AllowPerformTestRun(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsProjectAllowPerform(userCred, alert, "test-run")
}

func (alert *SAlert) PerformTestRun(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.AlertTestRunInput,
) (*monitor.AlertTestRunOutput, error) {
	return alert.testRunAlert(userCred, input)
}

func (alert *SAlert) testRunAlert(userCred mcclient.TokenCredential, input monitor.AlertTestRunInput) (*monitor.AlertTestRunOutput, error) {
	return AlertManager.GetTester().DoTest(alert, userCred, input)
}

/*func (alert *SAlert) AllowPerformAttachNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsProjectAllowPerform(userCred, alert, "attach-notification")
}

func (alert *SAlert) PerformAttachNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.AlertAttachNotificationInput,
) (*monitor.AlertAttachNotificationOutput, error) {
	notiObj, err := NotificationManager.FetchByIdOrName(userCred, input.NotificationId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("Alert notification %s not found", input.NotificationId)
		}
		return nil, err
	}
	noti := notiObj.(*SNotification)
	ret, err := alert.AttachNotification(ctx, userCred, noti, monitor.AlertNotificationStateUnknown, input.UsedBy)
	if err != nil {
		return nil, err
	}
	return &monitor.AlertAttachNotificationOutput{
		NotificationId: ret.NotificationId,
		UsedBy: ret.UsedBy,
	}, nil
}*/

func (alert *SAlert) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	notis, err := alert.GetNotifications()
	if err != nil {
		return err
	}
	for _, noti := range notis {
		if err := noti.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}
