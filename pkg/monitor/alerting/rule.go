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
	"regexp"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/validators"
)

var (
	// ErrFrequencyCannotBeZeroOrLess frequency cannot be below zero
	ErrFrequencyCannotBeZeroOrLess = errors.Error(`"evaluate every" cannot be zero or below`)

	// ErrFrequencyCouldNotBeParsed frequency cannot be parsed
	ErrFrequencyCouldNotBeParsed = errors.Error(`"evaluate every" field could not be parsed`)
)

func init() {
	models.AlertManager.SetTester(NewAlertRuleTester())
}

// Rule is the in-memory version of an alert rule.
type Rule struct {
	Id                  string
	Frequency           int64
	Title               string
	Name                string
	Message             string
	LastStateChange     time.Time
	For                 time.Duration
	NoDataState         monitor.NoDataOption
	ExecutionErrorState monitor.ExecutionErrorOption
	State               monitor.AlertStateType
	Conditions          []Condition
	Notifications       []string
	// AlertRuleTags       []*models.AlertRuleTag
	Level string

	StateChanges int
}

var (
	valueFormatRegex = regexp.MustCompile(`^\d+`)
	unitFormatRegex  = regexp.MustCompile(`\w{1}$`)
)

var unitMultiplier = map[string]int{
	"s": 1,
	"m": 60,
	"h": 3600,
	"d": 86400,
}

func getTimeDurationStringToSeconds(str string) (int64, error) {
	multiplier := 1

	matches := valueFormatRegex.FindAllString(str, 1)

	if len(matches) <= 0 {
		return 0, ErrFrequencyCouldNotBeParsed
	}

	value, err := strconv.Atoi(matches[0])
	if err != nil {
		return 0, err
	}

	if value == 0 {
		return 0, ErrFrequencyCannotBeZeroOrLess
	}

	unit := unitFormatRegex.FindAllString(str, 1)[0]

	if val, ok := unitMultiplier[unit]; ok {
		multiplier = val
	}

	return int64(value * multiplier), nil
}

// NewRuleFromDBAlert maps an db version of
// alert to an in-memory version
func NewRuleFromDBAlert(ruleDef *models.SAlert) (*Rule, error) {
	model := &Rule{}
	model.Id = ruleDef.Id
	model.Title = ruleDef.GetTitle()
	model.Name = ruleDef.Name
	model.Message = ruleDef.Message
	model.State = monitor.AlertStateType(ruleDef.State)
	model.LastStateChange = ruleDef.LastStateChange
	model.For = time.Duration(ruleDef.For)
	model.NoDataState = monitor.NoDataOption(ruleDef.NoDataState)
	model.ExecutionErrorState = monitor.ExecutionErrorOption(ruleDef.ExecutionErrorState)
	model.StateChanges = ruleDef.StateChanges

	model.Frequency = ruleDef.Frequency
	// frequency cannot be zero since that would not execute the alert rule.
	// so we fallback to 60 seconds if `Frequency` is missing
	if model.Frequency == 0 {
		model.Frequency = 60
	}

	settings, err := ruleDef.GetSettings()
	if err != nil {
		return nil, err
	}

	model.Level = ruleDef.Level
	nIds := []string{}
	notis, err := ruleDef.GetNotifications()
	if err != nil {
		return nil, err
	}
	for _, n := range notis {
		nIds = append(nIds, n.NotificationId)
	}
	model.Notifications = nIds
	// model.AlertRuleTags = ruleDef.GetTagsFromSettings()

	for index, condition := range settings.Conditions {
		condType := condition.Type
		factory, exist := conditionFactories[condType]
		if !exist {
			return nil, errors.Wrapf(validators.ErrAlertConditionUnknown, "condition type %s", condType)
		}
		queryCond, err := factory(&condition, index)
		if err != nil {
			return nil, errors.Wrapf(err, "construct query condition %s", jsonutils.Marshal(condition))
		}
		model.Conditions = append(model.Conditions, queryCond)
	}

	if len(model.Conditions) == 0 {
		return nil, validators.ErrAlertConditionEmpty
	}
	return model, nil
}

type AlertRuleTester struct{}

func NewAlertRuleTester() models.AlertTestRunner {
	return new(AlertRuleTester)
}

func (_ AlertRuleTester) DoTest(ruleDef *models.SAlert, userCred mcclient.TokenCredential, input monitor.AlertTestRunInput) (*monitor.AlertTestRunOutput, error) {
	rule, err := NewRuleFromDBAlert(ruleDef)
	if err != nil {
		return nil, err
	}
	handler := NewEvalHandler()

	ctx := NewEvalContext(context.Background(), userCred, rule)
	ctx.IsTestRun = true
	ctx.IsDebug = input.IsDebug

	handler.Eval(ctx)

	return ctx.ToTestRunResult(), nil
}

func (ctx *EvalContext) ToTestRunResult() *monitor.AlertTestRunOutput {
	return &monitor.AlertTestRunOutput{
		Firing:         ctx.Firing,
		EvalMatches:    ctx.EvalMatches,
		Logs:           ctx.Logs,
		Error:          ctx.Error,
		ConditionEvals: ctx.ConditionEvals,
		StartTime:      ctx.StartTime,
		EndTime:        ctx.EndTime,
	}
}

// ConditionFactory is the function signature for creating `Conditions`
type ConditionFactory func(model *monitor.AlertCondition, index int) (Condition, error)

var conditionFactories = make(map[string]ConditionFactory)

// RegisterCondition adds support for alerting conditions.
func RegisterCondition(typeName string, factory ConditionFactory) {
	conditionFactories[typeName] = factory
}
