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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	AlertNotificationUsedByMeterAlert     = monitor.AlertNotificationUsedByMeterAlert
	AlertNotificationUsedByNodeAlert      = monitor.AlertNotificationUsedByNodeAlert
	AlertNotificationUsedByCommonAlert    = monitor.AlertNotificationUsedByCommonAlert
	AlertNotificationUsedByMigrationAlert = monitor.AlertNotificationUsedByMigrationAlert
)

// +onecloud:swagger-gen-ignore
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

// +onecloud:swagger-gen-ignore
type SAlertnotification struct {
	SAlertJointsBase
	NotificationId string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	State          string               `nullable:"false" list:"user" create:"required"`
	Index          int8                 `nullable:"false" default:"0" list:"user" list:"user" update:"user"`
	UsedBy         string               `width:"36" charset:"ascii" nullable:"true" list:"user"`
	Params         jsonutils.JSONObject `nullable:"true" list:"user" update:"user"`
}

func (man *SAlertNotificationManager) GetSlaveFieldName() string {
	return "notification_id"
}

func (man *SAlertNotificationManager) Get(alertId string, notiId string) (*SAlertnotification, error) {
	q := man.Query().Equals("alert_id", alertId).Equals("notification_id", notiId)
	obj := new(SAlertnotification)
	err := q.First(obj)
	obj.SetModelManager(man, obj)
	return obj, err
}

func (man *SAlertNotificationManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertNotificationListInput) (*sqlchemy.SQuery, error) {
	return man.SAlertJointsManager.ListItemFilter(ctx, q, userCred, query.AlertJointListInput)
}

func (man *SAlertNotificationManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertnotificationDetails {
	rows := make([]monitor.AlertnotificationDetails, len(objs))
	alertRows := man.SAlertJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	notiIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = monitor.AlertnotificationDetails{
			AlertJointResourceBaseDetails: alertRows[i],
		}
		notiIds[i] = objs[i].(*SAlertnotification).NotificationId
	}

	notis := make(map[string]SNotification)
	if err := db.FetchModelObjectsByIds(NotificationManager, "id", notiIds, notis); err != nil {
		return rows
	}

	for i := range rows {
		if noti, ok := notis[notiIds[i]]; ok {
			rows[i].Notification = noti.Name
			rows[i].Frequency = noti.Frequency
		}
	}
	return rows
}

func (man *SAlertNotificationManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input monitor.AlertnotificationCreateInput,
) (*jsonutils.JSONDict, error) {
	if input.AlertId == "" {
		return nil, httperrors.NewMissingParameterError("alert_id")
	}
	if input.NotificationId == "" {
		return nil, httperrors.NewMissingParameterError("notification_id")
	}
	_, err := AlertManager.FetchById(input.AlertId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("not find alert %s", input.AlertId)
		}
		return nil, err
	}
	_, err = NotificationManager.FetchById(input.NotificationId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("not find notification %s", input.NotificationId)
		}
		return nil, err
	}
	ret := input.JSON(input)
	ret.Add(jsonutils.NewString(string(monitor.AlertNotificationStateUnknown)), "state")
	return ret, nil
}

func (joint *SAlertnotification) getExtraDetails(noti SNotification, out monitor.AlertnotificationDetails) monitor.AlertnotificationDetails {
	out.Notification = noti.GetName()
	return out
}

func (joint *SAlertnotification) DoSave(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := AlertNotificationManager.TableSpec().Insert(ctx, joint); err != nil {
		return err
	}
	joint.SetModelManager(AlertNotificationManager, joint)
	return nil
}

func (joint *SAlertnotification) GetNotification() (*SNotification, error) {
	noti, err := NotificationManager.GetNotification(joint.NotificationId)
	if err != nil {
		return nil, err
	}
	return noti, nil
}

func (join *SAlertnotification) ShouldSendNotification() (bool, error) {
	notification, err := join.GetNotification()
	if err != nil {
		return false, errors.Wrap(err, "Alertnotification GetNotification err")
	}
	return notification.ShouldSendNotification(), nil
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

func (state *SAlertnotification) GetParams() jsonutils.JSONObject {
	return state.Params
}

func (joint *SAlertnotification) UpdateSendTime() error {
	notification, err := joint.GetNotification()
	if err != nil {
		return errors.Wrap(err, "GetNotification err")
	}
	return notification.UpdateSendTime()
}
