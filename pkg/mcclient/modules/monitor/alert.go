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

package monitor

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Alerts             *SAlertManager
	Notifications      *SNotificationManager
	Alertnotification  *SAlertnotificationManager
	AlertResources     *SAlertResourceManager
	AlertResourceAlert *SAlertResourceAlertManager
)

type SAlertManager struct {
	*modulebase.ResourceManager
}

func NewAlertManager() *SAlertManager {
	man := modules.NewMonitorV2Manager("alert", "alerts",
		[]string{"id", "name", "frequency", "enabled", "settings", "state"},
		[]string{})
	return &SAlertManager{
		ResourceManager: &man,
	}
}

type SNotificationManager struct {
	*modulebase.ResourceManager
}

func NewNotificationManager() *SNotificationManager {
	man := modules.NewMonitorV2Manager(
		"alert_notification", "alert_notifications",
		[]string{"id", "name", "type", "is_default", "disable_resolve_message", "send_reminder", "frequency", "settings"},
		[]string{})
	return &SNotificationManager{
		ResourceManager: &man,
	}
}

type SAlertnotificationManager struct {
	*modulebase.JointResourceManager
}

func NewAlertnotificationManager() *SAlertnotificationManager {
	man := modules.NewJointMonitorV2Manager("alertnotification", "alertnotifications",
		[]string{"Alert_ID", "Alert", "Notification_ID", "Notification", "Used_by", "State", "Frequency"},
		[]string{},
		Alerts, Notifications)
	return &SAlertnotificationManager{&man}
}

type SAlertResourceManager struct {
	*modulebase.ResourceManager
}

func NewAlertResourceManager() *SAlertResourceManager {
	man := modules.NewMonitorV2Manager("alertresource", "alertresources",
		[]string{"Id", "Name", "Type", "Count", "Tags"},
		[]string{})
	return &SAlertResourceManager{
		ResourceManager: &man,
	}
}

type SAlertResourceAlertManager struct {
	*modulebase.JointResourceManager
}

func NewAlertResourceAlertManager() *SAlertResourceAlertManager {
	man := modules.NewJointMonitorV2Manager("alertresourcealert", "alertresourcealerts",
		[]string{"Alert_Resource_ID", "Alert_Resource", "Alert_ID", "Alert", "Alert_Record_ID", "Trigger_Time", "Data", "Common_Alert_Metric_Details"},
		[]string{},
		AlertResources, Alerts)
	return &SAlertResourceAlertManager{&man}
}

func init() {
	Alerts = NewAlertManager()
	Notifications = NewNotificationManager()
	AlertResources = NewAlertResourceManager()
	for _, m := range []modulebase.IBaseManager{
		Alerts,
		Notifications,
		AlertResources,
	} {
		modules.Register(m)
	}

	Alertnotification = NewAlertnotificationManager()
	AlertResourceAlert = NewAlertResourceAlertManager()
	for _, m := range []modulebase.IBaseManager{
		Alertnotification,
		AlertResourceAlert,
	} {
		modules.Register(m)
	}
}

func (m *SAlertManager) DoCreate(s *mcclient.ClientSession, config *AlertConfig) (jsonutils.JSONObject, error) {
	input := config.ToAlertCreateInput()
	return m.Create(s, input.JSON(input))
}

func (m *SAlertManager) DoTestRun(s *mcclient.ClientSession, id string, input *monitor.AlertTestRunInput) (jsonutils.JSONObject, error) {
	ret, err := m.PerformAction(s, id, "test-run", input.JSON(input))
	if err != nil {
		return nil, errors.Wrap(err, "call test-run")
	}
	return ret, nil
}
