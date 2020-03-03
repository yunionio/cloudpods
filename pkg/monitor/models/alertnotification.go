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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	AlertNotificationUsedByMeterAlert = "meter_alert"
	AlertNotificationUsedByNodeAlert  = "node_alert"
)

type SAlertNotificationManager struct {
	SAlertJointsManager
}

var AlertNotificationManager *SAlertNotificationManager

func init() {
	db.InitManager(func() {
		AlertNotificationManager = &SAlertNotificationManager{
			SAlertJointsManager: NewAlertJointsManager(
				SAlertnotification{},
				"alertnotifications_tbl",
				"alertnotification",
				"alertnotifications",
				NotificationManager),
		}
		AlertNotificationManager.SetVirtualObject(AlertNotificationManager)
		AlertNotificationManager.TableSpec().AddIndex(true, "notification_id", "alert_id")
	})
}

type SAlertnotification struct {
	SAlertJointsBase
	NotificationId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	State          string `nullable:"false" list:"user" create:"required"`
	Index          int8   `nullable:"false" default:"0" list:"user" list:"user" update:"user"`
	UsedBy         string `width:"36" charset:"ascii" nullable:"true" list:"user"`
}

func (man *SAlertNotificationManager) GetSlaveFieldName() string {
	return "notification_id"
}

func (man *SAlertNotificationManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (man *SAlertNotificationManager) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (man *SAlertNotificationManager) Get(alertId string, notiId string) (*SAlertnotification, error) {
	q := man.Query().Equals("alert_id", alertId).Equals("notification_id", notiId)
	obj := new(SAlertnotification)
	err := q.First(obj)
	obj.SetModelManager(man, obj)
	return obj, err
}

func (joint *SAlertnotification) DoSave(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := AlertNotificationManager.TableSpec().Insert(joint); err != nil {
		return err
	}
	joint.SetModelManager(AlertNotificationManager, joint)
	return nil
}

func (joint *SAlertnotification) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SAlertnotification) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (joint *SAlertnotification) GetNotification() (*SNotification, error) {
	noti, err := NotificationManager.GetNotification(joint.NotificationId)
	if err != nil {
		return nil, err
	}
	return noti, nil
}

func (joint *SAlertnotification) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, joint)
}

func (joint *SAlertnotification) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, joint)
}

func (joint *SAlertnotification) GetUsedBy() string {
	return joint.UsedBy
}

func (state *SAlertnotification) SetToPending() error {
	return state.setState(monitor.AlertNotificationStatePending)
}

func (state *SAlertnotification) SetToCompleted() error {
	return state.setState(monitor.AlertNotificationStateCompleted)
}

func (state *SAlertnotification) setState(changeState monitor.AlertNotificationStateType) error {
	_, err := db.Update(state, func() error {
		state.State = string(changeState)
		return nil
	})
	return err
}

func (state *SAlertnotification) GetState() monitor.AlertNotificationStateType {
	return monitor.AlertNotificationStateType(state.State)
}
