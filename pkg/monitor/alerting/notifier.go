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
	"fmt"
	"strconv"
	"strings"
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
	notifierStates, err := n.getNeededNotifiers(evalCtx.Rule.Notifications, evalCtx)
	if err != nil {
		return errors.Wrap(err, "failed to get alert notifiers")
	}

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
			return err
		}
	}
	return n.sendAndMarkAsComplete(evalCtx, state)
}

func (n *notificationService) sendAndMarkAsComplete(evalCtx *EvalContext, state *notifierState) error {
	notifier := state.notifier

	log.Debugf("Sending notification, type %s, id %s", notifier.GetType(), notifier.GetNotifierId())

	if err := notifier.Notify(evalCtx, state.state.GetParams()); err != nil {
		log.Errorf("failed to send notification %s: %v", notifier.GetNotifierId(), err)
		return err
	}

	if evalCtx.IsTestRun {
		return nil
	}

	return state.state.SetToCompleted()
}

func (n *notificationService) sendNotifications(evalCtx *EvalContext, states notifierStateSlice) error {
	for _, state := range states {
		if err := n.sendNotification(evalCtx, state); err != nil {
			log.Errorf("failed to send %s notification: %v", state.notifier.GetNotifierId(), err)
			if evalCtx.IsTestRun {
				return err
			}
		}
	}
	return nil
}

func (n *notificationService) getNeededNotifiers(nIds []string, evalCtx *EvalContext) (notifierStateSlice, error) {
	notis, err := models.NotificationManager.GetNotificationsWithDefault(nIds)
	if err != nil {
		return nil, err
	}

	var result notifierStateSlice
	shouldNotify := false
	for _, obj := range notis {
		not, err := InitNotifier(NotificationConfig{
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
	if shouldNotify {
		recordCreateInput := monitor.AlertRecordCreateInput{
			StandaloneResourceCreateInput: apis.StandaloneResourceCreateInput{
				GenerateName: evalCtx.Rule.Name,
			},
			AlertId:   evalCtx.Rule.Id,
			Level:     evalCtx.Rule.Level,
			State:     string(evalCtx.Rule.State),
			EvalData:  evalCtx.EvalMatches,
			AlertRule: newAlertRecordRule(evalCtx),
		}
		_, err = db.DoCreate(models.AlertRecordManager, evalCtx.Ctx, evalCtx.UserCred, jsonutils.NewDict(),
			jsonutils.Marshal(&recordCreateInput), evalCtx.UserCred)
		if err != nil {
			log.Errorf("create alert record err:%v", err)
		}
	}
	return result, nil
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

func newAlertRecordRule(evalCtx *EvalContext) monitor.AlertRecordRule {
	alertRule := monitor.AlertRecordRule{}
	if evalCtx.Rule.Frequency < 60 {
		alertRule.Period = fmt.Sprintf("%ds", evalCtx.Rule.Frequency)
	} else {
		alertRule.Period = fmt.Sprintf("%dm", evalCtx.Rule.Frequency/60)
	}
	ruleStr := evalCtx.Rule.Message
	ruleElementArr := strings.Split(ruleStr, " ")
	if len(ruleElementArr) == 3 {
		alertRule.Metric = ruleElementArr[0]
		alertRule.Comparator = ruleElementArr[1]
		alertRule.Threshold, _ = strconv.ParseFloat(ruleElementArr[2], 64)
	}
	if len(evalCtx.EvalMatches) != 0 {
		alertRule.MeasurementDesc = evalCtx.EvalMatches[0].MeasurementDesc
		alertRule.FieldDesc = evalCtx.EvalMatches[0].FieldDesc
	}
	return alertRule
}
