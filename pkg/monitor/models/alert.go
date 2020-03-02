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

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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

type SAlertManager struct {
	db.SVirtualResourceBaseManager
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

	Frequency int64                `nullable:"false" list:"user" create:"required" update:"user"`
	Settings  jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
	Enabled   bool                 `nullable:"false" default:"false" list:"user" create:"optional"`

	Message string `charset:"utf8" list:"user" update:"user"`
	State   string `width:"36" charset:"ascii" list:"user"`
	// Silenced       bool
	ExecutionError string `charset:"utf8" list:"user"`
	For            int64  `nullable:"false" list:"user"`

	EvalData        jsonutils.JSONObject `list:"user"`
	LastStateChange time.Time            `json:"last_state_change" list:"user"`
	StateChanges    int                  `default:"0" nullable:"false" list:"user" json:"state_changes"`

	NoDataState         string `charset:"utf8" list:"user"`
	ExecutionErrorState string `charset:"utf8" list:"user"`
}

func (alert *SAlert) IsEnable() bool {
	return alert.Enabled
}

func (alert *SAlert) SetEnable() error {
	alert.Enabled = true
	return nil
}

func (alert *SAlert) SetDisable() error {
	alert.Enabled = false
	return nil
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

func (man *SAlertManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input monitor.AlertListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = db.ListEnableItemFilter(q, input.Enabled)
	if err != nil {
		return nil, err
	}
	return q, nil
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

func (alert *SAlert) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.AllowPerformEnable(alert, rbacutils.ScopeProject, userCred)
}

func (alert *SAlert) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return db.PerformEnable(alert, userCred)
}

func (alert *SAlert) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.AllowPerformDisable(alert, rbacutils.ScopeProject, userCred)
}

func (alert *SAlert) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return db.PerformDisable(alert, userCred)
}

func (alert *SAlert) GetNotifications() ([]SAlertNotification, error) {
	settings, err := alert.GetSettings()
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	nIds := settings.Notifications
	notis, err := AlertNotificationManager.GetNotifications(nIds)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return notis, nil
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

func (alert *SAlert) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input monitor.AlertUpdateInput) (*jsonutils.JSONDict, error) {
	if input.Enabled == nil {
		enable := true
		input.Enabled = &enable
	}
	input.Settings = setAlertDefaultSetting(input.Settings, "")
	return alert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.JSON(input))
}
