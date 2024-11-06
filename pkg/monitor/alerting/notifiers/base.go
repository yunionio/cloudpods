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

package notifiers

import (
	"context"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

// NotifierBase is the base implentation of a notifier
type NotifierBase struct {
	Ctx                   context.Context
	Name                  string
	Type                  string
	Id                    string
	IsDefault             bool
	SendReminder          bool
	DisableResolveMessage bool
	Frequency             time.Duration
}

// NewNotifierBase returns a new NotifierBase
func NewNotifierBase(config alerting.NotificationConfig) NotifierBase {
	return NotifierBase{
		Ctx:  config.Ctx,
		Id:   config.Id,
		Name: config.Name,
		// IsDefault: config.IsDefault,
		Type:                  config.Type,
		SendReminder:          config.SendReminder,
		DisableResolveMessage: config.DisableResolveMessage,
		Frequency:             config.Frequency,
	}
}

// ShouldNotify checks this evaluation should send an alert notification
func (n *NotifierBase) ShouldNotify(_ context.Context, evalCtx *alerting.EvalContext, state *models.SAlertnotification) bool {
	prevState := evalCtx.PrevAlertState
	newState := evalCtx.Rule.State

	if evalCtx.HasRecoveredMatches() {
		return true
	}

	// Do not notify if alert state is no_data
	if newState == monitor.AlertStateNoData {
		return false
	}

	if newState == monitor.AlertStatePending {
		return false
	}

	if newState == monitor.AlertStateAlerting {
		if prevState == monitor.AlertStateOK {
			return true
		}
		send, err := state.ShouldSendNotification()
		if err != nil {
			log.Errorf("Alertnotification ShouldSendNotification exec err:%v", err)
			return false
		}
		return send
	}

	// Only notify on state change
	if prevState == newState && !n.SendReminder {
		return false
	}

	if prevState == newState && n.SendReminder {
		// Do not notify if interval has not elapsed
		lastNotify := state.UpdatedAt
		// if state.UpdatedAt != 0 && lastNotify.Add(n.Frequency).After(time.Now()) {
		if lastNotify.Add(n.Frequency).After(time.Now()) {
			return false
		}

		// Do not notify if alert state is OK or pending even on repeated notify
		if newState == monitor.AlertStateOK || newState == monitor.AlertStatePending {
			return false
		}
	}

	okOrPending := newState == monitor.AlertStatePending || newState == monitor.AlertStateOK

	// Do not notify when new state is ok/pending when previous is unknown
	if prevState == monitor.AlertStateUnknown && okOrPending {
		return false
	}

	// Do not notify when we become OK from pending
	if prevState == monitor.AlertStatePending && okOrPending {
		return false
	}

	// Do not notify when we OK -> Pending
	if prevState == monitor.AlertStateOK && okOrPending {
		return false
	}

	// Do not notify if state pending and it have been updated last minute
	if state.GetState() == monitor.AlertNotificationStatePending {
		lastUpdated := state.UpdatedAt
		if lastUpdated.Add(1 * time.Minute).After(time.Now()) {
			return false
		}
	}

	// Do not notify when state is OK if DisableResolveMessage is set to true
	if newState == monitor.AlertStateOK && n.DisableResolveMessage {
		return false
	}

	return true
}

// GetType returns the notifier type.
func (n *NotifierBase) GetType() string {
	return n.Type
}

// GetNotifierId returns the notifier `uid`.
func (n *NotifierBase) GetNotifierId() string {
	return n.Id
}

// GetIsDefault returns true if the notifiers should
// be used for all alerts.
/*func (n *NotifierBase) GetIsDefault() bool {
	return n.IsDeault
}*/

// GetSendReminder returns true if reminders should be sent.
func (n *NotifierBase) GetSendReminder() bool {
	return n.SendReminder
}

// GetDisableResolveMessage returns true if ok alert notifications
// should be skipped.
func (n *NotifierBase) GetDisableResolveMessage() bool {
	return n.DisableResolveMessage
}

// GetFrequency returns the frequency for how often
// alerts should be evaluated.
func (n *NotifierBase) GetFrequency() time.Duration {
	return n.Frequency
}
