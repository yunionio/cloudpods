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

package alerting

import (
	"database/sql"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers"
)

type notificationService struct {
}

func newNotificationService() *notificationService {
	return &notificationService{}
}

func (n *notificationService) SendIfNeeded(evalCtx *EvalContext) error {
	notifierStates, shouldNotify, err := n.getNeededNotifiers(evalCtx.Rule.Notifications, evalCtx)
	if err != nil {
		return errors.Wrap(err, "failed to get alert notifiers")
	}

	n.syncResources(evalCtx, shouldNotify)

	if len(notifierStates) == 0 {
		return nil
	}

	return n.sendNotifications(evalCtx, notifierStates)
}

type notifierState struct {
	notifier Notifier
	state    *models.SAlertnotification
}

type notifierStateSlice []*notifierState

func (n *notificationService) sendNotification(evalCtx *EvalContext, state *notifierState) error {
	if !evalCtx.IsTestRun {
		if err := state.state.SetToPending(); err != nil {
			return errors.Wrap(err, "SetToPending")
		}
	}
	return n.sendAndMarkAsComplete(evalCtx, state)
}

func (n *notificationService) sendAndMarkAsComplete(evalCtx *EvalContext, state *notifierState) error {
	notifier := state.notifier

	log.Debugf("Sending notification, type %s, id %s", notifier.GetType(), notifier.GetNotifierId())

	if err := notifier.Notify(evalCtx, state.state.GetParams()); err != nil {
		return errors.Wrapf(err, "notify driver %s(%s)", notifier.GetType(), notifier.GetNotifierId())
	}

	if evalCtx.IsTestRun {
		return nil
	}
	err := state.state.UpdateSendTime()
	if err != nil {
		return errors.Wrap(err, "notifierState UpdateSendTime")
	}
	return state.state.SetToCompleted()
}

func (n *notificationService) sendNotifications(evalCtx *EvalContext, states notifierStateSlice) error {
	for _, state := range states {
		if err := n.sendNotification(evalCtx, state); err != nil {
			log.Errorf("failed to send %s(%s) notification: %v", state.notifier.GetType(), state.notifier.GetNotifierId(), err)
			if evalCtx.IsTestRun {
				return err
			}
		}
	}
	return nil
}

func (n *notificationService) getNeededNotifiers(nIds []string, evalCtx *EvalContext) (notifierStateSlice, bool, error) {
	notis, err := models.NotificationManager.GetNotificationsWithDefault(nIds)
	if err != nil {
		return nil, false, errors.Wrapf(err, "GetNotificationsWithDefault with %v", nIds)
	}

	var result notifierStateSlice
	shouldNotify := false
	for _, obj := range notis {
		not, err := InitNotifier(NotificationConfig{
			Ctx:                   evalCtx.Ctx,
			Id:                    obj.GetId(),
			Name:                  obj.GetName(),
			Type:                  obj.Type,
			Frequency:             time.Duration(obj.Frequency),
			SendReminder:          obj.SendReminder,
			DisableResolveMessage: obj.DisableResolveMessage,
			Settings:              obj.Settings,
		})
		if err != nil {
			log.Errorf("Could not create notifier %s, error: %v", obj.GetId(), err)
			continue
		}
		state, err := models.AlertNotificationManager.Get(evalCtx.Rule.Id, obj.GetId())
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				state, err = obj.AttachToAlert(evalCtx.Ctx, evalCtx.UserCred, evalCtx.Rule.Id)
				if err != nil {
					log.Errorf("Attach notification %s to alert %s error: %v", obj.GetName(), evalCtx.Rule.Id, err)
					continue
				}
			} else {
				log.Errorf("Get alert state: %v, alertId %s, notifierId: %s", err, evalCtx.Rule.Id, obj.GetId())
				continue
			}
		}

		if not.ShouldNotify(evalCtx.Ctx, evalCtx, state) {
			shouldNotify = true
			result = append(result, &notifierState{
				notifier: not,
				state:    state,
			})
		}
	}

	return result, shouldNotify, nil
}

func (n *notificationService) syncResources(evalCtx *EvalContext, shouldNotify bool) {
	if shouldNotify || evalCtx.Rule.State == monitor.AlertStateAlerting {
		go func() {
			if err := n.createAlertRecordWhenNotify(evalCtx, shouldNotify); err != nil {
				log.Errorf("createAlertRecordWhenNotify error: %v", err)
			}
		}()
	}
	if !shouldNotify && evalCtx.shouldUpdateAlertState() && evalCtx.NoDataFound {
		go func() {
			n.detachAlertResourceWhenNodata(evalCtx)
		}()
	}

	if len(evalCtx.AlertOkEvalMatches) > 0 {
		go func() {
			if err := n.syncMonitorResourceAlerts(evalCtx); err != nil {
				log.Errorf("syncMonitorResourceAlerts error: %v", err)
			}
		}()
	}
}

