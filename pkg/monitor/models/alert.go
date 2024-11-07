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
	"yunion.io/x/pkg/util/rbacscope"
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
	//db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	SMonitorScopedResourceManager
	//db.SStatusResourceBaseManager

	tester AlertTestRunner
}

func NewAlertManager(dt interface{}, keyword, keywordPlural string) *SAlertManager {
	man := &SAlertManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
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

func (manager *SAlertManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeSystem
}

func (manager *SAlertManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
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

func (man *SAlertManager) FetchAllAlerts() ([]SAlert, error) {
	objs := make([]SAlert, 0)
	q := man.Query()
	q = q.IsTrue("enabled")
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return objs, nil
}

type SAlert struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	SMonitorScopedResource
	//db.SStatusResourceBase

	// Frequency is evaluate period
	Frequency int64                 `nullable:"false" list:"user" create:"required" update:"user"`
	Settings  *monitor.AlertSetting `nullable:"false" list:"user" create:"required" update:"user" length:"medium"`
	Level     string                `charset:"ascii" width:"36" nullable:"false" default:"normal" list:"user" update:"user"`
	Message   string                `charset:"utf8" list:"user" create:"optional" update:"user"`
	UsedBy    string                `charset:"ascii" create:"optional" list:"user"`

	// Silenced       bool
	ExecutionError string `charset:"utf8" list:"user"`

	// If an alert rule has a configured `For` and the query violates the configured threshold
	// it will first go from `OK` to `Pending`. Going from `OK` to `Pending` will not send any
	// notifications. Once the alert rule has been firing for more than `For` duration, it will
	// change to `Alerting` and send alert notifications.
	For int64 `nullable:"false" list:"user" update:"user"`

	EvalData            jsonutils.JSONObject `list:"user" length:"medium"`
	State               string               `width:"36" charset:"ascii" nullable:"false" default:"unknown" list:"user" update:"user"`
	NoDataState         string               `width:"36" charset:"ascii" nullable:"false" default:"no_data" create:"optional"  list:"user" update:"user"`
	ExecutionErrorState string               `width:"36" charset:"ascii" nullable:"false" default:"alerting" create:"optional" list:"user" update:"user"`
	LastStateChange     time.Time            `list:"user"`
	StateChanges        int                  `default:"0" nullable:"false" list:"user"`
	CustomizeConfig     jsonutils.JSONObject `list:"user" create:"optional" update:"user" length:"medium"`
	ResType             string               `width:"32" list:"user" update:"user"`
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
	return alert.GetMetadata(context.Background(), AlertMetadataTitle, nil)
}

func (alert *SAlert) ShouldUpdateState(newState monitor.AlertStateType) bool {
	return monitor.AlertStateType(alert.State) != newState
}

func (alert *SAlert) GetSettings() (*monitor.AlertSetting, error) {
	return alert.Settings, nil
}

type AlertRuleTags map[string]AlertRuleTag

type AlertRuleTag struct {
	Key   string
	Value string
}

func setAlertDefaultSetting(setting *monitor.AlertSetting) *monitor.AlertSetting {
	for idx, cond := range setting.Conditions {
		cond = setAlertDefaultCondition(cond)
		setting.Conditions[idx] = cond
	}
	return setting
}

func setAlertDefaultCreateData(data monitor.AlertCreateInput) monitor.AlertCreateInput {
	setting := setAlertDefaultSetting(&data.Settings)
	data.Settings = *setting
	enable := true
	if data.Enabled == nil {
		data.Enabled = &enable
	}
	return data
}

func setAlertDefaultCondition(cond monitor.AlertCondition) monitor.AlertCondition {
	if cond.Type == "" {
		cond.Type = "query"
	}
	if cond.Query.To == "" {
		cond.Query.To = "now"
	}
	if cond.Operator == "" {
		cond.Operator = "and"
	}
	return cond
}

func (man *SAlertManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, data monitor.AlertCreateInput) (monitor.AlertCreateInput, error) {
	data = setAlertDefaultCreateData(data)
	if err := validators.ValidateAlertCreateInput(data); err != nil {
		return data, err
	}

	if err := man.validateStates(data.NoDataState, data.ExecutionErrorState); err != nil {
		return data, err
	}
	return data, nil
}

func (man *SAlertManager) validateStates(noData string, execErr string) error {
	if noData != "" {
		if err := man.validateNoDataState(monitor.NoDataOption(noData)); err != nil {
			return err
		}
	}

	if execErr != "" {
		if err := man.validateExecutionErrorState(monitor.ExecutionErrorOption(execErr)); err != nil {
			return err
		}
	}
	return nil
}

func (man *SAlertManager) validateNoDataState(state monitor.NoDataOption) error {
	if !state.IsValid() {
		return httperrors.NewInputParameterError("unsupported no_data_state %s", state)
	}
	return nil
}

func (man *SAlertManager) validateExecutionErrorState(state monitor.ExecutionErrorOption) error {
	if !state.IsValid() {
		return httperrors.NewInputParameterError("unsupported execution_error_state %s", state)
	}
	return nil
}

func (man *SAlertManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = man.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	if len(input.MonitorResourceId) != 0 {
		sq := MonitorResourceAlertManager.Query("alert_id").In("monitor_resource_id", input.MonitorResourceId).SubQuery()
		q = q.In("id", sq)
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

func (man *SAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SScopedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
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
	stdRows := man.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	scopedRows := man.SScopedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.AlertDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ScopedResourceBaseInfo:          scopedRows[i],
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
	return alert.SScopedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (alert *SAlert) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	alert.SetStatus(ctx, userCred, monitor.ALERT_STATUS_READY, "")
}

func (alert *SAlert) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(alert, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
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

func (alert *SAlert) GetExecutionErrorState() monitor.ExecutionErrorOption {
	return monitor.ExecutionErrorOption(alert.ExecutionErrorState)
}

func (alert *SAlert) GetNoDataState() monitor.NoDataOption {
	return monitor.NoDataOption(alert.NoDataState)
}

func (alert *SAlert) GetState() monitor.AlertStateType {
	return monitor.AlertStateType(alert.State)
}

type AlertSetStateInput struct {
	State           monitor.AlertStateType
	EvalData        jsonutils.JSONObject
	UpdateStateTime time.Time
	ExecutionError  string
}

func (alert *SAlert) SetState(input AlertSetStateInput) error {
	if alert.State == string(monitor.AlertStatePaused) {
		return ErrAlertChannotChangeStateOnPaused
	}
	_, err := db.Update(alert, func() error {
		alert.State = string(input.State)
		if input.State != monitor.AlertStatePending {
			alert.LastStateChange = input.UpdateStateTime
		}
		alert.EvalData = input.EvalData
		alert.ExecutionError = input.ExecutionError
		alert.StateChanges = alert.StateChanges + 1
		return nil
	})
	return err
}

func (alert *SAlert) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input monitor.AlertUpdateInput) (monitor.AlertUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = alert.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}

	if err := AlertManager.validateStates(input.NoDataState, input.ExecutionErrorState); err != nil {
		return input, err
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

func (alert *SAlert) deleteNotifications(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	notis, err := alert.GetNotifications()
	if err != nil {
		return err
	}
	for _, noti := range notis {
		conf, err := noti.GetNotification()
		if err != nil {
			return errors.Wrap(err, "GetNotification")
		}
		if err := conf.CustomizeDelete(ctx, userCred, query, data); err != nil {
			return errors.Wrapf(err, "notification %s(%s) CustomizeDelete", conf.GetName(), conf.GetId())
		}
		if err := noti.Detach(ctx, userCred); err != nil {
			return errors.Wrapf(err, "notification %s(%s) Detach ", conf.GetName(), conf.GetId())
		}
		if err := conf.Delete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "notification %s(%s) Delete", conf.GetName(), conf.GetId())
		}
	}
	return nil
}

func (alert *SAlert) GetAlertResources() ([]*SAlertResource, error) {
	jRess := make([]SAlertResourceAlert, 0)
	jm := GetAlertResourceAlertManager()
	q := jm.Query().Equals("alert_id", alert.GetId())
	if err := db.FetchModelObjects(jm, q, &jRess); err != nil {
		return nil, err
	}
	ress := make([]*SAlertResource, len(jRess))
	for idx := range jRess {
		res, err := jRess[idx].GetAlertResource()
		if err != nil {
			return nil, errors.Wrapf(err, "get alert %s related alret resource", alert.GetName())
		}
		ress[idx] = res
	}
	return ress, nil
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

	lockman.LockObject(ctx, alert)
	defer lockman.ReleaseObject(ctx, alert)

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

func (alert *SAlert) PerformTestRun(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.AlertTestRunInput,
) (*monitor.AlertTestRunOutput, error) {
	return alert.TestRunAlert(userCred, input)
}

func (alert *SAlert) TestRunAlert(userCred mcclient.TokenCredential, input monitor.AlertTestRunInput) (*monitor.AlertTestRunOutput, error) {
	return AlertManager.GetTester().DoTest(alert, userCred, input)
}

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
	alertRess, err := alert.GetAlertResources()
	if err != nil {
		return errors.Wrap(err, "get alert resources")
	}
	for _, res := range alertRess {
		if err := res.DetachAlert(ctx, userCred, alert.GetId()); err != nil {
			return errors.Wrapf(err, "detach alert resource %s", res.LogPrefix())
		}
	}
	return nil
}

func (alert *SAlert) PerformPause(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input monitor.AlertPauseInput,
) (jsonutils.JSONObject, error) {
	curState := alert.GetState()
	if curState != monitor.AlertStatePaused && !input.Paused {
		return nil, httperrors.NewNotAcceptableError("Alert is already un-paused")
	}

	if curState == monitor.AlertStatePaused && input.Paused {
		return nil, httperrors.NewNotAcceptableError("Alert is already paused")
	}

	var newState monitor.AlertStateType
	if input.Paused {
		newState = monitor.AlertStatePaused
	} else {
		newState = monitor.AlertStateUnknown
	}
	err := alert.SetState(AlertSetStateInput{
		State: newState,
	})
	return nil, err
}
