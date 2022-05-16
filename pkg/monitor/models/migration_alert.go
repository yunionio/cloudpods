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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	migrationAlertMan *SMigrationAlertManager
)

func init() {
	migrationAlertMan = GetMigrationAlertManager()
}

func GetMigrationAlertManager() *SMigrationAlertManager {
	if migrationAlertMan != nil {
		return migrationAlertMan
	}
	migrationAlertMan = &SMigrationAlertManager{
		SAlertManager: *NewAlertManager(SMigrationAlert{}, "migrationalert", "migrationalerts"),
	}
	migrationAlertMan.SetVirtualObject(migrationAlertMan)
	return migrationAlertMan
}

type SMigrationAlertManager struct {
	SAlertManager
}

type SMigrationAlert struct {
	SAlert

	MetricType   string               `create:"admin_required" list:"admin" get:"admin"`
	MigrateNotes jsonutils.JSONObject `nullable:"true" list:"admin" get:"admin" update:"admin" create:"admin_optional"`
}

type MigrateNoteGuest struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	HostId    string  `json:"host_id"`
	Host      string  `json:"host"`
	VCPUCount int     `json:"vcpu_count"`
	VMemSize  int     `json:"vmem_size"`
	Score     float64 `json:"score"`
}

type MigrateNoteTarget struct {
	Id    string  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type MigrateNote struct {
	Guest  *MigrateNoteGuest  `json:"guest"`
	Target *MigrateNoteTarget `json:"target_host"`
	Error  string             `json:"error"`
}

func (m *SMigrationAlertManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query monitor.MigrationAlertListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SAlertManager.ListItemFilter(ctx, q, userCred, query.AlertListInput)
	if err != nil {
		return nil, err
	}
	if len(query.MetricType) > 0 {
		q = q.Equals("metric_type", query.MetricType)
	}
	q = q.Equals("used_by", AlertNotificationUsedByMigrationAlert)
	return q, nil
}

func (m *SMigrationAlertManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *monitor.MigrationAlertCreateInput) (*monitor.MigrationAlertCreateInput, error) {
	if input.Period == "" {
		input.Period = "5m"
	}
	if _, err := time.ParseDuration(input.Period); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid period format: %s", input.Period)
	}

	if err := monitor.IsValidMigrationAlertMetricType(input.MetricType); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid metric_type %v", err)
	}

	if input.MigrationSettings == nil {
		input.MigrationSettings = &monitor.MigrationAlertSettings{
			Source: new(monitor.MigrationAlertSettingsSource),
			Target: new(monitor.MigrationAlertSettingsTarget),
		}
	}
	if err := m.ValidateMigrationSettings(input.MigrationSettings); err != nil {
		return nil, errors.Wrap(err, "validate migration settings")
	}

	aInput := *(input.ToAlertCreateInput())
	aInput, err := AlertManager.ValidateCreateData(ctx, userCred, nil, query, aInput)
	if err != nil {
		return input, errors.Wrap(err, "AlertManager.ValidateCreateData")
	}
	input.AlertCreateInput = aInput

	return input, nil
}

func (m *SMigrationAlertManager) ValidateMigrationSettings(s *monitor.MigrationAlertSettings) error {
	if s.Source != nil {
		if err := m.ValidateMigrationSettingsSource(s.Source); err != nil {
			return errors.Wrap(err, "validate source")
		}
	}
	if s.Target != nil {
		if err := m.ValidateMigrationSettingsTarget(s.Target); err != nil {
			return errors.Wrap(err, "validate target")
		}
	}
	return nil
}

func (m *SMigrationAlertManager) GetResourceByIdOrName(rType string, id string) (jsonutils.JSONObject, error) {
	ok, objs := MonitorResourceManager.GetResourceObjByResType(rType)
	if !ok {
		return nil, errors.Errorf("Get by %q", rType)
	}
	for _, obj := range objs {
		name, _ := obj.GetString("name")
		if name == id {
			return obj, nil
		}
		objId, _ := obj.GetString("id")
		if objId == id {
			return obj, nil
		}
	}
	return nil, errors.Errorf("Not found resource %q by %q", rType, id)
}

func (m *SMigrationAlertManager) GetHostByIdOrName(id string) (jsonutils.JSONObject, error) {
	return m.GetResourceByIdOrName(monitor.METRIC_RES_TYPE_HOST, id)
}

func (m *SMigrationAlertManager) GetGuestByIdOrName(id string) (jsonutils.JSONObject, error) {
	return m.GetResourceByIdOrName(monitor.METRIC_RES_TYPE_GUEST, id)
}

func (m *SMigrationAlertManager) validateResource(vf func(idOrName string) (jsonutils.JSONObject, error), ids []string) ([]string, error) {
	nIds := make([]string, len(ids))
	for idx, idName := range ids {
		obj, err := vf(idName)
		if err != nil {
			return nil, errors.Wrapf(err, "find by %s", idName)
		}
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrap(err, "get id")
		}
		nIds[idx] = id
	}
	return nIds, nil
}