func (n *notificationService) createAlertRecordWhenNotify(evalCtx *EvalContext, shouldNotify bool) error {
	var matches []*monitor.EvalMatch
	if evalCtx.Firing {
		matches = evalCtx.EvalMatches
	} else {
		matches = evalCtx.AlertOkEvalMatches
	}
	n.dealNeedShieldEvalMatches(evalCtx, matches)
	recordCreateInput := monitor.AlertRecordCreateInput{
		StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{
			GenerateName: evalCtx.Rule.Name,
		},
		AlertId:   evalCtx.Rule.Id,
		Level:     evalCtx.Rule.Level,
		State:     string(evalCtx.Rule.State),
		SendState: monitor.SEND_STATE_OK,
		EvalData:  matches,
		AlertRule: evalCtx.Rule.RuleDescription,
	}
	if !shouldNotify {
		recordCreateInput.SendState = monitor.SEND_STATE_SILENT
	}
	recordCreateInput.ResType = recordCreateInput.AlertRule[0].ResType
	if len(recordCreateInput.ResType) == 0 {
		recordCreateInput.ResType = monitor.METRIC_RES_TYPE_HOST
	}
	createData := recordCreateInput.JSON(recordCreateInput)
	alert, _ := models.CommonAlertManager.GetAlert(evalCtx.Rule.Id)
	record, err := db.DoCreate(models.AlertRecordManager, evalCtx.Ctx, evalCtx.UserCred, jsonutils.NewDict(), createData, evalCtx.UserCred)
	if err != nil {
		return errors.Wrapf(err, "db.DoCreate")
	}
	alertData := jsonutils.Marshal(alert)
	alertData.(*jsonutils.JSONDict).Set("project_id", jsonutils.NewString(alert.GetProjectId()))
	db.PerformSetScope(evalCtx.Ctx, record.(*models.SAlertRecord), evalCtx.UserCred, alertData)
	dbMatches, _ := record.(*models.SAlertRecord).GetEvalData()
	if !evalCtx.Firing {
		evalCtx.AlertOkEvalMatches = make([]*monitor.EvalMatch, len(dbMatches))
		for i := range dbMatches {
			evalCtx.AlertOkEvalMatches[i] = &dbMatches[i]
		}
	}
	record.PostCreate(evalCtx.Ctx, evalCtx.UserCred, evalCtx.UserCred, nil, createData)
	return nil
}

func (n *notificationService) dealNeedShieldEvalMatches(evalCtx *EvalContext, match []*monitor.EvalMatch) {
	input := monitor.AlertRecordShieldListInput{
		ResType: evalCtx.Rule.RuleDescription[0].ResType,
		AlertId: evalCtx.Rule.Id,
	}
filterMatch:
	for i := range match {
		input.ResId = monitor.GetMeasurementResourceId(match[i].Tags, input.ResType)
		alertRecordShields, err := models.AlertRecordShieldManager.GetRecordShields(input)
		if err != nil {
			log.Errorf("GetRecordShields byAlertId:%s,err:%v", input.AlertId, err)
			return
		}
		if len(alertRecordShields) != 0 {
			for _, shield := range alertRecordShields {
				if shield.EndTime.After(time.Now().UTC()) && shield.StartTime.Before(time.Now().UTC()) {
					match[i].Tags[monitor.ALERT_RESOURCE_RECORD_SHIELD_KEY] = monitor.ALERT_RESOURCE_RECORD_SHIELD_VALUE
					continue filterMatch
				}
			}
		}
	}

}

func (n *notificationService) detachAlertResourceWhenNodata(evalCtx *EvalContext) {
	errs := models.CommonAlertManager.DetachAlertResourceByAlertId(evalCtx.Ctx, evalCtx.UserCred, evalCtx.Rule.Id)
	if len(errs) != 0 {
		log.Errorf("detachAlertResourceWhenNodata err:%#v", errors.NewAggregate(errs))
	}
}

func (n *notificationService) syncMonitorResourceAlerts(evalCtx *EvalContext) error {
	if len(evalCtx.AlertOkEvalMatches) == 0 {
		log.Infof("alert_ok_eval_matches is empty, skip syncMonitorResourceAlerts")
		return nil
	}
	// only sync resource not need notify
	matches := make([]monitor.EvalMatch, len(evalCtx.AlertOkEvalMatches))
	for i := range evalCtx.AlertOkEvalMatches {
		matches[i] = *evalCtx.AlertOkEvalMatches[i]
	}
	alertRule := evalCtx.Rule.RuleDescription
	input := &models.UpdateMonitorResourceAlertInput{
		AlertId:       evalCtx.Rule.Id,
		Matches:       matches,
		ResType:       alertRule[0].ResType,
		AlertState:    string(monitor.AlertStateOK),
		SendState:     monitor.SEND_STATE_SILENT,
		TriggerTime:   time.Now(),
		AlertRecordId: "",
	}
	if err := models.MonitorResourceManager.UpdateMonitorResourceAttachJoint(evalCtx.Ctx, evalCtx.UserCred, input); err != nil {
		return errors.Wrap(err, "UpdateMonitorResourceAttachJoint")
	}
	return nil
}

type NotifierPlugin struct {
	Type               string
	Factory            NotifierFactory
	ValidateCreateData func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error)
}

type NotificationConfig notifydrivers.NotificationConfig

// NotifierFactory is a signature for creating notifiers
type NotifierFactory func(config NotificationConfig) (Notifier, error)

func RegisterNotifier(plug *NotifierPlugin) {
	notifydrivers.RegisterNotifier(&notifydrivers.NotifierPlugin{
		Type: plug.Type,
		Factory: func(cfg notifydrivers.NotificationConfig) (notifydrivers.Notifier, error) {
			ret, err := plug.Factory(NotificationConfig(cfg))
			if err != nil {
				return nil, err
			}
			return ret.(notifydrivers.Notifier), nil
		},
		ValidateCreateData: plug.ValidateCreateData,
	})
}

// InitNotifier construct a new notifier
func InitNotifier(config NotificationConfig) (Notifier, error) {
	plug, err := notifydrivers.InitNotifier(notifydrivers.NotificationConfig(config))
	if err != nil {
		return nil, err
	}
	return plug.(Notifier), nil
}
