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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers"
)

var (
	AlertNotificationManager      *SAlertNotificationManager
	AlertNotificationStateManager *SAlertNotificationStateManager
)

func init() {
	AlertNotificationManager = NewAlertNotificationManager()
	AlertNotificationStateManager = NewAlertNotificationStateManager()
}

type SAlertNotificationManager struct {
	db.SVirtualResourceBaseManager
}

type SAlertNotificationStateManager struct {
	db.SStandaloneResourceBaseManager
}

func NewAlertNotificationManager() *SAlertNotificationManager {
	man := &SAlertNotificationManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAlertNotification{},
			"alert_notifications_tbl",
			"alert_notification",
			"alert_notifications",
		),
	}
	man.SetVirtualObject(man)
	return man
}

func NewAlertNotificationStateManager() *SAlertNotificationStateManager {
	man := &SAlertNotificationStateManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SAlertNotificationState{},
			"alert_notification_states_tbl",
			"alert_notification_state",
			"alert_notification_states",
		),
	}
	man.SetVirtualObject(man)
	return man
}

type SAlertNotification struct {
	db.SVirtualResourceBase

	Type                  string               `nullable:"false" list:"user" create:"required"`
	IsDefault             bool                 `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	SendReminder          bool                 `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	DisableResolveMessage bool                 `nullable:"false" default:"false" list:"user" create:"optional" update:"user"`
	Frequency             int64                `nullable:"false" default:"0" list:"user" create:"optional" update:"user"`
	Settings              jsonutils.JSONObject `nullable:"false" list:"user" create:"required" update:"user"`
}

type SAlertNotificationState struct {
	db.SStandaloneResourceBase

	AlertId    string `nullable:"false" list:"user" create:"required"`
	NotifierId string `nullable:"false" list:"user" create:"required"`
	State      string `nullable:"false" list:"user" create:"required"`
}

func (man *SAlertNotificationManager) GetPlugin(typ string) (*notifydrivers.NotifierPlugin, error) {
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

func (man *SAlertNotificationManager) GetNotification(id string) (*SAlertNotification, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return obj.(*SAlertNotification), nil
}

func (man *SAlertNotificationManager) GetNotifications(ids []string) ([]SAlertNotification, error) {
	objs := make([]SAlertNotification, 0)
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

func (man *SAlertNotificationManager) GetNotificationsWithDefault(ids []string) ([]SAlertNotification, error) {
	objs := make([]SAlertNotification, 0)
	notis := man.Query().SubQuery()
	q := notis.Query().Filter(
		sqlchemy.OR(
			sqlchemy.IsTrue(notis.Field("is_default")),
			sqlchemy.In(notis.Field("id"), ids)))
	if err := db.FetchModelObjects(man, q, &objs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return objs, nil
}

func (man *SAlertNotificationManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input monitor.AlertNotificationCreateInput) (monitor.AlertNotificationCreateInput, error) {
	if input.Type == "" {
		return input, httperrors.NewInputParameterError("notification type is empty")
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

func (man *SAlertNotificationManager) CreateOneCloudNotification(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	alertName string,
	channel string,
	userIds []string) (*SAlertNotification, error) {
	settings := &monitor.NotificationSettingOneCloud{
		Channel: channel,
		UserIds: userIds,
	}
	newName, err := db.GenerateName(man, userCred, alertName)
	if err != nil {
		return nil, errors.Wrapf(err, "generate name: %s", alertName)
	}
	input := &monitor.AlertNotificationCreateInput{
		Name:     newName,
		Type:     monitor.AlertNotificationTypeOneCloud,
		Settings: jsonutils.Marshal(settings),
	}
	obj, err := db.DoCreate(man, ctx, userCred, nil, input.JSON(input), userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "create notification input: %s", input.JSON(input))
	}
	return obj.(*SAlertNotification), nil
}

func (n *SAlertNotification) GetStates() ([]SAlertNotificationState, error) {
	states := AlertNotificationStateManager.Query().SubQuery()
	q := states.Query().Filter(sqlchemy.Equals(states.Field("notifier_id"), n.GetId()))
	objs := make([]SAlertNotificationState, 0)
	if err := db.FetchModelObjects(AlertNotificationStateManager, q, &objs); err != nil {
		return nil, err
	}
	return objs, nil
}

func (n *SAlertNotification) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	stats, err := n.GetStates()
	if err != nil {
		return err
	}
	for _, stat := range stats {
		if err := stat.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (man *SAlertNotificationStateManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	_ jsonutils.JSONObject,
	input monitor.AlertNotificationStateCreateInput) (monitor.AlertNotificationStateCreateInput, error) {
	if input.AlertId == "" {
		return input, httperrors.NewNotEmptyError("alert_id is empty")
	}
	if input.NotifierId == "" {
		return input, httperrors.NewNotEmptyError("notifier_id is empty")
	}
	var name string
	if obj, err := AlertManager.FetchById(input.AlertId); err != nil {
		return input, err
	} else {
		name = obj.GetName()
	}
	if obj, err := AlertNotificationManager.FetchById(input.NotifierId); err != nil {
		return input, err
	} else {
		name = fmt.Sprintf("%s_%s", name, obj.GetName())
	}
	name, err := db.GenerateName(man, ownerId, name)
	if err != nil {
		return input, err
	}
	input.Name = name
	return input, nil
}

func (man *SAlertNotificationStateManager) CreateState(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input monitor.AlertNotificationStateCreateInput) (*SAlertNotificationState, error) {
	obj, err := db.DoCreate(man, ctx, userCred, nil, input.JSON(input), userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "create notification state: %s", input.JSON(input))
	}
	return obj.(*SAlertNotificationState), nil
}

func (man *SAlertNotificationStateManager) GetState(alertId, notifierId string) (*SAlertNotificationState, error) {
	state := man.Query().SubQuery()
	q := state.Query().Filter(sqlchemy.AND(
		sqlchemy.Equals(state.Field("alert_id"), alertId),
		sqlchemy.Equals(state.Field("notifier_id"), notifierId)))
	obj := new(SAlertNotificationState)
	err := q.First(obj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return obj, nil
}

func (man *SAlertNotificationStateManager) GetOrCreateState(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	alertId string,
	notifierId string) (*SAlertNotificationState, error) {
	state, err := man.GetState(alertId, notifierId)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return man.CreateState(ctx, userCred, monitor.AlertNotificationStateCreateInput{
			AlertId:    alertId,
			NotifierId: notifierId,
			State:      monitor.AlertNotificationStateUnknown,
		})
	}
	state.SetModelManager(man, state)
	return state, nil
}

func (state *SAlertNotificationState) SetToPending() error {
	return state.setState(monitor.AlertNotificationStatePending)
}

func (state *SAlertNotificationState) SetToCompleted() error {
	return state.setState(monitor.AlertNotificationStateCompleted)
}

func (state *SAlertNotificationState) setState(changeState monitor.AlertNotificationStateType) error {
	_, err := db.Update(state, func() error {
		state.State = string(changeState)
		return nil
	})
	return err
}

func (state *SAlertNotificationState) GetState() monitor.AlertNotificationStateType {
	return monitor.AlertNotificationStateType(state.State)
}
