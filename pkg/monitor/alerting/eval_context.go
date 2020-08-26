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
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// EvalContext is the context object for an alert evaluation.
type EvalContext struct {
	Firing         bool
	IsTestRun      bool
	IsDebug        bool
	EvalMatches    []*monitor.EvalMatch
	Logs           []*monitor.ResultLogEntry
	Error          error
	ConditionEvals string
	StartTime      time.Time
	EndTime        time.Time
	Rule           *Rule

	NoDataFound    bool
	PrevAlertState monitor.AlertStateType

	Ctx      context.Context
	UserCred mcclient.TokenCredential
}

// NewEvalContext is the EvalContext constructor.
func NewEvalContext(alertCtx context.Context, userCred mcclient.TokenCredential, rule *Rule) *EvalContext {
	return &EvalContext{
		Ctx:            alertCtx,
		UserCred:       userCred,
		StartTime:      time.Now(),
		Rule:           rule,
		EvalMatches:    make([]*monitor.EvalMatch, 0),
		PrevAlertState: rule.State,
	}
}

// SateDescription contains visual information about the alert state.
type StateDescription struct {
	//Color string
	Text string
	Data string
}

// GetStateModel returns the `StateDescription` based on current state.
func (c *EvalContext) GetStateModel() *StateDescription {
	switch c.Rule.State {
	case monitor.AlertStateOK:
		return &StateDescription{
			Text: "OK",
		}
	case monitor.AlertStateNoData:
		return &StateDescription{
			Text: "No Data",
		}
	case monitor.AlertStateAlerting:
		return &StateDescription{
			Text: "Alerting",
		}
	case monitor.AlertStateUnknown:
		return &StateDescription{
			Text: "Unknown",
		}
	default:
		panic(fmt.Sprintf("Unknown rule state %q for alert %s", c.Rule.State, c.Rule.Name))
	}
}

func (c *EvalContext) shouldUpdateAlertState() bool {
	return c.Rule.State != c.PrevAlertState
}

// GetDurationMs returns the duration of the alert evaluation.
func (c *EvalContext) GetDurationMs() float64 {
	return float64(c.EndTime.Nanosecond()-c.StartTime.Nanosecond()) / float64(1000000)
}

func (c *EvalContext) GetRuleTitle() string {
	rule := c.Rule
	if rule.Title != "" {
		return rule.Title
	}
	return rule.Name
}

// GetNotificationTitle returns the title of the alert rule including alert state.
func (c *EvalContext) GetNotificationTitle() string {
	return "[" + c.GetStateModel().Text + "] " + c.GetRuleTitle()
}

// GetNewState returns the new state from the alert rule evaluation.
func (c *EvalContext) GetNewState() monitor.AlertStateType {
	ns := getNewStateInternal(c)
	if ns != monitor.AlertStateAlerting || c.Rule.For == 0 {
		return ns
	}

	since := time.Since(c.Rule.LastStateChange)
	if c.PrevAlertState == monitor.AlertStatePending && since > c.Rule.For {
		return monitor.AlertStateAlerting
	}

	if c.PrevAlertState == monitor.AlertStateAlerting {
		return monitor.AlertStateAlerting
	}

	return monitor.AlertStatePending
}

func getNewStateInternal(c *EvalContext) monitor.AlertStateType {
	if c.Error != nil {
		log.Errorf("Alert Rule Result Error, ruleId: %s, name: %s, error: %v, changing state to %v",
			c.Rule.Id,
			c.Rule.Name,
			c.Error,
			c.Rule.ExecutionErrorState.ToAlertState())

		if c.Rule.ExecutionErrorState == monitor.ExecutionErrorKeepState {
			return c.PrevAlertState
		}
		return c.Rule.ExecutionErrorState.ToAlertState()
	}

	if c.Firing {
		return monitor.AlertStateAlerting
	}

	if c.NoDataFound {
		log.Infof("Alert Rule returned no data, ruleId: %s, name: %s, changing state to %v",
			c.Rule.Id,
			c.Rule.Name,
			c.Rule.NoDataState.ToAlertState())

		if c.Rule.NoDataState == monitor.NoDataKeepState {
			return c.PrevAlertState
		}
		return c.Rule.NoDataState.ToAlertState()
	}

	return monitor.AlertStateOK
}

func (c *EvalContext) GetNotificationTemplateConfig() monitor.NotificationTemplateConfig {
	desc := c.Rule.Message
	if c.Error != nil {
		if desc != "" {
			desc += "\n"
		}
		desc += "Error: " + c.Error.Error()
	}
	return monitor.NotificationTemplateConfig{
		Title:       c.GetNotificationTitle(),
		Name:        c.Rule.Name,
		Matches:     c.GetEvalMatches(),
		StartTime:   c.StartTime.Format(time.RFC3339),
		EndTime:     c.EndTime.Format(time.RFC3339),
		Description: desc,
		Level:       c.Rule.Level,
		NoDataFound: c.NoDataFound,
	}
}

func (c *EvalContext) GetEvalMatches() []monitor.EvalMatch {
	ret := make([]monitor.EvalMatch, 0)
	for _, c := range c.EvalMatches {
		ret = append(ret, monitor.EvalMatch{
			Condition: c.Condition,
			Value:     c.Value,
			Metric:    c.Metric,
			Tags:      c.Tags,
		})
	}
	return ret
}
