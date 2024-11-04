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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

// EvalContext is the context object for an alert evaluation.
type EvalContext struct {
	Firing             bool
	IsTestRun          bool
	IsDebug            bool
	EvalMatches        []*monitor.EvalMatch
	AlertOkEvalMatches []*monitor.EvalMatch
	Logs               []*monitor.ResultLogEntry
	Error              error
	ConditionEvals     string
	StartTime          time.Time
	EndTime            time.Time
	Rule               *Rule

	NoDataFound    bool
	PrevAlertState monitor.AlertStateType

	Ctx      context.Context
	UserCred mcclient.TokenCredential
}

// NewEvalContext is the EvalContext constructor.
func NewEvalContext(alertCtx context.Context, userCred mcclient.TokenCredential, rule *Rule) *EvalContext {
	return &EvalContext{
		Ctx:                alertCtx,
		UserCred:           userCred,
		StartTime:          time.Now(),
		Rule:               rule,
		EvalMatches:        make([]*monitor.EvalMatch, 0),
		AlertOkEvalMatches: make([]*monitor.EvalMatch, 0),
		PrevAlertState:     rule.State,
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
	return c.Rule.State != c.PrevAlertState || c.Rule.State == monitor.AlertStateAlerting
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

func (c *EvalContext) GetCallbackURLPrefix() string {
	config, err := modules.ServicesV3.GetSpecific(auth.GetAdminSession(c.Ctx, ""), "common", "config",
		jsonutils.NewDict())
	if err != nil {
		log.Errorf("GetCallbackURLPrefix err:%v", err)
		return ""
	}
	url, _ := config.GetString("config", "default", "api_server")
	defaultWebUri := "alertrecord"
	matchTag := map[string]string{}
	if c.Firing {
		matchTag = c.EvalMatches[0].Tags
	} else {
		matchTag = c.AlertOkEvalMatches[0].Tags
	}
	if uri, ok := matchTag["web_url"]; ok {
		defaultWebUri = uri
	}
	return fmt.Sprintf("%s/%s", url, defaultWebUri)
}

// GetNewState returns the new state from the alert rule evaluation.
func (c *EvalContext) GetNewState() monitor.AlertStateType {
	ns := getNewStateInternal(c)
	if ns != monitor.AlertStateAlerting || c.Rule.For == 0 {
		return ns
	}

	since := time.Since(c.Rule.LastStateChange)
	if c.PrevAlertState == monitor.AlertStatePending && since/time.Second >= c.Rule.For {
		return monitor.AlertStateAlerting
	}

	if c.Rule.For != 0 {
		log.Errorf("ruleName:%s,since:%d,for:%d", c.Rule.Name, since/time.Second, c.Rule.For)
	}
	if ns == monitor.AlertStateAlerting && since/time.Second >= c.Rule.For {
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

func (c *EvalContext) GetNotificationTemplateConfig(matches []*monitor.EvalMatch) monitor.NotificationTemplateConfig {
	desc := c.Rule.Message
	if len(c.Rule.TriggeredMessages) > 0 {
		desc = strings.Join(c.Rule.TriggeredMessages, " ")
	}
	if c.Error != nil {
		if desc != "" {
			desc += "\n"
		}
		desc += "Error: " + c.Error.Error()
	}
	tz, _ := time.LoadLocation(options.Options.TimeZone)
	return monitor.NotificationTemplateConfig{
		Title:        c.GetNotificationTitle(),
		Name:         c.Rule.Name,
		ResourceName: c.GetResourceNameOfMatches(matches),
		Matches:      matches,
		//Matches:      c.GetEvalMatches(),
		StartTime:   c.StartTime.In(tz).Format("2006-01-02 15:04:05"),
		EndTime:     c.EndTime.In(tz).Format("2006-01-02 15:04:05"),
		Description: desc,
		Level:       c.Rule.Level,
		NoDataFound: c.NoDataFound,
		WebUrl:      c.GetCallbackURLPrefix(),
	}
}

func (c *EvalContext) GetEvalMatches() []monitor.EvalMatch {
	ret := make([]monitor.EvalMatch, 0)
	matches := c.EvalMatches
	if !c.Firing {
		matches = c.AlertOkEvalMatches
	}
	for _, c := range matches {
		if _, ok := c.Tags[monitor.ALERT_RESOURCE_RECORD_SHIELD_KEY]; ok {
			continue
		}
		ret = append(ret, monitor.EvalMatch{
			Condition: c.Condition,
			Value:     c.Value,
			ValueStr:  c.ValueStr,
			Metric:    c.Metric,
			Tags:      c.Tags,
		})
	}
	return ret
}

func (c *EvalContext) GetResourceNameOfMatches(matches []*monitor.EvalMatch) string {
	names := strings.Builder{}
	for i, match := range matches {
		if name, ok := match.Tags["name"]; ok {
			names.WriteString(fmt.Sprintf("%s.%s(%s)", name, match.Metric, match.ValueStr))
			if i < len(matches)-1 {
				names.WriteString(", ")
			}
		}
	}
	return names.String()
}

func (c *EvalContext) GetRecoveredMatches() []*monitor.EvalMatch {
	ret := make([]*monitor.EvalMatch, 0)
	for i := range c.AlertOkEvalMatches {
		m := c.AlertOkEvalMatches[i]
		if m.IsRecovery {
			ret = append(ret, m)
		}
	}
	return ret
}

func (c *EvalContext) HasRecoveredMatches() bool {
	return len(c.GetRecoveredMatches()) != 0
}
