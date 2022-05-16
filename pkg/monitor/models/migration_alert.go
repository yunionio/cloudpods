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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	migrationAlertMan *SMigrationAlertManager
)

func init() {

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

	MetricType string `create:"admin_required" list:"admin"`
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

	aInput := *(input.ToAlertCreateInput())
	aInput, err := AlertManager.ValidateCreateData(ctx, userCred, nil, query, aInput)
	if err != nil {
		return input, errors.Wrap(err, "AlertManager.ValidateCreateData")
	}
	input.AlertCreateInput = aInput

	return input, nil
}

func (alert *SMigrationAlert) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	out := new(monitor.MigrationAlertCreateInput)
	if err := data.Unmarshal(out); err != nil {
		return errors.Wrap(err, "Unmarshal to MigrationAlertCreateInput")
	}
	fs := out.GetMetricDriver().GetQueryFields()
	alert.ResType = string(fs.ResourceType)

	if err := alert.SAlert.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return errors.Wrap(err, "SAlert.CustomizeCreate")
	}
	return alert.CreateNotification(ctx, userCred)
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
	noti, err := NotificationManager.CreateAutoMigrationNotification(ctx, userCred, alert.GetName())
	if err != nil {
		return errors.Wrap(err, "CreateAutoMigrationNotification")
	}
	if alert.Id == "" {
		alert.Id = db.DefaultUUIDGenerator()
	}
	if _, err := alert.AttachNotification(ctx, userCred, noti, monitor.AlertNotificationStateUnknown, ""); err != nil {
		return errors.Wrap(err, "alert.AttachNotification")
	}
	return nil
}

func (alert *SMigrationAlert) GetMetricType() monitor.MigrationAlertMetricType {
	return monitor.MigrationAlertMetricType(alert.MetricType)
}