func (m *SMigrationAlertManager) ValidateMigrationSettingsSource(input *monitor.MigrationAlertSettingsSource) error {
	if len(input.HostIds) != 0 {
		hIds, err := m.validateResource(m.GetHostByIdOrName, input.HostIds)
		if err != nil {
			return errors.Wrap(err, "validate host")
		}
		input.HostIds = hIds
	}
	if len(input.GuestIds) != 0 {
		gIds, err := m.validateResource(m.GetGuestByIdOrName, input.GuestIds)
		if err != nil {
			return errors.Wrap(err, "validate guest")
		}
		input.GuestIds = gIds
	}
	return nil
}

func (m *SMigrationAlertManager) ValidateMigrationSettingsTarget(input *monitor.MigrationAlertSettingsTarget) error {
	if len(input.HostIds) != 0 {
		hIds, err := m.validateResource(m.GetHostByIdOrName, input.HostIds)
		if err != nil {
			return errors.Wrap(err, "validate host")
		}
		input.HostIds = hIds
	}
	return nil
}

func (alert *SMigrationAlert) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	out := new(monitor.MigrationAlertCreateInput)
	if err := data.Unmarshal(out); err != nil {
		return errors.Wrap(err, "Unmarshal to MigrationAlertCreateInput")
	}
	fs := out.GetMetricDriver().GetQueryFields()
	alert.ResType = string(fs.ResourceType)
	alert.UsedBy = AlertNotificationUsedByMigrationAlert

	if err := alert.SAlert.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return errors.Wrap(err, "SAlert.CustomizeCreate")
	}
	return alert.CreateNotification(ctx, userCred)
}

func (m *SMigrationAlertManager) FetchAllMigrationAlerts() ([]SMigrationAlert, error) {
	objs := make([]SMigrationAlert, 0)
	q := m.Query()
	q = q.IsTrue("enabled")
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return objs, nil
}

func (m *SMigrationAlertManager) GetInMigrationAlerts() ([]*SMigrationAlert, error) {
	alerts, err := m.FetchAllMigrationAlerts()
	if err != nil {
		return nil, errors.Wrap(err, "FetchAllMigrationAlerts")
	}
	objs := make([]*SMigrationAlert, 0)
	for _, a := range alerts {
		if ok, _, _ := a.IsInMigrationProcess(); ok {
			tmp := a
			objs = append(objs, &tmp)
		}
	}
	return objs, nil
}

func (alert *SMigrationAlert) GetMigrateNotes() (map[string]MigrateNote, error) {
	if alert.MigrateNotes == nil {
		return make(map[string]MigrateNote, 0), nil
	}
	objs := make(map[string]MigrateNote, 0)
	if err := alert.MigrateNotes.Unmarshal(&objs); err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return objs, nil
}

func (alert *SMigrationAlert) SetMigrateNote(ctx context.Context, ns *MigrateNote, isDelete bool) error {
	_, err := db.UpdateWithLock(ctx, alert, func() error {
		curNotes, err := alert.GetMigrateNotes()
		if err != nil {
			return errors.Wrap(err, "GetMigrateNotes")
		}
		if isDelete {
			delete(curNotes, ns.Guest.Id)
		} else {
			curNotes[ns.Guest.Id] = *ns
		}
		alert.MigrateNotes = jsonutils.Marshal(curNotes)
		return nil
	})
	return err
}

func (alert *SMigrationAlert) IsInMigrationProcess() (bool, map[string]MigrateNote, error) {
	notes, err := alert.GetMigrateNotes()
	if err != nil {
		return false, nil, errors.Wrap(err, "GetMigrateNotes")
	}
	if len(notes) == 0 {
		return false, nil, nil
	}
	return true, nil, nil
}

func (alert *SMigrationAlert) GetMigrationSettings() (*monitor.MigrationAlertSettings, error) {
	if alert.CustomizeConfig == nil {
		return nil, errors.Errorf("CustomizeConfig is nil")
	}
	out := new(monitor.MigrationAlertSettings)
	if err := alert.CustomizeConfig.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (alert *SMigrationAlert) CreateNotification(ctx context.Context, userCred mcclient.TokenCredential) error {
	if alert.Id == "" {
		alert.Id = db.DefaultUUIDGenerator()
	}
	noti, err := NotificationManager.CreateAutoMigrationNotification(ctx, userCred, alert)
	if err != nil {
		return errors.Wrap(err, "CreateAutoMigrationNotification")
	}
	if _, err := alert.AttachNotification(ctx, userCred, noti, monitor.AlertNotificationStateUnknown, ""); err != nil {
		return errors.Wrap(err, "alert.AttachNotification")
	}
	return nil
}

func (alert *SMigrationAlert) GetMetricType() monitor.MigrationAlertMetricType {
	return monitor.MigrationAlertMetricType(alert.MetricType)
}

func (alert *SMigrationAlert) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := alert.deleteNotifications(ctx, userCred, query, data); err != nil {
		return errors.Wrap(err, "delete related notification")
	}
	return alert.SAlert.CustomizeDelete(ctx, userCred, query, data)
}
