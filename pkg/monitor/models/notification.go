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
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers"
)

var (
	NotificationManager *SNotificationManager
)

func init() {
	NotificationManager = NewNotificationManager()
}

// +onecloud:swagger-gen-model-singular=alert_notification
// +onecloud:swagger-gen-model-plural=alert_notifications
type SNotificationManager struct {
	db.SVirtualResourceBaseManager
}

type SAlertNotificationStateManager struct {
	db.SStandaloneResourceBaseManager
}

func NewNotificationManager() *SNotificationManager {
	man := &SNotificationManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SNotification{},
			"notifications_tbl",
			"alert_notification",
			"alert_notifications",
		),
	}
	man.SetAlias("notification", "notifications")
	man.SetVirtualObject(man)
	return man
}

type SNotification struct {
	db.SVirtualResourceBase

	Type                  string `nullable:"false" list:"user" create:"required"`
	IsDefault             bool   `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	SendReminder          bool   `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	DisableResolveMessage bool   `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	// unit is second
	Frequency            int64                `nullable:"false" default:"0" list:"user" create:"optional" update:"user"`
	Settings             jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
	LastSendNotification time.Time            `list:"user" create:"optional" update:"user"`
}

func (man *SNotificationManager) GetPlugin(typ string) (*notifydrivers.NotifierPlugin, error) {
	drv, err := notifydrivers.GetPlugin(typ)
	if err != nil {
		if errors.Cause(err) == notifydrivers.ErrUnsupportedNotificationType {
			return nil, httperrors.NewInputParameterError("unsupported notification type %s", typ)
		} else {
			return nil, err
		}
	}
	return drv, nil
}

func (man *SNotificationManager) GetNotification(id string) (*SNotification, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return obj.(*SNotification), nil
}

func (man *SNotificationManager) GetNotifications(ids []string) ([]SNotification, error) {
	objs := make([]SNotification, 0)
	notis := man.Query().SubQuery()
	q := notis.Query().Filter(sqlchemy.In(notis.Field("id"), ids))
	if err := db.FetchModelObjects(man, q, &objs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return objs, nil
}

func (man *SNotificationManager) GetNotificationsWithDefault(ids []string) ([]SNotification, error) {
	objs := make([]SNotification, 0)
	notis := man.Query().SubQuery()
	q := notis.Query().Filter(sqlchemy.In(notis.Field("id"), ids))
	//sqlchemy.OR(
	//	sqlchemy.IsTrue(notis.Field("is_default")),
	//	sqlchemy.In(notis.Field("id"), ids)))
	if err := db.FetchModelObjects(man, q, &objs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return objs, nil
}

func (man *SNotificationManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
	if input.Type == "" {
		return input, merrors.NewArgIsEmptyErr("type")
	}
	if input.SendReminder == nil {
		sendReminder := true
		input.SendReminder = &sendReminder
	}
	if input.DisableResolveMessage == nil {
		dr := false
		input.DisableResolveMessage = &dr
	}
	plug, err := man.GetPlugin(input.Type)
	if err != nil {
		return input, err
	}
	return plug.ValidateCreateData(userCred, input)
}

func (man *SNotificationManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, input monitor.NotificationListInput) (*sqlchemy.SQuery, error) {

	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	if len(input.Type) > 0 {
		q = q.Equals("type", input.Type)
	}
	return q, err
}

func (man *SNotificationManager) CreateOneCloudNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	alertName string,
	channel string,
	userIds []string, silentPeriod string) (*SNotification, error) {
	settings := &monitor.NotificationSettingOneCloud{
		Channel: channel,
		UserIds: userIds,
	}

	newName, err := db.GenerateName(ctx, man, userCred, alertName)
	if err != nil {
		return nil, errors.Wrapf(err, "generate name: %s", alertName)
	}
	input := &monitor.NotificationCreateInput{
		Name:     newName,
		Type:     monitor.AlertNotificationTypeOneCloud,
		Settings: jsonutils.Marshal(settings),
	}
	if silentPeriod != "" {
		duration, _ := time.ParseDuration(silentPeriod)
		input.Frequency = duration / time.Second
	}
	obj, err := db.DoCreate(man, ctx, userCred, nil, input.JSON(input), userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "create notification input: %s", input.JSON(input))
	}
	return obj.(*SNotification), nil
}

func (n *SNotification) AttachToAlert(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	alertId string) (*SAlertnotification, error) {
	alert, err := AlertManager.GetAlert(alertId)
	if err != nil {
		return nil, err
	}
	return alert.AttachNotification(ctx, userCred, n, monitor.AlertNotificationStateUnknown, "")
}

func (n *SNotification) GetAlertNotificationCount() (int, error) {
	alertNotis := AlertNotificationManager.Query()
	return alertNotis.Equals("notification_id", n.Id).CountWithError()
}

func (n *SNotification) IsAttached() (bool, error) {
	cnt, err := n.GetAlertNotificationCount()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (n *SNotification) ValidateDeleteCondition(ctx context.Context) error {
	cnt, err := n.GetAlertNotificationCount()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("Alert notification used by %d alert", cnt)
	}
	return n.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (n *SNotification) ShouldSendNotification() bool {
	if n.Frequency == 0 {
		return true
	}
	if int64(time.Now().Sub(n.LastSendNotification)/time.Second)+int64(60) >= n.Frequency {
		return true
	}
	return false
}

func (n *SNotification) UpdateSendTime() error {
	_, err := db.Update(n, func() error {
		n.LastSendNotification = time.Now()
		return nil
	})
	return err
}
